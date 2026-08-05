# Disguise

## OSC syntax

https://help.disguise.one/en/Content/Configuring/Transports/OSC/OSC-syntax-library.html

### Outputs

The documentation leaves much to be desired...

* /d3/showcontrol/heartbeat has a float parameter from the range of 0.0 to 1.0
* /d3/showcontrol/trackposition time from the start of the track as string "00:09:20.00"
* /d3/showcontrol/trackname track name as string
* /d3/showcontrol/currentsectionname haven't seen anything other than a blank string
* /d3/showcontrol/nextsectionname same as above
* /d3/showcontrol/sectionhint
  * Weird one string containing several fields...
  * Fields are space separated
  * First is the name of previous cue/note/start
  * Time from last cue/note/start is "+00:00:00.00"
  * The name of next cue/note/end
  * Time to next cue/note/end is "-00:00:00.00"
* /d3/showcontrol/volume
* /d3/showcontrol/brightness
* /d3/showcontrol/bpm
* /d3/showcontrol/playmode
  * On the version I had access to this defaulted to d3/showcontrol/playmode
  * Several possible values
    * Play
    * PlaySection
    * LoopSection
    * Stop
    * HoldSection
    * HoldEnd

## OSC Setup

1. Device manager -> add device -> osc device
2. Set ports and destination IP address
3. TransportManager -> Add local / remote transport (what is the difference?)
4. Set the oscTransport to use the correct OSC device in its properties
5. Make sure the OSC message names are correct