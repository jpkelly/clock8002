# DSAN Perfect Cue

## Requirements

### Hardware

The limitimer functionality requires an RS-485 adapter and a wiring adapter. For raspberry pi a RS-485 hat, like the waveshare rs-485 + can hat, is recommended. Any hat that uses the UART provided on the gpio header should work. Most USB rs-485 dongles can also be made to work, but doing so will require loading the kernel modules for it.

For raspberry pi hats the serial device to use is `/dev/ttyAMA0`

### Wiring

See the [piClock wiring diagram (PDF)](docs/piClockWiring.pdf) for an overview of all connections.

The XLR and RJ45 connectors in the perfect cue follow the same general pinout as limitimer. The communication on the unit I had access to is quite close to rs485, but not quite.

It is suggested to first try with connecting to a RS-485 adapter or hat like one would with a limitimer system. If that doesn't work try connecting perfect cue A signal to A on the RS-485 adapter and perfect cue GND to B on the adapter.

So first try with connecting A from Perfect Cue to A on the RS-485 receiver and B to B on the receiver. If this doesn't work connect DSAN A to receiber A and DSAN GND to receiver B.

The complete RS-485 differential pair signal is only available on the rj-45 (ethernet) connectors from perfect cue. The pinout of the connector is as follows:

1. +12V
2. B (D-)
3. A (D+)
4. GND
5. GND
6. A (D+)
7. B (D-)
8. +12V

With T-568 A colors:
* Orange/white + orange to A
* Green + brown/white to B.
* Green/white + Brown to +12V
* Green + Brown/white to GND

Or with T-568 B:
* Green/white + green to A
* Grange + brown/white to B.
* Orange/white + Brown to +12V
* Orange + Brown/white to GND

The XLR pinout is:
1. +12V
2. GND
3. B (D-)

It is recommended to use continuity tester while building the adapter and check that the signals aren't shorted and that the 3 common signals from the ethernet and XLR connection match before turning the system on.

## Protocol

The Perfect Cue protocol is really simple. It uses 19200-8-N-1 UART as transport. The perfect cue sends the following bytes on button press:

* 0x0F - Green right arrow aka "Next"
* 0x1F - Red left arrow aka "Previous"
* 0x2F - White crossed circle aka "Blank" - Yellow led stays off
* 0x3F - White crossed circle aka "Blank" - Yellow led stays on

Sending the following to a perfect cue in master mode triggers leds on it:

* Enter - Green right arrow
* Any arrow key - Red left arrow
* R,T,Y,U - Blink yellow led and leave on
* E,A,D - Blink yellow led and leave off
* Any other - Blink yellow led for a while (error report?)