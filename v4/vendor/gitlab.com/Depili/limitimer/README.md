# Reverse engineering DSAN limitimer stage timer communications

## Physical layer

RS-485, 19200bps 8 data bits, no parity, 1 stop bit.

RJ-45 pinout is as follows:

1 - +12V
2 - RS-485 B (D-)
3 - RS-485 A (D+)
4 - GND
5 - GND
6 - RS-485 A (D+)
7 - RS-485 B (D-)
8 - +12V

The XLR connector:

1 - +12V
2 - GND
3 - D-

## Packet format

Packets start with 0x81 byte. The payload is encoded in 7 bits per byte, leaving high bit unset. 0x83 designates end of payload and start of checksum. It is followed by two checksum bytes for a 16bit checksum, using CRC with the modbus polynomial and settings, calculated from the message start character to the start of checksum character, including them both. Message ends after the two checksum bytes with 0xFF

Second byte of the message seems to be the message type identifier. Two types of messages were observed with the limitimer control box in master mode. a 55 byte message with type 0x00 which contains the whole state in one packet and a 6 byte 0x10 message, which is always sent between two state updates, maybe a sync packet of some kind.