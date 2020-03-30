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

func (d *Device) ExtendedStatus(data []byte) error {
	if d.deviceType != DeviceType(data[0]) {
		log.Printf("received non matching device type in extended status message %d, expected %d\n", data[0], d.deviceType)
	} else {
		d.subtype = data[1]
		log.Printf("Device %d, type %v/%d sent extended status message: value %d, temp %dC, rssi %d, battery %d\n",
			d.serialNumber, d.deviceType, data[1], data[2], data[5], data[9], data[10])
		switch d.subtype {
		case 0:
		case 1:
		case 2:
		case 4:
		case 6:
		case 9:
		case 10:
		}
	}
	return nil
}

/*unsigned char  device_type;
unsigned char  variant;
unsigned char  state;
unsigned char  output_value;
unsigned char  binary_inputs;
unsigned char  temperature;
short int      power;
unsigned char  load_error;
unsigned char  rssi;            // range 0 - 120
unsigned char  battery;         // see BatteryStatus
unsigned char  seqno;           // 0-15 monotonously increasing*/
