#!/usr/bin/env ruby


file = ARGV.last

File.open(file) do |f|
	msg = []
	start_found = false
	f.each_byte do |b|
		if !start_found && b == 0x81
			start_found = true
			msg = [b]
		elsif start_found
			msg << b
			if msg.size == 55 && b == 0xFF
				start_found = false
				output = ""
				msg.each do |m|
					output << ("%0.2X "% m)
				end
				output.chomp!
				puts output
			elsif msg.size > 55
				start_found = false
				msg = []
			elsif msg[1] != 0x00
				start_found = false
			end
		end
	end
end