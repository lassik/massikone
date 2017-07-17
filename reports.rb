# coding: utf-8

require 'date'
require 'open3'

require 'prawn'
require 'prawn/table'

require_relative 'model'
require_relative 'util'

module Reports
  def self.chart_of_accounts_pdf(accounts:)
    # TODO: Exclude accounts (and headings) that haven't been used this period.
    prefs = Model.get_preferences
    org_short_name = prefs['org_short_name']
    org_full_name = prefs['org_full_name']
    year = 2017 # TODO
    filename = "#{org_short_name}-#{year}-tilikartta.pdf"
    pdf_data = Prawn::Document.new do
      font 'Helvetica'
      text org_full_name, size: 18
      text "Tilikartta #{year}", size: 14
      move_down 10
      stroke_horizontal_rule
      move_down 10
      rows = accounts.map do |acct|
        [acct[:account_id], acct[:title]]
      end
      accounts.each do |acct|
        if acct[:account_id]
          indent(20 * 4) do
            text "#{acct[:account_id]} #{acct[:title]}"
          end
        else
          indent(20 * (acct[:htag_level] - 2)) do
            text acct[:title], size: 12, style: :bold
          end
        end
      end
      number_pages 'Sivu <page>/<total>',
                   at: [bounds.right - 150, 0],
                   width: 150,
                   align: :right
    end.render
    [pdf_data, filename]
  end
end
