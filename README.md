xComfort gateway
================

This code implements communication with the Eaton xComfort CKOZ-00/14
USB and CCIA-0x/01 Ethernet Communication Interfaces (CI devices).
The code can talk to multiple CI devices in parallel, whether that be
one or more connected USB devices or multiple ECI devices.

Datapoints can be read out from the eprom on the devices, which must
be kept updated *manually* if and when devices are added.  Consult the
MRF manual (paragraph USB-RF-Communication Stick) for documentation on
how to do this.  For testing purposes, both TXT and DPL file formats
are supported, but the latter format is generally superior.

To build:

    go build .

Typical invocation:

    ./xcomfortd-go -v -e -s tcp://user:password@mqtthost:1883

A prepackaged addon for Home Assistant is available at https://github.com/karloygard/hassio-addons

xComfort is a wireless European home automation system, using the
868,3MHz band.  The system is closed source.  This code was reverse
engineered from a variety of sources, without documentation from Eaton,
and may not follow their specifications.

This code supports both extended and regular status messages.  Older
devices only send the latter, which are not routed and have no
delivery guarantees.  Careful placement of the CI is important,
so that it can see these messages, or you can use more than one CI
to improve coverage.

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

Copyright 2021 Karl Anders Øygard and collaborators.  All rights reserved.
Use of this source code is governed by a BSD-style license that can be
found in the LICENSE file.
