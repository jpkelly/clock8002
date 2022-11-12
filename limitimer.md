# Limitimer integration

clock-8001 supports acting as a display for limitimer controllers or acting as a controller for limitimer display units. This functionality requires additional hardware.

## Requirements

### Hardware

The limitimer functionality requires an RS-485 adapter and a wiring adapter. For raspberry pi a RS-485 hat, like the waveshare rs-485 + can hat, is recommended. Any hat that uses the UART provided on the gpio header should work. Most USB rs-485 dongles can also be made to work, but doing so will require loading the kernel modules for it.

For raspberry pi hats the serial device to use is `/dev/ttyAMA0`

### Wiring

The complete RS-485 differential pair signal is only available on the rj-45 (ethernet) connectors from limitimer. The pinout of the connector is as follows:

1. +12V
2. B (D-)
3. A (D+)
4. GND
5. GND
6. A (D+)
7. B (D-)
8. +12V

With T-568 A colors, orange/white + orange to A and green + brown/white to B. Or with T-568 B, green/white + green to A and orange + brown/white to B.

The XLR pinout is:
1. +12V
2. GND
3. B (D-)

It is recommended to use continuity tester while building the adapter and check that the signals aren't shorted and that the 3 common signals from the ethernet and XLR connection match before turning the system on.

### Limitimer display settings when controlled by clock-8001

If you use clock-8001 to contorl limitimer display units you need to set their count direction to down on the dip switches. Other switches can be set based on desired behaviour.

## Controlling limitimer displays with clock-8001

Enable sending of data to limitimer displays on the clock config. After that timers 1-4 will be sent as limitimer programs 1-4. The first running timer of them will be sent as the "active" limitimer program.

Configure the display unit dip switches to receive the timer you want on it.


## Using clock-8001 as limitimer display

Enable receiving of limitimer data in the config. This will cause timers 1-5 to be set based on received limitimer state. Timers 1-4 are the programs 1-4 from limitimer and timer 5 is the currently selected active program from the limitimer controller.

Set the clock-8001 overtime config as desired.

Currently limitimer beeping behaviour isn't implemented.