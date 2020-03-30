package xc

import (
	"context"
	"log"
)

/* New switching actuator output channels:

   0 = status
   1 = binary input
   2 = energy
   3 = power
   4 = load error */

const (
	CSAU_0101_10   byte = 0
	CSAU_0101_10I       = 1 // Binary input
	CSAU_0101_10IE      = 3 // Binary input, Energy function
	CSAU_0101_16        = 4
	CSAU_0101_16I       = 5  // Binary input
	CSAU_0101_16IE      = 7  // Binary input, Energy function
	CSAP_01XX_12E       = 14 // Energy function
)

var switchNames = map[byte]string{
	CSAU_0101_10:   "CSAU 01/01-10",
	CSAU_0101_10I:  "CSAU 01/01-10I",
	CSAU_0101_10IE: "CSAU 01/01-10IE",
	CSAU_0101_16:   "CSAU 01/01-16",
	CSAU_0101_16I:  "CSAU 01/01-16I",
	CSAU_0101_16IE: "CSAU 01/01-16IE",
	CSAP_01XX_12E:  "CSAP 01/xx-12E",
}

func switchName(subtype byte) string {
	if name, exists := switchNames[subtype]; exists {
		return name
	}
	return "unknown"
}

func (d *Datapoint) Switch(ctx context.Context, on bool) ([]byte, error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	if on {
		return d.device.iface.sendTxCommand(ctx, []byte{d.number, TX_EVENT_SWITCH, TX_EVENTDATA_ON})
	} else {
		return d.device.iface.sendTxCommand(ctx, []byte{d.number, TX_EVENT_SWITCH, TX_EVENTDATA_OFF})
	}
}

func (d *Device) extendedStatusSwitch(data []byte) {
	log.Printf("Device %d, type %s sent extended status message: value %d, temp %dC, rssi %s, battery %s\n",
		d.serialNumber, switchName(d.subtype), data[0], data[3], SignalStrength(data[7]), BatteryState(data[8]))

	switch d.subtype {
	case CSAU_0101_10:
	case CSAU_0101_10I:
	case CSAU_0101_10IE:
	case CSAU_0101_16:
	case CSAU_0101_16I:
	case CSAU_0101_16IE:
	case CSAP_01XX_12E:
	}
}
