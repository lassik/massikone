#! /usr/bin/env ruby
# coding: utf-8

require "date"
require "open3"

require "mustache"
require "omniauth"
require "omniauth-google-oauth2"
require "roda"
require "sequel"
require "zip"

require_relative "model"

FI_DATE = "%d.%m.%Y"
ISO_DATE = "%Y-%m-%d"

def fi_from_iso_date(str)
  return nil unless str and not str.empty?
  DateTime.strptime(str, ISO_DATE).strftime(FI_DATE)
end

def iso_from_fi_date(str)
  return nil unless str and not str.empty?
  DateTime.strptime(str, FI_DATE).strftime(ISO_DATE)
end

def slug(str)
  return "" if str.nil?
  str.downcase.gsub(/\s+/, '-').gsub(/[^\w-]/, '').gsub(/^-/, '').gsub(/-$/, '').gsub(/--+/, '')
end

def load_account_tree
  items = {}
  File.foreach("chart-of-accounts.txt").drop(1).each do |line|
    fields = line.chomp.split(";")
    raise "#{fields.length}" unless fields.length == 4
    rowtype = fields[0]
    raise "#{rowtype}" unless ["A", "H"].include?(rowtype)
    rownum = fields[1].to_i
    title = fields[2]
    level = if rowtype == "H" then fields[3].to_i else nil end
    key = "#{rownum}-#{level or 9999}"
    raise if items.has_key?(key)
    items[key] = if level then "-" * (level+1) + " " else "---- " end + title
  end
  list = []
  items.sort.each do |k, v|
    list.push({:account_id => k, :account_title => v})
  end
  list
end

def amount_from_cents(cents)
  return "" if cents.nil?
  euros, cents = cents.divmod(100)
  sprintf("%d,%02d", euros, cents)
end

# We don't need the compexity of tilt.
def mustache(template, opts={})
  view = Mustache.new
  view.template_file = "views/#{template}.mustache"
  opts.each_pair {|k,v| view[k.to_sym] = v}
  view[:organization] = ENV["ORGANIZATION"]
  view[:app_title] = "#{view[:organization]} Massikone"
  view.render
end

