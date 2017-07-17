# coding: utf-8

require 'date'
require 'open3'

require 'prawn'
require 'prawn/table'
require 'zip'

require_relative 'model'
require_relative 'util'

module Reports
  def self.chart_of_accounts_pdf
    # TODO: Exclude accounts (and headings) that haven't been used this period.
    accounts = Model::Accounts
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

  def self.add_bill_images_to_zip(zipfile, subdir)
    missing = []
    bills = Model.get_bills_for_report
    bills.each do |bill|
      if bill[:image_data]
        imginzip = format('%s/tosite-%04d-%s%s',
                          subdir, bill[:bill_id],
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
      zipfile.get_output_stream(format('%s/puuttuvat.txt', subdir)) do |out|
        out.write(missing.join("\r\n"))
      end
    end
  end

  def self.bill_images_zip
    zipfilepath = '/tmp/massikone.zip' # TODO: use mktemp
    FileUtils.rm_f(zipfilepath)
    Zip::File.open(zipfilepath, Zip::File::CREATE) do |zipfile|
      add_bill_images_to_zip(zipfile, 'massikone')
    end
    zipfilepath
  end
end
