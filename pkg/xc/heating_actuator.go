package xc

import (
	"context"
	"encoding/binary"
	"log"
)

/* New heating actuator output channels:

   0 = status
   2 = energy
   3 = load error */

const (
	CHAU_0101_10E byte = 0
	CHAU_0101_16E      = 1
	CHAU_0101_1ES      = 2
	CHAP_01x5_12E      = 3
	CHAP_01x5_1ES      = 4
)

var heatingNames = map[byte]string{
	CHAU_0101_10E: "CHAU 01/01-10E",
	CHAU_0101_16E: "CHAU 01/01-16E",
	CHAU_0101_1ES: "CHAU 01/01-1ES",
	CHAP_01x5_12E: "CHAP 01/x5-12E",
	CHAP_01x5_1ES: "CHAP 01/x5-1ES",
}

func heatingActuatorName(subtype byte) string {
	if name, exists := heatingNames[subtype]; exists {
		return name
	}
	return "unknown"
}

func (d *Device) extendedStatusHeatingActuator(h Handler, data []byte) {
	dutyCycle := (float32(data[0]) * 100.0 / 255.0)
	power := float32(binary.LittleEndian.Uint16(data[1:3])) / 10

	internalTemperature := data[3]

	d.setBattery(h, BatteryState(data[6]))
	d.setRssi(h, SignalStrength(data[5]))

	h.InternalTemperature(d, int(internalTemperature))

	log.Printf("Device %d, type %s sent extended status message: duty "+
		"cycle %.1f%%, temp %dC, power %.1fW (battery %s, signal %s)\n",
		d.serialNumber, heatingActuatorName(d.subtype), dutyCycle,
		internalTemperature, power, d.battery, d.rssi)

	for _, dp := range d.datapoints {
		if dp.channel == 0 {
			// Status channel is always 0
			if dutyCycle > 0 {
				h.Value(dp, "heat")
			} else {
				h.Value(dp, "off")
			}
		}
	}
}

func (d *Datapoint) DesiredTemperature(ctx context.Context,
	value float32) ([]byte, error) {

	last := d.queue.Lock()
	defer d.queue.Unlock()

	data := make([]byte, 2)

	if !last {
		// There are newer commands, discard
		return nil, nil
	}

	binary.BigEndian.PutUint16(data, uint16(value*10))

	return d.device.iface.sendTxCommand(ctx, []byte{
		d.number,
		MCI_TE_DIMPLEX_CONFIG,
		data[0],
		data[1],
		MCI_TED_DPLMODE_OFFICE,
	})
}

func (d *Datapoint) CurrentTemperature(ctx context.Context,
	value float32) ([]byte, error) {

	last := d.queue.Lock()
	defer d.queue.Unlock()

	data := make([]byte, 2)

	if !last {
		// There are newer commands, discard
		return nil, nil
	}

	binary.BigEndian.PutUint16(data, uint16(value*10))

	return d.device.iface.sendTxCommand(ctx, []byte{
		d.number,
		MCI_TE_DIMPLEX_TEMP,
		data[0],
		data[1],
		0,
		0,
	})
}
