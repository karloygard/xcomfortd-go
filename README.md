xComfort gateway
================

This code implements communication with the Eaton xComfort CKOZ-00/14
Communication stick (CI stick).  You need to export the datapoints file
with the CKOZ-00/13 USB stick and the associated MRF software.  Consult
the MRF manual (paragraph USB-RF-Communication Stick) for documentation
on how to do this.  The format must be TXT.

To build:

    go build .

Typical invocation:

    ./xcomfortd-go usb -v -f datapoints.txt -i xcomfortd -s tcp://user:password@mqtthost:1883

xComfort is a wireless European home automation system, using the
868,3MHz band.  The system is closed source.  This code was reverse
engineered from a variety of sources, without documentation from Eaton,
and may not follow their specifications.  If this code damages your
devices, it's on you.

This code supports both extended and regular status messages.  Older
devices only send the latter, which are not routed and have no
delivery guarantees.  Careful placement of the USB stick is important,
so that it can see these messages.

A simple application for forwarding events to and from an MQTT server is
provided.  This can be used eg. to interface an xComfort installation with
[Home Assistant](https://home-assistant.io/), with a little imagination.
The application subscribes to the topics:

    "xcomfort/+/set/dimmer" (accepts values from 0-100)
    "xcomfort/+/set/switch" (accepts true or false)

and publishes on the topics:

    "xcomfort/[datapoint number]/get/dimmer" (value from 0-100)
    "xcomfort/[datapoint number]/get/switch" (true or false)

Sending `true` to topic `xcomfort/1/set/switch` will send a message to
datapoint 1 to turn on.  This will work for both switches and dimmers.
Sending the value `50` to `xcomfort/1/set/dimmer` will send a message
to datapoint 1 to set 50% dimming.  This will work only for dimmers.

Likewise, `xcomfort/1/get/dimmer` and `xcomfort/1/get/switch` will be
set to the value reported by the dimmer/switch, if and when datapoint
1 reports changes.  Subscribe to the topic that's relevant for the
device that's actually associated with the datapoint.

Copyright 2020 Karl Anders Øygard.  All rights reserved.  Use of this
source code is governed by a BSD-style license that can be found in
the LICENSE file.
