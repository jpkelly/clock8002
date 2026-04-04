# Limitimer integration

clock-8001 supports acting as a display for limitimer controllers or acting as a controller for limitimer display units. This functionality requires additional hardware.

## Requirements

### Hardware

The limitimer functionality requires an RS-485 adapter and a wiring adapter. For raspberry pi a RS-485 hat, like the waveshare rs-485 + can hat, is recommended. Any hat that uses the UART provided on the gpio header should work. Most USB rs-485 dongles can also be made to work, but doing so will require loading the kernel modules for it.

For raspberry pi hats the serial device to use is `/dev/ttyAMA0`

### Wiring

See the [piClock wiring diagram (PDF)](docs/piClockWiring.pdf) for an overview of all connections.

The complete RS-485 differential pair signal is only available on the rj-45 (ethernet) connectors from limitimer. The pinout of the connector is as follows:

1. +12V
2. B (D-)
3. A (D+)
4. GND
5. GND
6. A (D+)
7. B (D-)
8. +12V

With T-568 A colors:
* Orange + orange/white to A
* Green + brown/white to B.
* Brown + green/white to +12V
* Blue + blue/white to GND

Or with T-568 B:
* Green + green/white to A
* Orange + brown/white to B.
* Brown + orange/white to +12V
* Blue + blue/white to GND

The XLR pinout is:
1. +12V
2. GND
3. B (D-)

It is recommended to use continuity tester while building the adapter and check that the signals aren't shorted and that the 3 common signals from the ethernet and XLR connection match before turning the system on.

#### RS-485 line termination resistors

RS-485 spesification requires 120 ohm termination resistors at the ends of the bus. This is simply just a resistor between A and B lines. DSAN has done some additional engineering in their RS-485 receivers and limitimer itself doesn't require the terminators, but many RS-485 receivers will need them. Usually two devices on the bus is fine no matter what the devices are without termination, but after that the clock-8001 might not be able to send or receive to limitimer units without at least one end of the chain having the 120 ohm terminator resistor.

### Limitimer display settings when controlled by clock-8001

If you use clock-8001 to contorl limitimer display units you need to set their count direction to down on the dip switches. Other switches can be set based on desired behaviour.

## Controlling limitimer displays with clock-8001

Enable sending of data to limitimer displays on the clock config. After that timers 1-4 will be sent as limitimer programs 1-4. The first running timer of them will be sent as the "active" limitimer program.

Configure the display unit dip switches to receive the timer you want on it.


## Using clock-8001 as limitimer display

Enable receiving of limitimer data in the config. This will cause timers 1-5 to be set based on received limitimer state. Timers 1-4 are the programs 1-4 from limitimer and timer 5 is the currently selected active program from the limitimer controller.

Set the clock-8001 overtime config as desired.

Currently limitimer beeping behaviour isn't implemented.