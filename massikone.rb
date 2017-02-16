#! /usr/bin/env ruby
# coding: utf-8

require "date"
require "open3"

require "mustache"
require "omniauth"
require "omniauth-google-oauth2"
require "roda"
require "sequel"
require "sqlite3"
require "zip"

USERIMAGE_DIR = ENV.fetch("USERIMAGE_DIR")

DB = Sequel.connect(ENV.fetch("DATABASE_URL"))

if false

  DB.create_table :users do
    String  :email
    String  :full_name
    String  :iban
    Boolean :is_admin
    String  :user_id_google_oauth2
  end

  DB.create_table :bills do
    primary_key :bill_id
    String  :bill_type  # "own" | "orgbank" | "car"
    String  :image_id
    String  :tags
    String  :description
    Integer :unit_count
    Integer :unit_cost_cents
    String  :paid_date
    Integer :paid_user_id
    String  :reimbursed_date
    Integer :reimbursed_user_id
    String  :closed_date
    Integer :closed_user_id
    String  :created_date
  end

  DB.create_table :history do
    Integer :bill_id
    String :timestamp
    String :operation
    String :done_by
    String
  end

end

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
  str.downcase.gsub(/\s+/, '-').gsub(/[^\w-]/, '').gsub(/^-/, '').gsub('/-$', '')
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

VALID_PAID_TYPES = ["car", "card", "ebank", "self"]

def valid_paid_type(x)
  return nil unless x
  raise unless VALID_PAID_TYPES.include?(x)
  x
end

def valid_closed_type(x)
  return nil unless x
  raise unless ["reimbursed", "denied"].include?(x)
  x
end

def valid_user_id(x)
  return nil unless x
  x = x.to_i
  raise unless x >= 1
  x
end

def valid_nonneg_integer(x)
  x = x.to_i
  raise unless x >= 0
  x
end

def valid_image_id(x)
  return nil unless x and not x.empty?
  raise unless valid_imgbasename?(x)
  x
end

def valid_tags(x)
  return nil unless x
  raise "Invalid tags: #{x.inspect}" unless x =~ /^[ a-zA-Z0-9]*$/
  x.split(" ").sort.uniq.join(" ")
end

def valid_imgbasename?(imgbasename)
  not not /^[0-9a-f]{40}\.(jpeg|png)$/.match imgbasename
end

def store_image_from_tmpfile(tmpfilename, extension)
  filehash = Digest::SHA1.file(tmpfilename).hexdigest
  imgbasename = "#{filehash}#{extension}"
  imgfullpath = File.join(USERIMAGE_DIR, imgbasename)
  FileUtils.cp(tmpfilename, imgfullpath)  # TODO how reliable?
  imgbasename
end

def store_image(tmpfilename)
  # We don't need the complexity of minimagick et.al.
  #
  # Don't trust filename extension given by user. It is sometimes wrong even
  # when not malicious. Better trust ImageMagick to identify the file format.
  tmpfilename = File.absolute_path(tmpfilename)
  file_format, status = Open3.capture2(
                 "identify",
                 "-format", "%[m]",
                 tmpfilename)
  extension = nil
  if status.exitstatus == 0
    extension = case file_format
                when "JPEG" then ".jpeg"
                when "PNG"  then ".png"
                end
  end
  abort "Bad image format" unless extension  # TODO better error message for user. does roda have a good pre-made exception class we can use?
  message, status = Open3.capture2(
             "mogrify",
             "-strip",
             "-define", "png:include-chunk=none",
             "-resize", "900x900>",
             "-colorspace", "Gray",
             "-separate",
             "-average",
             tmpfilename)
  puts "Message from mogrify is #{message}"
  store_image_from_tmpfile(tmpfilename, extension)
end

def rotate_image(oldbasename)
  raise unless valid_imgbasename?(oldbasename)
  oldfullpath = File.join(USERIMAGE_DIR, oldbasename)
  extension = File.extname(oldbasename)
  tmpfullpath = "/tmp/massikone.resize#{extension}"  # TODO use mktemp
  message, status = Open3.capture2(
             "convert",
             oldfullpath,
             "-rotate", "90",
             tmpfullpath)
  puts "Message from convert is #{message}"
  store_image_from_tmpfile(tmpfullpath, extension)
end

def fetch_bill(bill_id)
  bill = DB.fetch("select * from bills where bill_id = :bill_id", :bill_id => bill_id).first
  return nil unless bill
  bill[:paid_type] = valid_paid_type(bill[:paid_type])
  VALID_PAID_TYPES.each do |pt|
    bill["paid_type_#{pt}_checked".to_sym] =
      if bill[:paid_type] == pt then "checked" else "" end
  end
  bill[:paid_user] = DB.fetch("select * from users where user_id = :user_id", :user_id => bill[:paid_user_id]).first
  bill[:closed_user] = DB.fetch("select * from users where user_id = :user_id", :user_id => bill[:closed_user_id]).first
  bill[:paid_date_fi] = fi_from_iso_date(bill[:paid_date])
  bill[:closed_date_fi] = fi_from_iso_date(bill[:closed_date])
  bill[:tags] = if bill[:tags] then bill[:tags].split.sort.uniq else [] end
  bill
end

