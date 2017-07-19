# coding: utf-8

require 'date'
require 'open3'

require 'prawn'
require 'prawn/table'
require 'zip'

require_relative 'model'
require_relative 'util'

module Reports
  def self.general_journal_pdf
    bills = Model.get_bills_for_journal
    prefs = Model.get_preferences
    org_full_name = prefs['org_full_name']
    filename = generate_filename('paivakirja')+'.pdf'
    pdf_data = Prawn::Document.new do
      font 'Helvetica'
      text org_full_name, size: 18
      text "Päiväkirja", size: 14
      move_down 10
      text "Nro Päivämäärä"
      text "Tili Debet Kredit Selite"
      stroke_horizontal_rule
      move_down 10
      rows = bills.map do |bill|
        [[bill[:bill_id], bill[:paid_date_fi]],
         [bill[:account_id], bill[:title], bill[:amount], bill[:description]]]
      end.flatten(1)
      Prawn::Table.new(rows, self).draw
      number_pages 'Sivu <page>/<total>',
                   at: [bounds.right - 150, 0],
                   width: 150,
                   align: :right
    end.render
    [pdf_data, filename]
  end

  def self.chart_of_accounts_pdf
    # TODO: Exclude accounts (and headings) that haven't been used this period.
    accounts = Model::Accounts
    prefs = Model.get_preferences
    org_full_name = prefs['org_full_name']
    filename = generate_filename('tilikartta')+'.pdf'
    pdf_data = Prawn::Document.new do
      font 'Helvetica'
      text org_full_name, size: 18
      text "Tilikartta", size: 14
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

  private_class_method def self.generate_zipfile(document, &block)
    prefs = Model.get_preferences
    org_short_name = prefs['org_short_name']
    zipfilepath = '/tmp/'+generate_filename(document)+'.zip' # TODO: use mktemp
    FileUtils.rm_f(zipfilepath)
    Zip::File.open(zipfilepath, Zip::File::CREATE, &block)
    zipfilepath
  end

  private_class_method def self.add_bill_images_to_zip(zipfile, subdir)
    subdir = File.join(File.basename(zipfile.name, '.zip'), subdir)
    missing = []
    bills = Model.get_bills_for_images
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
    generate_zipfile('tositteet') do |zipfile|
      add_bill_images_to_zip(zipfile, '')
    end
  end

  def self.full_statement_zip
    generate_zipfile('tilinpaatos') do |zipfile|
      subdir = File.basename(zipfile.name, '.zip')
      pdf_data, filename = general_journal_pdf
      zipfile.get_output_stream(File.join(subdir, filename)) do |output|
        output.write pdf_data
      end
      pdf_data, filename = chart_of_accounts_pdf
      zipfile.get_output_stream(File.join(subdir, filename)) do |output|
        output.write pdf_data
      end
      add_bill_images_to_zip(zipfile, 'tositteet')
    end
  end

  private_class_method def self.generate_filename(document)
    year = 2017 # TODO
    org_short_name = Model.get_preferences['org_short_name']
    Util.slug("#{org_short_name}-#{year}-#{document}")
  end

end
