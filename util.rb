module Util

  def self.slug(str)
    return '' if str.nil?
    str.downcase.gsub(/\s+/, '-').gsub(/[^\w-]/, '').gsub(/^-/, '').gsub(/-$/, '').gsub(/--+/, '')
  end

  def self.derive_full_and_short_name(full_name)
    words = full_name.split.map(&:capitalize)
    full_name = words.join(' ')
    short_name = full_name
    short_name = "#{words[0]} #{words[1][0]}" unless words.length < 2
    [full_name, short_name]
  end

end
