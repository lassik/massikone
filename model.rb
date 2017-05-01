require "sequel"

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

def valid_paid_type(x)
  return nil unless x
  raise unless VALID_PAID_TYPES.include?(x)
  x
end

def valid_closed_type(x)
  return nil unless x
  raise unless ["reimbursed", "denied"].include?(x)
  x
end

def valid_user_id(x)
  return nil unless x
  x = x.to_i
  raise unless x >= 1
  x
end

def valid_nonneg_integer(x)
  x = x.to_i
  raise unless x >= 0
  x
end

def valid_image_id(x)
  return nil unless x and not x.empty?
  raise unless valid_image_id?(x)
  x
end

def valid_tags(x)
  return nil unless x
  x = x.split(" ") unless x.kind_of?(Array)
  x.each do |tag|
    raise "Invalid tags: #{x.inspect}" unless tag =~ /^[a-zA-Z0-9]+$/
  end
  x.sort.uniq.join(" ")
end

def valid_image_id?(image_id)
  not not /^[0-9a-f]{40}\.(jpeg|png)$/.match image_id
end

def fetch_image_data(image_id)
  raise unless valid_image_id?(image_id)
  image = DB.fetch("select image_data from images"+
                   " where image_id = ?", image_id).first
  # TODO what if not found
  image[:image_data]
end

def store_image_data(image_data, image_format)
  hash = Digest::SHA1.hexdigest(image_data)
  image_id = "#{hash}.#{image_format}"
  DB[:images].insert(image_id: image_id, image_data: Sequel.blob(image_data))
  image_id
end

def store_image_file(tmpfilename)
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

def rotate_image(old_image_id)
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
