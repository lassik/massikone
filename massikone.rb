#! /usr/bin/env ruby
# coding: utf-8

require 'mustache'
require 'omniauth'
require 'omniauth-google-oauth2'
require 'roda'

require_relative 'model'
require_relative 'reports'
require_relative 'util'

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

  # We don't need the compexity of tilt.
  def mustache(template, opts = {})
    view = Mustache.new
    view.template_file = "views/#{template}.mustache"
    opts.each_pair { |k, v| view[k.to_sym] = v }
    prefs = @model.get_preferences
    view[:organization] = prefs['org_short_name']
    view[:app_title] = "#{view[:organization]} Massikone"
    view.render
  end

  route do |r|
    r.public

    r.on 'auth' do
      r.on [':provider', all: omniauth_providers] do |provider|
        r.is ['callback', {method: %w[get post]}] do
          auth = request.env['omniauth.auth']
          r.halt(403, 'Forbidden') unless auth && auth['provider'] == provider
          Model.new(nil) do |model|
            user = model.put_user provider: provider,
                                  uid: auth['uid'],
                                  email: auth['info']['email'],
                                  full_name: auth['info']['name']
            session[:user_id] = user[:user_id]
          end
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

    Model.new(session[:user_id]) do |model|
      users = model.get_users

      admin_data = if model.user && model.user[:is_admin]
                     {
                       users: users,
                     }
                   end

      @model = model

      r.on 'api' do
        r.halt(403, 'Forbidden') unless model.user

        r.on 'userimage' do
          # SECURITY NOTE: Users can view each other's images if they somehow
          # know the filehash.

          r.on 'rotated/:image_id' do |image_id|
            r.is do
              r.get do
                response['Content-Type'] = 'text/plain'
                model.rotate_image(image_id)
              end
            end
          end

          r.get :image_id do |image_id|
            # TODO: http header, esp. caching
            r.pass unless model.valid_image_id?(image_id)
            model.get_image_data(image_id)
          end

          r.is do
            r.post do
              response['Content-Type'] = 'text/plain'
              model.store_image_file(r[:file][:tempfile])
            end
          end
        end

        r.on 'preferences' do
          r.put do
            model.put_preferences(r.body)
            ''
          end
        end

        r.on 'tags' do
          r.get do
            model.get_available_tags
          end

          r.put do
            model.put_available_tags(r.body)
          end
        end

        r.on 'compare' do
          r.get do
            model.get_bills_for_compare
          end
        end
      end

      r.root do
        r.pass if model.user
        mustache :login
      end

      r.root do
        bills, all_tags = model.get_bills_and_all_tags
        if bills
          mustache 'bills',
                   current_user: model.user,
                   admin: admin_data,
                   tags: all_tags,
                   bills: {bills: bills}
        end
      end

      unless model.user
        r.redirect('/') if r.get?
        r.halt(403, 'Forbidden')
      end

      r.on 'bill' do
        r.is ':bill_id' do |bill_id|
          r.get do
            bill = model.get_bill(bill_id)
            u = users.find { |u| u[:user_id] == bill[:paid_user_id] }
            u[:is_paid_user] = true if u
            u = users.find { |u| u[:user_id] == bill[:closed_user_id] }
            u[:is_closed_user] = true if u
            r.halt(404, 'No such bill') unless bill
            accts = model.get_accounts
            credit_accounts = accts.map do |acct|
              acct = acct.dup
              acct[:selected] = (acct[:account_id] && (acct[:account_id] == bill[:credit_account_id]))
              acct
            end
            debit_accounts = accts.map do |acct|
              acct = acct.dup
              acct[:selected] = (acct[:account_id] && (acct[:account_id] == bill[:debit_account_id]))
              acct
            end
            mustache(:bill,
                     admin: admin_data,
                     current_user: model.user,
                     tags: [{tag: 'ruoka', active: false}],
                     bill: bill,
                     credit_accounts: credit_accounts,
                     debit_accounts: debit_accounts)
          end

          r.post do
            model.put_bill(bill_id, r)
            r.redirect "/bill/#{bill_id}"
          end
        end

        r.is do
          r.get do
            accounts = model.get_accounts
            mustache :bill,
                     current_user: model.user,
                     admin: admin_data,
                     credit_accounts: accounts,
                     debit_accounts: accounts
          end

          r.post do
            bill = model.post_bill(r)
            r.redirect "/bill/#{bill[:bill_id]}"
          end
        end
      end

      r.halt(403, 'Forbidden') unless admin_data

      r.on 'compare' do
        r.get do
          mustache :compare,
                   current_user: model.user,
                   admin: admin_data,
                   preferences: model.get_preferences
        end
      end

      r.on 'preferences' do
        r.get do
          mustache :preferences,
                   current_user: model.user,
                   admin: admin_data,
                   preferences: model.get_preferences
        end
      end

      r.on 'report' do
        r.get 'general-journal' do
          pdf_data, filename = Reports.general_journal_pdf(model)
          response['Content-Type'] = 'application/pdf'
          response['Content-Disposition'] = "inline; filename=\"#{filename}\""
          pdf_data
        end

        r.get 'chart-of-accounts' do
          pdf_data, filename = Reports.chart_of_accounts_pdf(model)
          response['Content-Type'] = 'application/pdf'
          response['Content-Disposition'] = "inline; filename=\"#{filename}\""
          pdf_data
        end

        r.get 'massikone.zip' do
          filename = Reports.full_statement_zip(model)
          response['Content-Disposition'] = "attachment; filename=\"#{File.basename(filename)}\""
          r.send_file filename
        end
      end
    end
  end
end
