package xc

import (
	"log"
)

type Device struct {
	deviceType   DeviceType
	subtype      byte
	serialNumber int
	rssi         SignalStrength
	battery      BatteryState
	iface        *Interface

	Datapoints []*Datapoint
}

func (d *Device) extendedStatus(data []byte) error {
	if d.deviceType != DeviceType(data[0]) {
		log.Printf("received non matching device type in extended status message %d, expected %d\n", data[0], d.deviceType)
		return nil
	}

	d.subtype = data[1]
	switch d.deviceType {
	case DT_CDAx_01NG:
		d.extendedStatusDimmer(data[2:])
	case DT_CSAU_0101:
		d.extendedStatusSwitch(data[2:])
	default:
		log.Printf("extended status message from unhandled device %d", data[0])
	}

	return nil
}