class Massikone < Roda

  plugin :all_verbs
  plugin :halt
  plugin :json
  plugin :not_allowed
  plugin :pass
  plugin :public
  plugin :sinatra_helpers
  plugin :status_handler

  def our_error_page
    Rack::Utils::HTTP_STATUS_CODES[response.status]
  end

  #plugin :error_handler do |e| our_error_page end
  status_handler 404 do our_error_page end
  status_handler 405 do our_error_page end

  use Rack::CommonLogger
  use Rack::Session::Cookie, :secret => ENV.fetch("SESSION_SECRET")
  use OmniAuth::Builder do
    provider :google_oauth2,
             ENV.fetch("GOOGLE_CLIENT_ID"),
             ENV.fetch("GOOGLE_CLIENT_SECRET"),
             {:scope => "email,profile"}
  end
  def omniauth_providers
    [:google_oauth2]
  end

  @@accounts = load_account_tree

  route do |r|

    r.public

    r.on "auth" do

      r.on [":provider", :all=>omniauth_providers] do |provider|
        r.is ["callback", {:method => ["get", "post"]}] do
          auth = request.env["omniauth.auth"]
          r.halt(403, "Forbidden") unless auth and auth["provider"] == provider
          uid_field = "user_id_#{provider}"
          uid = auth["uid"]
          email = auth["info"]["email"]
          full_name = auth["info"]["name"]
          missing = []
          missing.push("käyttäjä-ID") unless uid
          missing.push("nimi") unless full_name
          missing.push("sähköposti") unless email
          unless missing.empty?
            r.halt(403, "Seuraavia tietoja ei saatu: #{missing.join(", ")}")
          end
          user = Model::put_user(uid_field, uid, email, full_name)
          session[:user_id] = user[:user_id]
          r.redirect "/"
        end
      end

      r.get "failure" do
        # r[:message]
        r.redirect "/"
      end

    end

    r.post "logout" do
      session[:user_id] = nil
      r.redirect "/"
    end

    current_user = if session[:user_id]
                     Model::DB[:users].where(:user_id => session[:user_id]).first
                   else
                     nil
                   end

    users = Model.get_users()

    admin_data = if current_user and current_user[:is_admin]
                   {
                     :users => users
                   }
                 else
                   nil
                 end

    r.on "api" do

      r.halt(403, "Forbidden") unless current_user

      r.on "userimage" do

        # SECURITY NOTE: Users can view each other's images if they somehow
        # know the filehash.

        r.on "rotated/:image_id" do |image_id|
          r.is do
            r.get do
              response["Content-Type"] = "text/plain"
              Model.rotate_image(image_id)
            end
          end
        end

        r.get :image_id do |image_id|
          # TODO http header, esp. caching
          r.pass unless Model.valid_image_id?(image_id)
          Model.fetch_image_data(image_id)
        end

        r.is do
          r.post do
            response["Content-Type"] = "text/plain"
            Model.store_image_file(r[:file][:tempfile])
          end
        end

      end

      r.on "tags" do

        r.get do
          Model.get_available_tags()
        end

        r.put do
          Model::DB.delete("delete from tags")
          r.body.each do |tag|
            Model::DB.insert("insert into tags (tag) values (?)", tag)
          end
        end

      end

    end

    r.root do
      r.pass if current_user
      mustache :login
    end

    r.root do
      bills, all_tags = Model.fetch_bills_and_all_tags current_user
      mustache "bills",
               :current_user => current_user,
               :admin => admin_data,
               :tags => all_tags,
               :bills => {:bills => bills} if bills
    end

    unless current_user
      r.redirect("/") if r.get?
      r.halt(403, "Forbidden")
    end

    r.on "bill" do

      r.is ":bill_id" do |bill_id|

        r.get do
          bill = Model.fetch_bill(bill_id)
          u = users.find do |u| u[:user_id] == bill[:paid_user_id] end
          u[:is_paid_user] = true if u
          u = users.find do |u| u[:user_id] == bill[:closed_user_id] end
          u[:is_closed_user] = true if u
          r.halt(404, "No such bill") unless bill
          mustache :bill,
                   :admin => admin_data,
                   :current_user => current_user,
                   :accounts => @@accounts,
                   :tags => [{tag: "ruoka", active: false}],
                   :bill => bill
        end

        r.post do
          bill = Model::DB.fetch("select * from bills where bill_id = :bill_id",
                          :bill_id=>bill_id).first
          r.halt(404, "No such bill") unless bill
          Model.update_bill! bill_id, r, current_user
          r.redirect "/bill/#{bill_id}"
        end

      end

      r.is do

        r.get do
          mustache :bill,
                   :current_user => current_user,
                   :admin => admin_data,
                   :accounts => @@accounts
        end

        r.post do
          bill_id = Model::DB[:bills].insert(
            :created_date => DateTime.now.strftime("%Y-%m-%d"))
          Model.update_bill! bill_id, r, current_user
          r.redirect "/bill/#{bill_id}"
        end

      end

    end

    r.halt(403, "Forbidden") unless admin_data

    r.on "report" do

      r.get "chart-of-accounts" do
        mustache "report/chart-of-accounts",
                 :accounts => @@accounts
      end

      r.get "massikone.ofx" do
        bills, all_tags = Model.fetch_bills_and_all_tags current_user
        response["Content-Type"] = "text/xml"
        mustache "report/massikone.ofx",
                 :bills => bills
      end

      r.get "massikone.zip" do
        bills = Model::DB.fetch("select bill_id, image_id, description, tags from bills"+
                         " order by bill_id").all
        zipfilepath = "/tmp/massikone.zip"  # TODO use mktemp
        FileUtils.rm_f(zipfilepath)
        missing = []
        Zip::File.open(zipfilepath, Zip::File::CREATE) do |zipfile|
          bills.each do |bill|
            if bill[:image_id] and bill[:image_id] != ""
              imginzip = sprintf("massikone/tosite-%04d-%s%s",
                                 bill[:bill_id],
                                 slug(bill[:description] || bill[:tags]),
                                 File.extname(bill[:image_id]))
              image = Model::DB.fetch("select image_data from images"+
                               " where image_id = ?", bill[:image_id]).first
              zipfile.get_output_stream(imginzip) do |output|
                output.write image[:image_data]
              end
            else
              missing.push("##{bill[:bill_id]}")
            end
          end
          if not missing.empty?
            zipfile.get_output_stream("massikone/puuttuvat.txt") do |out|
              out.write(missing.join("\r\n"))
            end
          end
        end
        r.send_file zipfilepath
      end

    end

  end

end
