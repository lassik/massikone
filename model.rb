# coding: utf-8

require 'date'
require 'open3'

require 'sequel'

require_relative 'util'

class Model
  LOG_SQL_SELECTS = false
  ACCOUNT_NESTING_LEVEL = 9

  @@sql_logger = Logger.new($stdout)
  @@sql_logger.formatter = proc do |_serverity, time, _progname, msg|
    if !LOG_SQL_SELECTS && /[^a-z]+ select /i.match(msg)
      ''
    else
      "#{time}: SQL: #{msg}\n"
    end
  end

  def initialize(user_id)
    Sequel.connect(ENV.fetch('DATABASE_URL')) do |db|
      db.loggers << @@sql_logger
      @db = db
      db.transaction do
        migrate!
        @user = nil
        @user = get_user(user_id) if user_id
        yield(self)
      end
    end
  end

  attr_reader :user

  private def migrate!
    @db.create_table? :preference do
      String :name, primary_key: true
      String :value, null: false
    end

    @db.create_table? :user do
      primary_key :user_id
      String :email, null: false
      String :full_name, null: false
      Boolean :is_admin, null: false, default: false
      String :user_id_google_oauth2, null: true
    end

    @db.create_table? :period do
      primary_key :period_id
      String :start_date, null: true
      String :end_date, null: true
    end

    @db.create_table? :period_account do
      foreign_key :period_id, :period, null: false
      Integer :account_id, null: false
      String :title, null: false
      Integer :starting_balance_cents, null: false, default: 0
      Integer :nesting_level, null: false, default: 0
      primary_key %i[period_id account_id nesting_level]
    end

    @db.create_table? :bill do
      primary_key :bill_id
      String  :description, null: false, default: ''
      String  :paid_date, null: true
      foreign_key :paid_user_id, :user, null: true
      String :closed_date, null: true
      foreign_key :closed_user_id, :user, null: true
      String  :created_date, null: false
      String  :closed_type, null: true
    end

    @db.create_table? :bill_entry do
      foreign_key :bill_id, :bill, null: false
      Integer :row_number, null: false
      Integer :account_id, null: false
      # foreign_key :account_id, :account, null: false
      Boolean :debit, null: false
      Integer :unit_count, null: false, default: 1
      Integer :unit_cost_cents, null: false
      String :description, null: false
      primary_key %i[bill_id row_number]
    end

    @db.create_table? :tag do
      String :tag, primary_key: true
    end

    @db.create_table? :bill_tag do
      foreign_key :bill_id, :bill, null: false
      foreign_key :tag, :tag, type: String, null: false
      primary_key %i[bill_id tag]
    end

    @db.create_table? :image do
      String :image_id, primary_key: true
      File :image_data, null: false
    end

    @db.create_table? :bill_image do
      foreign_key :bill_id, :bill, null: false
      Integer :bill_image_num, null: false
      foreign_key :image_id, :image, type: String, null: false
      primary_key %i[bill_id bill_image_num]
    end
  end

  def valid_closed_type(x)
    return nil unless x
    raise unless %w[reimbursed denied].include?(x)
    x
  end

  def valid_user_id(x)
    return nil unless x && !x.empty?
    x = x.to_i
    raise unless x >= 1
    x
  end

  def valid_nonneg_integer(x)
    return 0 if x.nil?
    x = x.to_i
    raise unless x >= 0
    x
  end

  def valid_image_id(x)
    return nil unless x && !x.empty?
    raise unless valid_image_id?(x)
    x
  end

  def valid_tags(x)
    return nil unless x
    x = x.split(' ') unless x.is_a?(Array)
    x.each do |tag|
      raise "Invalid tags: #{x.inspect}" unless tag =~ /^[a-zA-Z0-9]+$/
    end
    x.sort.uniq.join(' ')
  end

  def valid_image_id?(image_id)
    !!/^[0-9a-f]{40}\.(jpeg|png)$/.match(image_id)
  end

  def whack_user(user)
    user = user.clone
    user[:full_name], user[:short_name] = \
      Util.full_and_short_name(user[:full_name])
    user
  end

  def get_users
    users = @db[:user].select(:user_id, :full_name).all
    users.map! { |u| whack_user u }
    users.sort! { |a, b| a[:full_name] <=> b[:full_name] }
    users
  end

  def put_user(provider:, uid:, email:, full_name:)
    puts "Login #{[provider, uid, email, full_name].inspect}"
    uid_field = "user_id_#{provider}"
    missing = []
    missing.push('käyttäjä-ID') unless uid
    missing.push('nimi') unless full_name
    missing.push('sähköposti') unless email
    unless missing.empty?
      raise "Seuraavia tietoja ei saatu: #{missing.join(', ')}"
    end
    columns = {
      :email => email, :full_name => full_name, uid_field.to_sym => uid
    }
    if @db[:user].count == 0
      puts 'Creating first user and making them an admin'
      columns[:is_admin] = true
      @db[:user].insert(columns)
    elsif @db[:user].where(uid_field.to_sym => uid).update(columns) != 1
      puts 'Creating new user since existing one not found'
      @db[:user].insert(columns)
    end
    user = @db[:user].where(uid_field.to_sym => uid)
                     .select(:user_id, :email, :full_name, :is_admin).first!
    puts("User is #{user.inspect}")
    user
  end

  private def get_user(user_id)
    @db[:user].where(user_id: user_id).first
  end

  # NOTE: The available tags are merely the ones that users can choose from
  # when *adding new tags* to bills. A bill can have *old* tags that are no
  # longer in the available tags list. This is intentional.

  def get_available_tags
    @db[:tags].select(:tag).order(:tag).distinct.all
  end

  def put_available_tags(tags)
    @db[:tag].delete
    tags.each do |tag|
      @db[:tag].insert(tag: tag)
    end
  end

  def get_image_data(image_id)
    raise unless valid_image_id?(image_id)
    image = @db[:image].select(:image_data).where(image_id: image_id).first! # TODO: what if not found
    image[:image_data]
  end

  def store_image_data(image_data, image_format)
    hash = Digest::SHA1.hexdigest(image_data)
    image_id = "#{hash}.#{image_format}"
    blob = Sequel.blob(image_data)
    if @db[:image].where(image_id: image_id).update(image_data: blob) != 1
      @db[:image].insert(image_id: image_id, image_data: blob)
    end
    image_id
  end

  def store_image_file(tmpfilename)
    # We don't need the complexity of minimagick et.al.
    #
    # Don't trust filename extension given by user. It is sometimes wrong even
    # when not malicious. Better trust ImageMagick to identify the file format.
    tmpfilename = File.absolute_path(tmpfilename)
    old_image_format, err_msg, status = Open3.capture3(
      'identify',
      '-format', '%[m]',
      tmpfilename
    )
    old_image_format.chomp!
    new_image_format = nil
    if status.exitstatus == 0
      new_image_format = case old_image_format
                         when 'JPEG' then 'jpeg'
                         when 'PNG'  then 'png'
                         end
    end
    unless new_image_format
      # TODO: better error message for user. does roda have a good pre-made exception class we can use?
      raise "Bad image format: #{old_image_format.inspect} #{err_msg.inspect}"
    end
    new_image_data, err_msg, status = Open3.capture3(
      'convert',
      '-strip',
      '-define', 'png:include-chunk=none',
      '-trim', '+repage',
      '-resize', '900x900>',
      '-colorspace', 'Gray',
      '-separate',
      '-average',
      tmpfilename,
      "#{new_image_format}:-"
    )
    raise "Image conversion error: #{err_msg}" unless status.exitstatus == 0
    store_image_data(new_image_data, new_image_format)
  end

  def rotate_image(old_image_id)
    raise unless valid_image_id?(old_image_id)
    old_image_data = get_image_data(old_image_id)
    image_format = File.extname(old_image_id)[1..-1]
    new_image_data, err_msg, status = Open3.capture3(
      'convert',
      "#{image_format}:-",
      '-rotate', '90',
      "#{image_format}:-",
      stdin_data: old_image_data
    )
    raise "Image conversion error: #{err_msg}" unless status.exitstatus == 0
    store_image_data(new_image_data, image_format)
  end

  def get_bill_tags(bill_id)
    @db[:bill_tag].select(:tag).order(:tag).distinct
                  .where(bill_id: bill_id).map { |x| { tag: x[:tag] } }
  end

  def get_bill_images(bill_id)
    @db[:bill_image].select(:image_id).where(bill_id: bill_id)
                    .order(:bill_image_num).all
  end

  def bill_image_missing?(bill_id)
    @db[:bill_image].where(bill_id: bill_id).count == 0
  end

  def get_bills_for_images
    @db[:bill]
      .left_join(:bill_image, bill_id: :bill_id)
      .left_join(:image, image_id: :image_id)
      .select(Sequel.qualify(:bill, :bill_id),
              :bill_image_num,
              Sequel.qualify(:image, :image_id),
              :description,
              :image_data)
      .order(Sequel.qualify(:bill, :bill_id), :bill_image_num).all
  end

  private def bill_base
    base = @db[:bill].order(:bill_id).select do
      [bill_id, paid_date, closed_date, description]
    end
    with_paid_user base
  end

  private def with_paid_user(bill)
    bill.left_join(Sequel[:user].as(:paid_user),
                   user_id: Sequel[:bill][:paid_user_id]).select_append do
      [paid_user_id, paid_user[:full_name].as(:paid_user_full_name)]
    end
  end

  private def with_closed_user(bill)
    bill.left_join(Sequel[:user].as(:closed_user),
                   user_id: Sequel[:bill][:closed_user_id]).select_append do
      [closed_user_id, closed_user[:full_name].as(:closed_user_full_name)]
    end
  end

  private def with_cents(bill)
    db = @db
    bill.select_append do
      sums = db[:bill_entry].where(bill_id: Sequel[:bill][:bill_id]).select do
        sum(unit_count * unit_cost_cents)
      end
      max(sums.where(:debit), sums.exclude(:debit)).as(:cents)
    end
  end

  private def bill_entries(bill_id)
    @db[:bill_entry].where(bill_id: bill_id).order(:row_number).select do
      [row_number, account_id, debit, description,
       (unit_count * unit_cost_cents).as(:cents)]
    end
  end

  private def bill_entries!(bill_id, entries)
    @db[:bill_entry].where(bill_id: bill_id).delete
    entries.each_with_index do |e, row_number|
      @db[:bill_entry].insert(
        bill_id: bill_id,
        row_number: row_number,
        account_id: e[:account_id],
        debit: e[:debit],
        unit_count: 1,
        unit_cost_cents: e[:unit_cost_cents],
        description: e[:description]
      )
    end
  end

  def get_bill(bill_id)
    this_bill_id = bill_id
    bill = bill_base
    bill = with_cents(bill)
    bill = with_closed_user(bill)
    bill = bill.where(bill_id: bill_id).first
    return nil unless bill
    bill[:paid_date_fi] = Util.fi_from_iso_date(bill[:paid_date])
    bill[:closed_date_fi] = Util.fi_from_iso_date(bill[:closed_date])
    bill[:tags] = get_bill_tags(bill_id)
    bill[:images] = get_bill_images(bill_id)
    bill[:amount] = Util.amount_from_cents(bill[:cents])
    entries = @db[:bill_entry].where(bill_id: bill_id).limit(1)
    bill[:credit_account_id] = entries.exclude(:debit).select_map(:account_id).first
    bill[:debit_account_id] = entries.where(:debit).select_map(:account_id).first
    bill[:prev_bill_id] = @db[:bill].where { Sequel[:bill][:bill_id] < this_bill_id }.max(:bill_id)
    bill[:next_bill_id] = @db[:bill].where { Sequel[:bill][:bill_id] > this_bill_id }.min(:bill_id)
    bill
  end

  def get_bills_and_all_tags
    puts("current user is #{@user.inspect}")
    bills = bill_base
    bills = with_cents(bills)
    bills = bills.where(paid_user_id: @user[:user_id]) unless @user[:is_admin]
    bills = bills.all
    bills.each do |bill|
      bill[:amount] = Util.amount_from_cents(bill[:cents])
      bill[:description] = Util.shorten(bill[:description])
      bill[:paid_user_full_name], = \
        Util.full_and_short_name(bill[:paid_user_full_name])
      bill[:paid_date_fi] = Util.fi_from_iso_date(bill[:paid_date])
      bill[:tags] = get_bill_tags(bill[:bill_id])
      bill[:image_missing] = bill_image_missing?(bill[:bill_id])
    end
    all_tags = bills.flat_map { |bill| bill[:tags] } .sort.uniq
    [bills, all_tags]
  end

  def get_bills_for_journal
    bill_base.map do |bill|
      bill[:entries] = bill_entries(bill[:bill_id])
      bill
    end
  end

  def get_bills_for_compare
    raise unless @user[:is_admin]
    with_cents(bill_base).all.map do |bill|
      {
        'bill_id' => bill[:bill_id],
        'date' => bill[:paid_date],
        'cents' => bill[:cents],
        'description' => Util.shorten(bill[:description])
      }
    end
  end

  def update_bill!(bill_id, r)
    # TODO: don't allow updating a closed bill
    bill = {}
    credit_account_id = nil
    debit_account_id = nil
    if @user[:is_admin]
      # TODO: more admin-only fields
      bill[:paid_user_id] = valid_user_id(r[:paid_user_id])
      bill[:closed_type] = valid_closed_type(r[:closed_type])
      bill[:closed_user_id] = valid_user_id(r[:closed_user_id])
      bill[:closed_date] = Util.iso_from_fi_date(r[:closed_date_fi])
      credit_account_id = valid_nonneg_integer(r[:credit_account_id])
      debit_account_id = valid_nonneg_integer(r[:debit_account_id])
    else
      # TODO: proper errors
      bill[:paid_user_id] ||= @user[:user_id]
      raise unless bill[:paid_user_id] == @user[:user_id]
    end
    bill[:paid_date] = Util.iso_from_fi_date(r[:paid_date_fi])
    # bill[:tags] = valid_tags(r[:tags])
    bill[:description] = r[:description]
    bill[:bill_id] = bill_id
    unit_cost_cents = Util.cents_from_amount(r[:amount])
    entries = []
    if credit_account_id && unit_cost_cents
      entries.push(debit: false, account_id: credit_account_id,
                   unit_cost_cents: unit_cost_cents, description: 'Credit')
    end
    if debit_account_id && unit_cost_cents
      entries.push(debit: true,  account_id: debit_account_id,
                   unit_cost_cents: unit_cost_cents, description: 'Debit')
    end
    bill_entries!(bill_id, entries)
    @db[:bill_image].where(bill_id: bill_id).delete
    image_id = valid_image_id(r[:image_id])
    if image_id
      @db[:bill_image].insert(bill_id: bill_id, bill_image_num: 1,
                              image_id: image_id)
    end
    @db[:bill].where(bill_id: bill_id).update(bill)
    bill
  end

  def put_bill(bill_id, params)
    bill = @db[:bill].where(bill_id: bill_id).first
    raise 'No such bill' unless bill
    update_bill! bill_id, params
  end

  def post_bill(params)
    bill_id = @db[:bill].insert(
      created_date: DateTime.now.strftime('%Y-%m-%d')
    )
    update_bill! bill_id, params
  end

  private def populate_accounts
    if @db[:period].where(period_id: 1).update(period_id: 1) != 1
      @db[:period].insert(period_id: 1)
    end
    if @db[:period_account].count == 0
      Util.load_account_tree.each do |a|
        @db[:period_account].insert(
          period_id: 1,
          account_id: a[:raw_account_id],
          title: a[:title],
          nesting_level: (a[:htag_level] ? a[:htag_level] - 1 : ACCOUNT_NESTING_LEVEL)
        )
      end
    end
  end

  def get_accounts
    populate_accounts
    @db[:period_account].order(:account_id, :nesting_level)
                        .select(:account_id, :title, :nesting_level).map do |a|
      is_account = (a[:nesting_level] == ACCOUNT_NESTING_LEVEL)
      dash_level = is_account ? nil : 1 + a[:nesting_level]
      htag_level = is_account ? nil : 2 + a[:nesting_level]
      { raw_account_id: a[:account_id],
        account_id: (is_account ? a[:account_id] : nil),
        title: a[:title],
        prefix: (is_account ? a[:account_id].to_s : '=' * dash_level),
        htag_level: htag_level }
    end
  end

  DEFAULT_PREFERENCES = {
    'org_full_name' => '',
    'org_short_name' => ''
  }.freeze

  def get_preferences
    prefs = DEFAULT_PREFERENCES.dup
    @db[:preference].select(:name, :value).each do |pref|
      prefs[pref[:name]] = pref[:value]
    end
    prefs
  end

  def put_preferences(prefs)
    raise unless prefs.keys.all? { |name| DEFAULT_PREFERENCES.keys.include?(name) }
    prefs.each_pair do |name, value|
      if @db[:preference].where(name: name).update(value: value) != 1
        @db[:preference].insert(name: name, value: value)
      end
    end
    nil
  end
end