def fetch_bills_and_all_tags
  all_tags = []
  bills = DB.fetch("select bill_id, unit_count * unit_cost_cents as cents, paid_date, tags, description, pu.full_name as paid_user_full_name"+
                   " from bills"+
                   " left join users pu on pu.user_id = bills.paid_user_id"+
                   " order by bill_id").all
  bills.each do |bill|
    bill[:amount] = amount_from_cents(bill[:cents])
    bill[:tags] = []
    DB.fetch("select distinct tag from bill_tags where bill_id = ? order by tag", bill[:bill_id]).each do |relation|
      tag = {:tag => relation[:tag]}
      bill[:tags].push(tag)
      all_tags.push(tag)
    end
    bill[:tags].sort! {|a,b| a[:tag] <=> b[:tag]}
    #bill[:tags].uniq! {|a,b| a[:tag] <=> b[:tag]}
  end
  all_tags.sort! {|a,b| a[:tag] <=> b[:tag]}
  #all_tags.uniq! {|a,b| a[:tag] <=> b[:tag]}
  [bills, all_tags]
end

def update_bill!(bill_id, r)
  # TODO only admin can set certain fields
  bill = {}
  bill[:paid_date] = iso_from_fi_date(r[:paid_date_fi])
  bill[:closed_date] = iso_from_fi_date(r[:closed_date_fi])
  bill[:paid_type] = valid_paid_type(r[:paid_type])
  bill[:closed_type] = valid_closed_type(r[:closed_type])
  bill[:paid_user_id] = valid_user_id(r[:paid_user_id])
  bill[:closed_user_id] = valid_user_id(r[:closed_user_id])
  bill[:unit_count] = valid_nonneg_integer(r[:unit_count])
  bill[:unit_cost_cents] = valid_nonneg_integer(r[:unit_cost_cents])
  bill[:image_id] = valid_image_id(r[:image_id])
  bill[:tags] = valid_tags(r[:tags])
  bill[:description] = r[:description]
  DB[:bills].where(:bill_id=>bill_id).update(bill)
  bill_id
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
          puts "Login #{[uid_field, uid, email, full_name].inspect}"
          user = DB["select * from users where #{uid_field} = ?", uid].first
          unless user
            DB["insert into users (#{uid_field}) values (?)", uid].insert
            user = DB["select * from users where #{uid_field} = ?", uid].first
          end
          DB["update users set email = ?, full_name = ? where #{uid_field} = ?", email, full_name, uid].update
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
                     DB[:users].where(:user_id => session[:user_id]).first
                   else
                     nil
                   end

    users = DB.fetch("select user_id, full_name from users").all
    users.each do |user|
      words = user[:full_name].split.map {|w| w.capitalize}
      user[:full_name] = words.join(" ")
      user[:short_name] = if words.length >= 2 then
                            "#{words[0]} #{words[1][0]}"
                          else
                            user[:full_name]
                          end
    end
    users.sort! {|a,b| a[:full_name] <=> b[:full_name]}

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

        r.on "rotated/:imgbasename" do |imgbasename|
          r.is do
            r.get do
              response["Content-Type"] = "text/plain"
              rotate_image(imgbasename)
            end
          end
        end

        r.get :imgbasename do |imgbasename|
          r.pass unless valid_imgbasename?(imgbasename)
          r.send_file(File.join(USERIMAGE_DIR, imgbasename))
        end

        r.is do
          r.post do
            response["Content-Type"] = "text/plain"
            store_image(r[:file][:tempfile])
          end
        end

      end

      r.on "tags" do

        # NOTE: The tags list merely controls what tags users can choose from
        # when *adding new tags* to bills. A bill can have *old* tags that are
        # no longer in the tags list. This is intentional.

        r.get do
          DB.fetch("select distinct tag from tags order by tag").all
        end

        r.put do
          DB.delete("delete from tags")
          r.body.each do |tag|
            DB.insert("insert into tags (tag) values (?)", tag)
          end
        end

      end

    end

    r.root do
      r.pass if current_user
      mustache :login
    end

    r.root do
      bills, all_tags = fetch_bills_and_all_tags
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
          bill = fetch_bill(bill_id)
          u = users.find do |u| u[:user_id] == bill[:paid_user_id] end
          u[:is_paid_user] = true if u
          u = users.find do |u| u[:user_id] == bill[:closed_user_id] end
          u[:is_closed_user] = true if u
          r.halt(404, "No such bill") unless bill
          puts bill[:tags].inspect
          mustache :bill,
                   :admin => admin_data,
                   :current_user => current_user,
                   :accounts => @@accounts,
                   :tags => [{tag: "ruoka", active: false}],
                   :bill => bill
        end

        r.post do
          bill = DB.fetch("select * from bills where bill_id = :bill_id",
                          :bill_id=>bill_id).first
          r.halt(404, "No such bill") unless bill
          update_bill! bill_id, r
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
          bill_id = DB[:bills].insert(
            :created_date => DateTime.now.strftime("%Y-%m-%d"))
          update_bill! bill_id, r
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
        bills, all_tags = fetch_bills_and_all_tags
        response["Content-Type"] = "text/xml"
        mustache "report/massikone.ofx",
                 :bills => bills
      end

      r.get "massikone.zip" do
        bills = DB.fetch("select bill_id, image_id, description, tags from bills"+
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
              zipfile.add(imginzip,
                          File.join(USERIMAGE_DIR, bill[:image_id]))
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
