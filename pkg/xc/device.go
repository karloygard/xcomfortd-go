package xc

import (
	"log"
)

// Device represents an xComfort device
type Device struct {
	deviceType   DeviceType
	subtype      byte
	serialNumber int
	rssi         SignalStrength
	battery      BatteryState
	iface        *Interface
	datapoints   []*Datapoint
}

func (d *Device) IsSwitchingActuator() bool {
	return d.deviceType == DT_CSAx_01 ||
		d.deviceType == DT_CSAU_0101 ||
		d.deviceType == DT_CBEU_0201
}

func (d *Device) IsDimmingActuator() bool {
	return d.deviceType == DT_CDAx_01 ||
		d.deviceType == DT_CDAx_01NG ||
		d.deviceType == DT_CAAE_01
}

func (d *Device) IsJalousie() bool {
	return d.deviceType == DT_CJAU_0101 ||
		d.deviceType == DT_CJAU_0102
}

// SerialNumber returns the serial number of the device
func (d *Device) SerialNumber() int {
	return d.serialNumber
}

func (d *Device) setRssi(rssi SignalStrength) {
	d.rssi = rssi
}

func (d *Device) setBattery(battery BatteryState) {
	d.battery = battery
}

func (d *Device) extendedStatus(data []byte) error {
	if d.deviceType != DeviceType(data[0]) {
		log.Printf("received non matching device type in extended status message %d, expected %d\n", data[0], d.deviceType)
		return nil
	}

	d.subtype = data[1]
	switch {
	case d.IsDimmingActuator():
		d.extendedStatusDimmer(data[2:])
	case d.IsSwitchingActuator():
		d.extendedStatusSwitch(data[2:])
	default:
		log.Printf("extended status message from unhandled device %d", data[0])
	}

	return nil
}
