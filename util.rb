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
    euros, cents = $1.to_i, ($3 or '0').to_i
    cents = (euros * 100) + cents
    cents
  end

  def self.amount_from_cents(cents)
    return '' if cents.nil? or cents == 0
    euros, cents = cents.divmod(100)
    sprintf('%d,%02d', euros, cents)
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
    words = (full_name or "").split.map(&:capitalize)
    full_name = words.join(' ')
    short_name = full_name
    short_name = "#{words[0]} #{words[1][0]}" unless words.length < 2
    [full_name, short_name]
  end
end
