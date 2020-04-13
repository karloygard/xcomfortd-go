package xc

import (
	"context"
	"encoding/binary"
	"log"
)

/* New dimming actuator output channels:

   0 = status
   1 = binary input A
   2 = binary input B
   3 = energy
   4 = power
   5 = load error */

const (
	CDAU_0104   byte = 0
	CDAU_0104_I      = 1 // 2 x binary input
	CDAU_0104_E      = 2 // Energy function
	CDAE_0104        = 4
	CDAE_0104_E      = 6  // Energy function
	CDAE_0105_I      = 9  // 2 x binary input
	CDAE_0105_E      = 10 // Energy function
)

var dimmerNames = map[byte]string{
	CDAU_0104:   "CDAU 01/04",
	CDAU_0104_I: "CDAU 01/04-I",
	CDAU_0104_E: "CDAU 01/04-E",
	CDAE_0104:   "CDAE 01/04",
	CDAE_0104_E: "CDAE 01/04-E",
	CDAE_0105_I: "CDAE 01/05-I",
	CDAE_0105_E: "CDAE 01/05-E",
}

func dimmerName(subtype byte) string {
	if name, exists := dimmerNames[subtype]; exists {
		return name
	}
	return "unknown"
}

func (d *Datapoint) Dim(ctx context.Context, value int) ([]byte, error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	return d.device.iface.sendTxCommand(ctx, []byte{d.number, TX_EVENT_DIM, TX_EVENTDATA_PERCENT, byte(value)})
}

func (d *Device) extendedStatusDimmer(h Handler, data []byte) {
	value := data[1]
	//binaryA := data[2] >> 4
	//binaryB := data[2] & 0xf
	internalTemperature := data[3]

	d.setBattery(h, BatteryState(data[8]))
	d.setRssi(h, SignalStrength(data[7]))

	h.InternalTemperature(d, int(internalTemperature))

	switch d.subtype {
	case CDAU_0104_E, CDAE_0104_E, CDAE_0105_E:
		power := float32(binary.LittleEndian.Uint16(data[4:6])) / 10
		log.Printf("Device %d, type %s sent extended status message: value %d, temp %dC, power %.1fW (battery %s, signal %s)\n",
			d.serialNumber, dimmerName(d.subtype), value, internalTemperature, power, d.battery, d.rssi)
	default:
		log.Printf("Device %d, type %s sent extended status message: value %d, temp %dC (battery %s, signal %s)\n",
			d.serialNumber, dimmerName(d.subtype), value, internalTemperature, d.battery, d.rssi)
	}

	for _, dp := range d.datapoints {
		if dp.channel == 0 {
			// Status channel is always 0
			h.StatusValue(dp, (int(value)*100)/255)
			break
		}
	}
}
