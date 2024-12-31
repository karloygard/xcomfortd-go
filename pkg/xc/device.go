package xc

import (
	"log"
	"strconv"
)

// Device represents an xComfort device
type Device struct {
	deviceType   DeviceType
	subtype      byte
	serialNumber int
	name         string
	rssi         SignalStrength
	battery      BatteryState
	iface        *Interface
	datapoints   []*Datapoint
}

func (d Device) IsSwitchingActuator() bool {
	return d.deviceType == DT_CSAx_01 ||
		d.deviceType == DT_CSAU_0101 ||
		d.deviceType == DT_CBEU_0201
}

func (d Device) IsHeatingActuator() bool {
	return d.deviceType == DT_CHAX_010x
}

func (d Device) IsDimmingActuator() bool {
	return d.deviceType == DT_CDAx_01 ||
		d.deviceType == DT_CDAx_01NG ||
		d.deviceType == DT_CAAE_01
}

func (d Device) IsShutter() bool {
	return d.deviceType == DT_CJAU_0101 ||
		d.deviceType == DT_CJAU_0102 ||
		d.deviceType == DT_CJAU_0104
}

func (d Device) IsBatteryOperated() bool {
	return d.deviceType == DT_CTAA_01 ||
		d.deviceType == DT_CTAA_02 ||
		d.deviceType == DT_CTAA_04 ||
		d.deviceType == DT_CRCA_000x ||
		d.deviceType == DT_CTEU_02 ||
		d.deviceType == DT_CBEU_0202 ||
		d.deviceType == DT_CHSZ_1201 ||
		d.deviceType == DT_CHSZ_02 ||
		d.deviceType == DT_CHSZ_01 ||
		d.deviceType == DT_CHSZ_1203 ||
		d.deviceType == DT_CHSZ_1204 ||
		d.deviceType == DT_CRCA_00 ||
		d.deviceType == DT_CBMA_02 ||
		d.deviceType == DT_CRCA_00xx
}

func (d Device) ReportsPower() bool {
	switch {
	case d.deviceType == DT_CDAx_01NG:
		return true
	case d.deviceType == DT_CSAU_0101:
		return true
	case d.IsHeatingActuator():
		return true
	}

	return false
}

func (d Device) Type() DeviceType {
	return d.deviceType
}

func (d Device) SerialNumber() int {
	return d.serialNumber
}

func (d Device) Name() string {
	if d.name == "" {
		return strconv.Itoa(d.SerialNumber())
	}
	return d.name
}

func matchingString(a, b string) string {
	c := []rune(a)
	d := []rune(b)

	length := len(c)
	if length > len(d) {
		length = len(d)
	}

	for i := 0; i < length; i++ {
		if c[i] != d[i] {
			return string(c[:i])
		}
	}

	return string(c[:length])
}

func (d *Device) setName(name string) {
	if d.name == "" {
		d.name = name
	} else {
		d.name = matchingString(d.name, name)
	}
}

func (d *Device) setRssi(h Handler, rssi SignalStrength) {
	d.rssi = rssi
	h.Rssi(d, int(rssi))
}

func (d *Device) setBattery(h Handler, battery BatteryState) {
	d.battery = battery
	h.Battery(d, battery.percentage())
}

func (d *Device) extendedStatus(h Handler, data []byte) error {
	if d.deviceType != DeviceType(data[0]) {
		log.Printf("received non matching device type in extended status message %d, expected %d\n", data[0], d.deviceType)
		return errMsgNotHandled
	}

	d.subtype = data[1]
	switch {
	case d.IsDimmingActuator():
		d.extendedStatusDimmer(h, data[2:])
	case d.IsSwitchingActuator():
		d.extendedStatusSwitch(h, data[2:])
	case d.IsHeatingActuator():
		d.extendedStatusHeatingActuator(h, data[2:])
	case d.IsShutter():
		d.extendedStatusShutter(h, data[2:])
	default:
		log.Printf("Device type: %s", d.deviceType)
		log.Printf("extended status message from unhandled device %d", data[0])
		return errMsgNotHandled
	}

	return nil
}
