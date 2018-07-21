# coding: utf-8

require 'date'
require 'open3'

require 'prawn'
require 'prawn/table'
require 'zip'

require_relative 'model'
require_relative 'util'

module Reports
  INDENT_SIZE = 15

  private_class_method def self.get_period_title(model, end_date_only = false)
                         start_date, end_date = model.get_period_start_and_end_date
                         start_date = Util.fi_from_iso_date(start_date)
                         end_date = Util.fi_from_iso_date(end_date)
                         if end_date_only then end_date                          else "#{start_date} - #{end_date}" end
                       end

  def self.financial_statement_pdf(model, template, title, end_date_only)
    prefs = model.get_preferences
    org_full_name = prefs['org_full_name']
    period_title = get_period_title(model, end_date_only)
    filename = generate_filename(model, template) + '.pdf'
    accounts = model.get_accounts(used_only: true)
    balances, profit = model.calc_balances_and_profit(accounts)
    puts("PROFIT = #{profit}")
    puts("BALANCES = #{balances.inspect}")
    pdf_data = Prawn::Document.new do
      font 'Helvetica'
      text org_full_name, size: 18
      text "#{title} #{period_title}", size: 14
      move_down 10
      stroke_horizontal_rule
      move_down 10
      Util.load_financial_statement_rows(template).map do |row|
        if row[:type] == '-'
          move_down 10
          next
        end
        if row[:type] == 'D'
          accounts.each do |acct|
            next unless acct[:account_id] and model.include_account_id?(row[:account_id_ranges], acct[:account_id])
            balance = balances[acct[:account_id]]
            if acct[:account_type] == :expense
              balance = -balance
            end
            indent(INDENT_SIZE * (row[:level] || 0)) do
              text "#{acct[:account_id]} #{acct[:title]} #{Util.amount_from_cents_thousands(balance) || 0}", style: row[:style]
            end
          end
          next
        end
        balance = model.calc_balance_of_account_ranges(balances, row[:account_id_ranges])
        next if balance == 0 and (row[:type] == 'G' or row[:type] == 'T')
        if row[:type] == 'H' or row[:type] == 'G'
          balance = nil  # Otsikkoriveillä ei näytetä euromäärää.
        end
        indent(INDENT_SIZE * (row[:level] || 0)) do
          text "#{row[:text]} #{Util.amount_from_cents_thousands(balance) || 0}", style: row[:style]
        end
      end
      number_pages 'Sivu <page>/<total>',
                   at: [bounds.right - 150, 0],
                   width: 150,
                   align: :right
    end.render
    [pdf_data, filename]
  end

  def self.income_statement_pdf(model)
    self.financial_statement_pdf(model, 'income-statement', 'Tuloslaskelma', false)
  end

  def self.income_statement_detailed_pdf(model)
    self.financial_statement_pdf(model, 'income-statement-detailed', 'Tuloslaskelma erittelyin', false)
  end

  def self.balance_sheet_pdf(model)
    self.financial_statement_pdf(model, 'balance-sheet', 'Tase', true)
  end

  def self.balance_sheet_detailed_pdf(model)
    self.financial_statement_pdf(model, 'balance-sheet-detailed', 'Tase erittelyin', true)
  end

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
            else ''             end)
            credit = (if !e[:debit]
              Util.amount_from_cents(e[:cents])
            else ''             end)
            ['',
             e[:account_id],
             account_lookup[e[:account_id]],
             debit,
             credit,
             bill[:description]]
          end)
      end
      Prawn::Table.new(rows, self, cell_style: {size: 9}).draw
      number_pages 'Sivu <page>/<total>',
                   at: [bounds.right - 150, 0],
                   width: 150,
                   align: :right
    end.render
    [pdf_data, filename]
  end

  def self.general_ledger_pdf(model)
    bills = model.get_bills_for_journal
    prefs = model.get_preferences
    org_full_name = prefs['org_full_name']
    filename = generate_filename(model, 'paakirja') + '.pdf'
    pdf_data = Prawn::Document.new do
      accounts = model.get_accounts
      account_lookup = accounts.map { |a| [a[:account_id], a[:title]] }.to_h
      font 'Helvetica'
      text org_full_name, size: 18
      text 'Pääkirja', size: 14
      move_down 10
      stroke_horizontal_rule
      move_down 10
      rows = [['Nro', 'Tili', '', '', '', '', ''],
              ['', 'Nro', 'Päivämäärä', 'Debet', 'Kredit', 'Saldo', 'Selite']]
      rows += bills.flat_map do |bill|
        [[bill[:bill_id], bill[:paid_date], '', '', '', '']] +
          (bill[:entries].map do |e|
            debit = (if e[:debit]
              Util.amount_from_cents(e[:cents])
            else ''             end)
            credit = (if !e[:debit]
              Util.amount_from_cents(e[:cents])
            else ''             end)
            ['',
             e[:account_id],
             account_lookup[e[:account_id]],
             debit,
             credit,
             bill[:description]]
          end)
      end
      Prawn::Table.new(rows, self, cell_style: {size: 9}).draw
      number_pages 'Sivu <page>/<total>',
                   at: [bounds.right - 150, 0],
                   width: 150,
                   align: :right
    end.render
    [pdf_data, filename]
  end

  def self.chart_of_accounts_pdf(model)
    prefs = model.get_preferences
    org_full_name = prefs['org_full_name']
    period_title = get_period_title(model)
    filename = generate_filename(model, 'tilikartta') + '.pdf'
    accounts = model.get_accounts(used_only: true)
    max_htag_level = accounts.map { |a| a[:htag_level] }.compact.max
    pdf_data = Prawn::Document.new do
      font 'Helvetica'
      text org_full_name, size: 18
      text 'Tilikartta ' + period_title, size: 14
      move_down 10
      stroke_horizontal_rule
      move_down 10
      accounts.each do |acct|
        if acct[:account_id]
          indent(INDENT_SIZE * (max_htag_level - 1)) do
            text "#{acct[:account_id]} #{acct[:title]}"
          end
        else
          indent(INDENT_SIZE * (acct[:htag_level] - 2)) do
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
                             image_in_zip = format('%s/tosite-%03d-%d-%s%s',
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
      pdf_data, filename = general_journal_pdf(model)
      zipfile.get_output_stream(File.join(subdir, filename)) do |output|
        output.write pdf_data
      end
      pdf_data, filename = chart_of_accounts_pdf(model)
      zipfile.get_output_stream(File.join(subdir, filename)) do |output|
        output.write pdf_data
      end
      add_bill_images_to_zip(model, zipfile, 'tositteet')
    end
  end

  private_class_method def self.generate_filename(model, document)
                         year = 2017 # TODO
                         org_short_name = model.get_preferences['org_short_name']
                         Util.slug("#{org_short_name}-#{year}-#{document}")
                       end
end
