#! /usr/bin/env ruby
# coding: utf-8

require 'mustache'
require 'omniauth'
require 'omniauth-google-oauth2'
require 'roda'
require 'zip'

require_relative 'util'
require_relative 'model'

def load_account_tree
  # H;1011;Vastaavaa;0 -- Heading ; first account ID ; title ; level
  # A;1011;Perustamismenot;0 -- Account ; ID ; title ; flags
  # Sort by ID. Multiple headings (and one account) can share the same ID.
  # Sort those by level (assume all accounts are deeper than any heading).
  list = []
  File.foreach('chart-of-accounts.txt').drop(1).each do |line|
    fields = line.chomp.split(';')
    raise unless fields.length == 4
    row_type, account_id, title, last_field = fields
    account_id, last_field = account_id.to_i, last_field.to_i
    dash_level = { 'H' => 1 + last_field, 'A' => nil }[row_type]
    htag_level = { 'H' => [2 + last_field, 6].min, 'A' => nil }[row_type]
    sort_level = { 'H' => last_field, 'A' => 9 }[row_type]
    sort_key = 10 * account_id + sort_level
    prefix = (row_type == 'H') ? ('=' * dash_level) : account_id.to_s
    account_id = nil unless row_type == 'A'
    list.push(account_id: account_id, title: title,
              prefix: prefix, htag_level: htag_level, sort_key: sort_key)
  end
  list.sort! { |a, b| a[:sort_key] <=> b[:sort_key] }
  list
end

# We don't need the compexity of tilt.
def mustache(template, opts = {})
  view = Mustache.new
  view.template_file = "views/#{template}.mustache"
  opts.each_pair { |k, v| view[k.to_sym] = v }
  view[:organization] = ENV['ORGANIZATION']
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

  # plugin :error_handler do |e| our_error_page end
  status_handler 404 do our_error_page end
  status_handler 405 do our_error_page end

  use Rack::CommonLogger
  use Rack::Session::Cookie, secret: ENV.fetch('SESSION_SECRET')
  use OmniAuth::Builder do
    provider :google_oauth2,
             ENV.fetch('GOOGLE_CLIENT_ID'),
             ENV.fetch('GOOGLE_CLIENT_SECRET'),
             scope: 'email,profile'
  end
  def omniauth_providers
    [:google_oauth2]
  end

  @@accounts = load_account_tree

  route do |r|
    r.public

    r.on 'auth' do
      r.on [':provider', all: omniauth_providers] do |provider|
        r.is ['callback', { method: %w[get post] }] do
          auth = request.env['omniauth.auth']
          r.halt(403, 'Forbidden') unless auth && auth['provider'] == provider
          user = Model.put_user provider: provider,
                                uid: auth['uid'],
                                email: auth['info']['email'],
                                full_name: auth['info']['name']
          session[:user_id] = user[:user_id]
          r.redirect '/'
        end
      end

      r.get 'failure' do
        # r[:message]
        r.redirect '/'
      end
    end

    r.post 'logout' do
      session[:user_id] = nil
      r.redirect '/'
    end

    current_user = nil
    current_user = Model.get_user(session[:user_id]) if session[:user_id]

    users = Model.get_users

    admin_data = if current_user && current_user[:is_admin]
                   {
                     users: users
                   }
                 end

    r.on 'api' do
      r.halt(403, 'Forbidden') unless current_user

      r.on 'userimage' do
        # SECURITY NOTE: Users can view each other's images if they somehow
        # know the filehash.

        r.on 'rotated/:image_id' do |image_id|
          r.is do
            r.get do
              response['Content-Type'] = 'text/plain'
              Model.rotate_image(image_id)
            end
          end
        end

        r.get :image_id do |image_id|
          # TODO: http header, esp. caching
          r.pass unless Model.valid_image_id?(image_id)
          Model.get_image_data(image_id)
        end

        r.is do
          r.post do
            response['Content-Type'] = 'text/plain'
            Model.store_image_file(r[:file][:tempfile])
          end
        end
      end

      r.on 'tags' do
        r.get do
          Model.get_available_tags
        end

        r.put do
          Model.put_available_tags(r.body)
        end
      end
    end

    r.root do
      r.pass if current_user
      mustache :login
    end

    r.root do
      bills, all_tags = Model.get_bills_and_all_tags current_user
      if bills
        mustache 'bills',
                 current_user: current_user,
                 admin: admin_data,
                 tags: all_tags,
                 bills: { bills: bills }
      end
    end

    unless current_user
      r.redirect('/') if r.get?
      r.halt(403, 'Forbidden')
    end

    r.on 'bill' do
      r.is ':bill_id' do |bill_id|
        r.get do
          bill = Model.get_bill(bill_id)
          u = users.find { |u| u[:user_id] == bill[:paid_user_id] }
          u[:is_paid_user] = true if u
          u = users.find { |u| u[:user_id] == bill[:closed_user_id] }
          u[:is_closed_user] = true if u
          r.halt(404, 'No such bill') unless bill
          mustache :bill,
                   admin: admin_data,
                   current_user: current_user,
                   accounts: @@accounts,
                   tags: [{ tag: 'ruoka', active: false }],
                   bill: bill
        end

        r.post do
          Model.put_bill(bill_id, r, current_user)
          r.redirect "/bill/#{bill_id}"
        end
      end

      r.is do
        r.get do
          mustache :bill,
                   current_user: current_user,
                   admin: admin_data,
                   accounts: @@accounts
        end

        r.post do
          bill = Model.post_bill(r, current_user)
          r.redirect "/bill/#{bill[:bill_id]}"
        end
      end
    end

    r.halt(403, 'Forbidden') unless admin_data

    r.on 'report' do
      r.get 'chart-of-accounts' do
        mustache 'report/chart-of-accounts',
                 accounts: @@accounts
      end

      r.get 'massikone.ofx' do
        bills, all_tags = Model.get_bills_and_all_tags current_user
        response['Content-Type'] = 'text/xml'
        mustache 'report/massikone.ofx',
                 bills: bills
      end

      r.get 'massikone.zip' do
        bills = Model.get_bills_for_report
        zipfilepath = '/tmp/massikone.zip' # TODO: use mktemp
        FileUtils.rm_f(zipfilepath)
        missing = []
        Zip::File.open(zipfilepath, Zip::File::CREATE) do |zipfile|
          bills.each do |bill|
            if bill[:image_data]
              imginzip = sprintf('massikone/tosite-%04d-%s%s',
                                 bill[:bill_id],
                                 Util.slug(bill[:description] || bill[:tags]),
                                 File.extname(bill[:image_id]))
              zipfile.get_output_stream(imginzip) do |output|
                output.write bill[:image_data]
              end
            else
              missing.push("##{bill[:bill_id]}")
            end
          end
          unless missing.empty?
            zipfile.get_output_stream('massikone/puuttuvat.txt') do |out|
              out.write(missing.join("\r\n"))
            end
          end
        end
        r.send_file zipfilepath
      end
    end
  end
end
