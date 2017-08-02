# coding: utf-8

require 'date'
require 'open3'

require 'sequel'

require_relative 'util'

module Model
  DB = Sequel.connect(ENV.fetch('DATABASE_URL'))

  sql_logger = Logger.new($stdout)
  sql_logger.formatter = proc do |_serverity, time, _progname, msg|
    # Log all SQL statements except SELECT statements.
    /[^a-z]+ select /i.match(msg) ? '' : "#{time}: SQL: #{msg}\n"
  end
  DB.loggers << sql_logger

  DB.create_table? :preference do
    String :name, primary_key: true
    String :value, null: false
  end

  DB.create_table? :user do
    primary_key :user_id
    String  :email, null: false
    String  :full_name, null: false
    Boolean :is_admin, null: false, default: false
    String  :user_id_google_oauth2, null: true
  end

  DB.create_table? :bill do
    primary_key :bill_id
    String  :description, null: false, default: ''
    Integer :credit_account_id, null: true
    Integer :debit_account_id, null: true
    Integer :unit_count, null: false, default: 1
    Integer :unit_cost_cents, null: true
    String  :paid_date, null: true
    foreign_key :paid_user_id, :user, null: true
    String :reimbursed_date, null: true
    foreign_key :reimbursed_user_id, :user, null: true
    String :closed_date, null: true
    foreign_key :closed_user_id, :user, null: true
    String  :created_date, null: false
    String  :paid_type, null: true
    String  :closed_type, null: true
  end

  DB.create_table? :tag do
    String :tag, primary_key: true
  end

  DB.create_table? :bill_tag do
    foreign_key :bill_id, :bill, null: false
    foreign_key :tag, :tag, null: false
    primary_key %i[bill_id tag]
  end

  DB.create_table? :image do
    String :image_id, primary_key: true
    File :image_data, null: false
  end

  DB.create_table? :bill_image do
    foreign_key :bill_id, :bill, null: false
    Integer :bill_image_num, null: false
    foreign_key :image_id, :image, type: String, null: false
    primary_key %i[bill_id bill_image_num]
  end
  Accounts = Util.load_account_tree

  VALID_PAID_TYPES = %w[car card ebank self].freeze

  def self.valid_paid_type(x)
    return nil unless x
    raise unless VALID_PAID_TYPES.include?(x)
    x
  end

  def self.valid_closed_type(x)
    return nil unless x
    raise unless %w[reimbursed denied].include?(x)
    x
  end

  def self.valid_user_id(x)
    return nil unless x && !x.empty?
    x = x.to_i
    raise unless x >= 1
    x
  end

  def self.valid_nonneg_integer(x)
    return 0 if x.nil?
    x = x.to_i
    raise unless x >= 0
    x
  end

  def self.valid_image_id(x)
    return nil unless x && !x.empty?
    raise unless valid_image_id?(x)
    x
  end

  def self.valid_tags(x)
    return nil unless x
    x = x.split(' ') unless x.is_a?(Array)
    x.each do |tag|
      raise "Invalid tags: #{x.inspect}" unless tag =~ /^[a-zA-Z0-9]+$/
    end
    x.sort.uniq.join(' ')
  end

  def self.valid_image_id?(image_id)
    !!/^[0-9a-f]{40}\.(jpeg|png)$/.match(image_id)
  end

  def self.whack_user(user)
    user = user.clone
    user[:full_name], user[:short_name] = \
      Util.full_and_short_name(user[:full_name])
    user
  end

  def self.get_users
    users = DB[:user].select(:user_id, :full_name).all
    users.map! { |u| whack_user u }
    users.sort! { |a, b| a[:full_name] <=> b[:full_name] }
    users
  end

  def self.put_user(provider:, uid:, email:, full_name:)
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
    if DB[:user].where(uid_field.to_sym => uid).update(columns) != 1
      puts "Creating new user since existing one not found: #{[uid_field, uid].inspect}"
      DB[:user].insert(columns)
    end
    users = DB[:user].select(:user_id, :email, :full_name, :is_admin).where(uid_field.to_sym => uid)
    user = users.first!
    puts("USER IS #{user.inspect}")
    user
  end

  def self.get_user(user_id)
    DB[:user].where(user_id: user_id).first
  end

  # NOTE: The available tags are merely the ones that users can choose from
  # when *adding new tags* to bills. A bill can have *old* tags that are no
  # longer in the available tags list. This is intentional.

  def self.get_available_tags
    DB[:tags].select(:tag).order(:tag).distinct.all
  end

  def self.put_available_tags(tags)
    DB[:tag].delete
    tags.each do |tag|
      DB[:tag].insert(tag: tag)
    end
  end

  def self.get_image_data(image_id)
    raise unless valid_image_id?(image_id)
    image = DB[:image].select(:image_data).where(image_id: image_id).first! # TODO: what if not found
    image[:image_data]
  end

  def self.store_image_data(image_data, image_format)
    hash = Digest::SHA1.hexdigest(image_data)
    image_id = "#{hash}.#{image_format}"
    blob = Sequel.blob(image_data)
    if DB[:image].where(image_id: image_id).update(image_data: blob) != 1
      DB[:image].insert(image_id: image_id, image_data: blob)
    end
    image_id
  end

  def self.store_image_file(tmpfilename)
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

  def self.rotate_image(old_image_id)
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

  def self.get_bill_tags(bill_id)
    DB[:bill_tag].select(:tag).order(:tag).distinct
                 .where(bill_id: bill_id).map { |x| { tag: x[:tag] } }
  end

  def self.get_bill_images(bill_id)
    DB[:bill_image].select(:image_id).where(bill_id: bill_id)
                   .order(:bill_image_num).all
  end

  def self.bill_image_missing?(bill_id)
    DB[:bill_image].where(bill_id: bill_id).count == 0
  end

  def self.get_bills_for_images
    DB[:bill]
      .left_outer_join(:bill_image, bill_id: :bill_id)
      .left_outer_join(:image, image_id: :image_id)
      .select(Sequel.qualify(:bill, :bill_id),
              :bill_image_num,
              Sequel.qualify(:image, :image_id),
              :description,
              :image_data)
      .order(Sequel.qualify(:bill, :bill_id), :bill_image_num).all
  end

  private_class_method def self.bill_base
    base = DB[:bill].order(:bill_id).select do
      [bill_id,
       paid_date,
       closed_date,
       description,
       (unit_count * unit_cost_cents).as(:cents)]
    end
    with_paid_user base
  end

  private_class_method def self.with_paid_user(bill)
    bill.select_append(:paid_user_id)
        .left_join(Sequel[:user].as(:paid_user),
                   user_id: Sequel[:bill][:paid_user_id]).select_append do
      [paid_user_id, paid_user[:full_name].as(:paid_user_full_name)]
    end
  end

  private_class_method def self.with_closed_user(bill)
    bill.select_append(:closed_user_id)
        .left_join(Sequel[:user].as(:closed_user),
                   user_id: Sequel[:bill][:closed_user_id]).select_append do
      [closed_user_id, closed_user[:full_name].as(:closed_user_full_name)]
    end
  end

  private_class_method def self.with_cents(bills)
    bills.select_append{(unit_count * unit_cost_cents).as(:cents)}
  end

  private_class_method def self.resolve_user(user_id)
    if user_id.nil? then nil else DB[:user].where(user_id: user_id).first! end
  end

  def self.get_bill(bill_id)
    bill = bill_base
    bill = with_cents(bill)
    bill = with_closed_user(bill)
    bill = bill.where(bill_id: bill_id).first
    return nil unless bill
    bill[:paid_type] = valid_paid_type(bill[:paid_type])
    VALID_PAID_TYPES.each do |pt|
      bill["paid_type_#{pt}_checked".to_sym] =
        (bill[:paid_type] == pt ? 'checked' : '')
    end
    bill[:paid_user] = resolve_user(bill[:paid_user_id])
    bill[:closed_user] = resolve_user(bill[:closed_user_id])
    bill[:paid_date_fi] = Util.fi_from_iso_date(bill[:paid_date])
    bill[:closed_date_fi] = Util.fi_from_iso_date(bill[:closed_date])
    bill[:tags] = get_bill_tags(bill_id)
    bill[:images] = get_bill_images(bill_id)
    bill[:amount] = Util.amount_from_cents(bill[:cents])
    bill
  end

  def self.get_bills_and_all_tags(current_user)
    puts("current user is #{current_user.inspect}")
    bills = DB[:bill].left_outer_join(:user, user_id: :paid_user_id)
                     .select do
      [bill_id,
       paid_date,
       full_name.as(:paid_user_full_name),
       description]
    end
    bills = with_cents(bills)
    bills = bills.order(:bill_id)
    unless current_user[:is_admin]
      bills = bills.where(paid_user_id: current_user[:user_id])
    end
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

  def self.get_bills_for_journal
    DB[:bill].select do
      [bill_id, paid_date, description,
       (unit_count * unit_cost_cents).as(:cents)]
    end.order(:paid_date).all.map do |bill|
      bill[:paid_date_fi] = Util.fi_from_iso_date(bill[:paid_date])
      bill[:amount] = Util.amount_from_cents(bill[:cents])
      bill
    end
  end

  def self.update_bill!(bill_id, r, current_user)
    # TODO: don't allow updating a closed bill
    bill = {}
    if current_user[:is_admin]
      # TODO: more admin-only fields
      bill[:paid_user_id] = valid_user_id(r[:paid_user_id])
      bill[:closed_type] = valid_closed_type(r[:closed_type])
      bill[:closed_user_id] = valid_user_id(r[:closed_user_id])
      bill[:closed_date] = Util.iso_from_fi_date(r[:closed_date_fi])
      bill[:credit_account_id] = valid_nonneg_integer(r[:credit_account_id])
      bill[:debit_account_id] = valid_nonneg_integer(r[:debit_account_id])
    else
      # TODO: proper errors
      bill[:paid_user_id] ||= current_user[:user_id]
      raise unless bill[:paid_user_id] == current_user[:user_id]
    end
    bill[:paid_date] = Util.iso_from_fi_date(r[:paid_date_fi])
    bill[:paid_type] = valid_paid_type(r[:paid_type])
    bill[:unit_count] = 1
    bill[:unit_cost_cents] = Util.cents_from_amount(r[:amount])
    # bill[:tags] = valid_tags(r[:tags])
    bill[:description] = r[:description]
    bill[:bill_id] = bill_id
    DB[:bill_image].where(bill_id: bill_id).delete
    image_id = valid_image_id(r[:image_id])
    if image_id
      DB[:bill_image].insert(bill_id: bill_id, bill_image_num: 1,
                             image_id: image_id)
    end
    DB[:bill].where(bill_id: bill_id).update(bill)
    bill
  end

  def self.put_bill(bill_id, params, current_user)
    bill = DB[:bill].where(bill_id: bill_id).first
    raise 'No such bill' unless bill
    update_bill! bill_id, params, current_user
  end

  def self.post_bill(params, current_user)
    bill_id = DB[:bill].insert(
      created_date: DateTime.now.strftime('%Y-%m-%d')
    )
    update_bill! bill_id, params, current_user
  end

  DEFAULT_PREFERENCES = {
    'org_full_name' => '',
    'org_short_name' => ''
  }.freeze

  def self.get_preferences
    prefs = DEFAULT_PREFERENCES.dup
    DB[:preference].select(:name, :value).each do |pref|
      prefs[pref[:name]] = pref[:value]
    end
    prefs
  end

  def self.put_preferences(prefs)
    raise unless prefs.keys.all? { |name| DEFAULT_PREFERENCES.keys.include?(name) }
    prefs.each_pair do |name, value|
      if DB[:preference].where(name: name).update(value: value) != 1
        DB[:preference].insert(name: name, value: value)
      end
    end
    nil
  end
end
