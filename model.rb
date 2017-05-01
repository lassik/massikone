require "sequel"

module Model

  DB = Sequel.connect(ENV.fetch("DATABASE_URL"))

  DB.create_table? :users do
    primary_key :user_id
    String  :email
    String  :full_name
    Boolean :is_admin
    String  :user_id_google_oauth2
  end

  DB.create_table? :bills do
    primary_key :bill_id
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
    String  :paid_type
    String  :closed_type
  end

  DB.create_table? :tags do
    String :tag
  end

  DB.create_table? :bill_tags do
    Integer :bill_id
    String :tag
  end

  DB.create_table? :images do
    Integer :image_id
    File :image_data
  end

  VALID_PAID_TYPES = ["car", "card", "ebank", "self"]

  def self.valid_paid_type(x)
    return nil unless x
    raise unless VALID_PAID_TYPES.include?(x)
    x
  end

  def self.valid_closed_type(x)
    return nil unless x
    raise unless ["reimbursed", "denied"].include?(x)
    x
  end

  def self.valid_user_id(x)
    return nil unless x
    x = x.to_i
    raise unless x >= 1
    x
  end

  def self.valid_nonneg_integer(x)
    x = x.to_i
    raise unless x >= 0
    x
  end

  def self.valid_image_id(x)
    return nil unless x and not x.empty?
    raise unless valid_image_id?(x)
    x
  end

  def self.valid_tags(x)
    return nil unless x
    x = x.split(" ") unless x.kind_of?(Array)
    x.each do |tag|
      raise "Invalid tags: #{x.inspect}" unless tag =~ /^[a-zA-Z0-9]+$/
    end
    x.sort.uniq.join(" ")
  end

  def self.valid_image_id?(image_id)
    not not /^[0-9a-f]{40}\.(jpeg|png)$/.match image_id
  end

  def self.get_users()
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
    users
  end

  def self.put_user(uid_field, uid, email, full_name)
    puts "Login #{[uid_field, uid, email, full_name].inspect}"
    user = DB["select * from users where #{uid_field} = ?", uid].first
    unless user
      DB["insert into users (#{uid_field}) values (?)", uid].insert
      user = DB["select * from users where #{uid_field} = ?", uid].first
    end
    DB["update users set email = ?, full_name = ? where #{uid_field} = ?", email, full_name, uid].update
    user
  end

  # NOTE: The available tags are merely the ones that users can choose from
  # when *adding new tags* to bills. A bill can have *old* tags that are no
  # longer in the available tags list. This is intentional.

  def self.get_available_tags()
    DB.fetch("select distinct tag from tags order by tag").all
  end

  def self.put_available_tags(tags)
    DB.delete("delete from tags")
    tags.each do |tag|
      DB.insert("insert into tags (tag) values (?)", tag)
    end
  end

  def self.fetch_image_data(image_id)
    raise unless valid_image_id?(image_id)
    image = DB.fetch("select image_data from images"+
                     " where image_id = ?", image_id).first
    # TODO what if not found
    image[:image_data]
  end

  def self.store_image_data(image_data, image_format)
    hash = Digest::SHA1.hexdigest(image_data)
    image_id = "#{hash}.#{image_format}"
    DB[:images].insert(image_id: image_id, image_data: Sequel.blob(image_data))
    image_id
  end

  def self.store_image_file(tmpfilename)
    # We don't need the complexity of minimagick et.al.
    #
    # Don't trust filename extension given by user. It is sometimes wrong even
    # when not malicious. Better trust ImageMagick to identify the file format.
    tmpfilename = File.absolute_path(tmpfilename)
    old_image_format, err_msg, status = Open3.capture3(
                                 "identify",
                                 "-format", "%[m]",
                                 tmpfilename)
    new_image_format = nil
    if status.exitstatus == 0
      new_image_format = case old_image_format
                         when "JPEG" then "jpeg"
                         when "PNG"  then "png"
                         end
    end
    unless new_image_format
      # TODO better error message for user. does roda have a good pre-made exception class we can use?
      raise "Bad image format: #{err_msg}"
    end
    new_image_data, err_msg, status = Open3.capture3(
                               "convert",
                               "-strip",
                               "-define", "png:include-chunk=none",
                               "-resize", "900x900>",
                               "-colorspace", "Gray",
                               "-separate",
                               "-average",
                               tmpfilename,
                               "#{new_image_format}:-")
    unless status.exitstatus == 0
      raise "Image conversion error: #{err_msg}"
    end
    store_image_data(new_image_data, new_image_format)
  end

  def self.rotate_image(old_image_id)
    raise unless valid_image_id?(old_image_id)
    old_image_data = fetch_image_data(old_image_id)
    image_format = File.extname(old_image_id)[1..-1]
    new_image_data, err_msg, status = Open3.capture3(
                               "convert",
                               "#{image_format}:-",
                               "-rotate", "90",
                               "#{image_format}:-",
                               :stdin_data => old_image_data)
    unless status.exitstatus == 0
      raise "Image conversion error: #{err_msg}"
    end
    store_image_data(new_image_data, image_format)
  end

  def self.fetch_bill(bill_id)
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

  def self.fetch_bills_and_all_tags(current_user)
    puts("current user is #{current_user.inspect}")
    all_tags = []
    sql = ("select bill_id, unit_count * unit_cost_cents as cents, paid_date, tags, description, pu.full_name as paid_user_full_name"+
           " from bills"+
           " left join users pu on pu.user_id = bills.paid_user_id")
    bills = if current_user[:is_admin]
              DB.fetch(sql+" order by bill_id").all
            else
              DB.fetch(sql+" where paid_user_id = ? order by bill_id", current_user[:user_id]).all
            end
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

  def self.update_bill!(bill_id, r, current_user)
    # TODO: don't allow updating a closed bill
    bill = {}
    if current_user[:is_admin]
      # TODO: more admin-only fields
      bill[:paid_user_id] = valid_user_id(r[:paid_user_id])
      bill[:closed_type] = valid_closed_type(r[:closed_type])
      bill[:closed_user_id] = valid_user_id(r[:closed_user_id])
      bill[:closed_date] = iso_from_fi_date(r[:closed_date_fi])
    else
      # TODO proper errors
      bill[:paid_user_id] ||= current_user[:user_id]
      raise unless bill[:paid_user_id] == current_user[:user_id]
    end
    bill[:paid_date] = iso_from_fi_date(r[:paid_date_fi])
    bill[:paid_type] = valid_paid_type(r[:paid_type])
    bill[:unit_count] = valid_nonneg_integer(r[:unit_count])
    bill[:unit_cost_cents] = valid_nonneg_integer(r[:unit_cost_cents])
    bill[:image_id] = valid_image_id(r[:image_id])
    bill[:tags] = valid_tags(r[:tags])
    bill[:description] = r[:description]
    DB[:bills].where(:bill_id=>bill_id).update(bill)
    bill_id
  end

  def self.put_bill(bill_id, params, current_user)
    bill = DB.fetch("select * from bills where bill_id = :bill_id",
                    :bill_id=>bill_id).first
    raise "No such bill" unless bill
    update_bill! bill_id, params, current_user
    nil
  end

end
