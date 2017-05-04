module Util
  FI_DATE = '%-d.%-m.%Y'.freeze
  ISO_DATE = '%Y-%m-%d'.freeze

  def self.fi_from_iso_date(str)
    return nil unless str && !str.empty?
    DateTime.strptime(str, ISO_DATE).strftime(FI_DATE)
  end

  def self.iso_from_fi_date(str)
    return nil unless str && !str.empty?
    DateTime.strptime(str, FI_DATE).strftime(ISO_DATE)
  end

  def self.amount_from_cents(cents)
    return '' if cents.nil?
    euros, cents = cents.divmod(100)
    sprintf('%d,%02d', euros, cents)
  end

  def self.shorten(str)
    str.strip[0..50]
  end

  def self.slug(str)
    return '' if str.nil?
    str.downcase.gsub(/\s+/, '-').gsub(/[^\w-]/, '').gsub(/^-/, '').gsub(/-$/, '').gsub(/--+/, '')
  end

  def self.full_and_short_name(full_name)
    words = (full_name or "").split.map(&:capitalize)
    full_name = words.join(' ')
    short_name = full_name
    short_name = "#{words[0]} #{words[1][0]}" unless words.length < 2
    [full_name, short_name]
  end
end
