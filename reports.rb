# coding: utf-8

require 'date'
require 'open3'

require 'prawn'
require 'prawn/table'
require 'zip'

require_relative 'model'
require_relative 'util'

module Reports
  def self.general_journal_pdf(model)
    bills = model.get_bills_for_journal
    prefs = model.get_preferences
    org_full_name = prefs['org_full_name']
    filename = generate_filename(model, 'paivakirja') + '.pdf'
    pdf_data = Prawn::Document.new do
      accounts = model.get_accounts
      account_lookup = accounts.map { |a| [a[:account_id], a[:title]] }.to_h
      font 'Helvetica'
      text org_full_name, size: 18
      text 'Päiväkirja', size: 14
      move_down 10
      stroke_horizontal_rule
      move_down 10
      rows = [['Nro', 'Päivämäärä', '', '', '', ''],
              ['', 'Tili', '', 'Debet', 'Kredit', 'Selite']]
      rows += bills.flat_map do |bill|
        [[bill[:bill_id], bill[:paid_date], '', '', '', '']] +
          (bill[:entries].map do |e|
             debit = (if e[:debit]
                        Util.amount_from_cents(e[:cents])
                      else '' end)
             credit = (if !e[:debit]
                         Util.amount_from_cents(e[:cents])
                       else '' end)
             ['',
              e[:account_id],
              account_lookup[e[:account_id]],
              debit,
              credit,
              bill[:description]]
           end)
      end
      Prawn::Table.new(rows, self, cell_style: { size: 9 }).draw
      number_pages 'Sivu <page>/<total>',
                   at: [bounds.right - 150, 0],
                   width: 150,
                   align: :right
    end.render
    [pdf_data, filename]
  end

  def self.chart_of_accounts_pdf(model)
    # TODO: Exclude accounts (and headings) that haven't been used this period.
    accounts = model.get_accounts
    prefs = model.get_preferences
    org_full_name = prefs['org_full_name']
    filename = generate_filename(model, 'tilikartta') + '.pdf'
    pdf_data = Prawn::Document.new do
      font 'Helvetica'
      text org_full_name, size: 18
      text 'Tilikartta', size: 14
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

  private_class_method def self.generate_zipfile(model, document, &block)
    prefs = model.get_preferences
    org_short_name = prefs['org_short_name']
    zipfilepath = '/tmp/' + generate_filename(model, document) + '.zip'
    # TODO: use mktemp
    FileUtils.rm_f(zipfilepath)
    Zip::File.open(zipfilepath, Zip::File::CREATE, &block)
    zipfilepath
  end

  private_class_method def self.add_bill_images_to_zip(model, zipfile, subdir)
    subdir = File.join(File.basename(zipfile.name, '.zip'), subdir)
    missing = []
    things = model.get_bills_for_images
    things.each do |thing|
      if thing[:image_id]
        image_in_zip = format('%s/tosite-%04d-%d-%s%s',
                              subdir,
                              thing[:bill_id],
                              thing[:bill_image_num],
                              Util.slug(thing[:description]),
                              File.extname(thing[:image_id]))
        zipfile.get_output_stream(image_in_zip) do |output|
          output.write thing[:image_data]
        end
      else
        missing.push("##{thing[:bill_id]}")
      end
    end
    unless missing.empty?
      zipfile.get_output_stream(format('%s/puuttuvat.txt', subdir)) do |out|
        out.write(missing.join("\r\n"))
      end
    end
  end

  def self.bill_images_zip(model)
    generate_zipfile(model, 'tositteet') do |zipfile|
      add_bill_images_to_zip(model, zipfile, '')
    end
  end

  def self.full_statement_zip(model)
    generate_zipfile(model, 'tilinpaatos') do |zipfile|
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

  private_class_method def self.generate_filename(model, document)
    year = 2017 # TODO
    org_short_name = model.get_preferences['org_short_name']
    Util.slug("#{org_short_name}-#{year}-#{document}")
  end
end
