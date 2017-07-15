module Util
  def self.fi_from_iso_date(str)
    return nil unless str && !str.empty?
    DateTime.strptime(str, '%Y-%m-%d').strftime('%-d.%-m.%Y')
  end

  def self.iso_from_fi_date(str)
    return nil unless str && !str.empty?
    DateTime.strptime(str, '%d.%m.%Y').strftime('%Y-%m-%d')
  end

  def self.cents_from_amount(amount)
    return nil if amount.nil?
    amount = amount.gsub /\s+/, ''
    return nil if amount == ''
    unless amount =~ /^(\d+)(,(\d\d))?$/
      raise "Invalid amount: #{amount.inspect}"
    end
    euros = Regexp.last_match(1).to_i
    cents = (Regexp.last_match(3) || '0').to_i
    cents = (euros * 100) + cents
    cents
  end

  def self.amount_from_cents(cents)
    return '' if cents.nil? || cents == 0
    euros, cents = cents.divmod(100)
    format('%d,%02d', euros, cents)
  end

  def self.shorten(str)
    str.partition("\n")[0].gsub(/\s+/, ' ').strip[0..50]
  end

  def self.slug(str)
    return '' if str.nil?
    str = str.partition("\n")[0]
    str = str.downcase.gsub(/\s+/, '-').gsub(/[^\w-]/, '')
    str = str.gsub(/--+/, '-').gsub(/^-/, '')
    str = shorten(str).gsub(/-$/, '')
    str
  end

  def self.full_and_short_name(full_name)
    words = (full_name || '').split.map(&:capitalize)
    full_name = words.join(' ')
    short_name = full_name
    short_name = "#{words[0]} #{words[1][0]}" unless words.length < 2
    [full_name, short_name]
  end

  def self.load_account_tree
    # H;1011;Vastaavaa;0 -- Heading ; first account ID ; title ; level
    # A;1011;Perustamismenot;0 -- Account ; ID ; title ; flags
    # Sort by ID. Multiple headings (and one account) can share the same ID.
    # Sort those by level (assume all accounts are deeper than any heading).
    list = []
    File.foreach('chart-of-accounts.txt').drop(1).each do |line|
      fields = line.chomp.split(';')
      raise unless fields.length == 4
      row_type, account_id, title, last_field = fields
      account_id = account_id.to_i
      last_field = last_field.to_i
      dash_level = { 'H' => 1 + last_field, 'A' => nil }[row_type]
      htag_level = { 'H' => [2 + last_field, 6].min, 'A' => nil }[row_type]
      sort_level = { 'H' => last_field, 'A' => 9 }[row_type]
      sort_key = 10 * account_id + sort_level
      prefix = row_type == 'H' ? ('=' * dash_level) : account_id.to_s
      account_id = nil unless row_type == 'A'
      list.push(account_id: account_id, title: title,
                prefix: prefix, htag_level: htag_level, sort_key: sort_key)
    end
    list.sort! { |a, b| a[:sort_key] <=> b[:sort_key] }
    list
  end
end
