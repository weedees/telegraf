#!/usr/bin/env ruby
require 'json'

loop do
  # example input: "{"fields":{"count":0},"name":"counter_ruby","tags":{"host":"localhost"},"timestamp":1586374982}"
  line = STDIN.readline.chomp

  l = JSON.parse(line)
  l["fields"]["count"] = l["fields"]["count"] * 2
  puts l.to_json
  STDOUT.flush
end
