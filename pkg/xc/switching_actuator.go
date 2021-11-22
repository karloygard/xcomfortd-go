package xc

import (
	"context"
	"encoding/binary"
	"log"
)

/* New switching actuator output channels:

   0 = status
   1 = binary input (RX_EVENT_UP_PRESSED/UP_RELEASED/SINGLE_ON)
   2 = energy (TX_EVENT_UINT32_3POINT)
   3 = power (TX_EVENT_UINT16_1POINT)
   4 = load error (RX_EVENT_SWITCH_ON/OFF)  */

const (
	CSAU_0101_10   = 0
	CSAU_0101_10I  = 1 // Binary input
	CSAU_0101_10IE = 3 // Binary input, Energy function
	CSAU_0101_16   = 4
	CSAU_0101_16I  = 5  // Binary input
	CSAU_0101_16IE = 7  // Binary input, Energy function
	CSAP_01XX_12E  = 14 // Energy function
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

const (
	CSAX_OFF                 = 0x1
	CSAX_ON                  = 0x2
	CSAX_BLINKING            = 0x3
	CSAX_ON_LOCKED           = 0x4
	CSAX_OFF_LOCKED          = 0x5
	CSAX_OFF_OVERTEMPERATURE = 0x7
	CSAX_OFF_OVERLOAD        = 0x8
)

func extendedOutputStatusName(status byte) string {
	switch status {
	case CSAX_OFF:
		return "OFF"
	case CSAX_ON:
		return "ON"
	case CSAX_BLINKING:
		return "BLINKING"
	case CSAX_ON_LOCKED:
		return "ON-LOCKED"
	case CSAX_OFF_LOCKED:
		return "OFF-LOCKED"
	case CSAX_OFF_OVERTEMPERATURE:
		return "OFF-OVERTEMPERATURE"
	case CSAX_OFF_OVERLOAD:
		return "OFF-OVERLOAD"
	default:
		return "UNKNOWN"
	}
}

func (d *Datapoint) Switch(ctx context.Context, on bool) ([]byte, error) {
	last := d.queue.Lock()
	defer d.queue.Unlock()

	if !last {
		// There are newer commands, discard
		return nil, nil
	}

	if on {
		return d.device.iface.sendTxCommand(ctx, []byte{d.number, MCI_TE_SWITCH, MCI_TED_ON})
	} else {
		return d.device.iface.sendTxCommand(ctx, []byte{d.number, MCI_TE_SWITCH, MCI_TED_OFF})
	}
}

func (d *Device) extendedStatusSwitch(h Handler, data []byte) {
	status := data[0] >> 4
	statusName := extendedOutputStatusName(status)

	//binaryInput := data[1]
	internalTemperature := data[1]

	d.setBattery(h, BatteryState(data[6]))
	d.setRssi(h, SignalStrength(data[5]))

	h.InternalTemperature(d, int(internalTemperature))

	if d.subtype == CSAU_0101_16IE ||
		d.subtype == CSAU_0101_10IE ||
		d.subtype == CSAP_01XX_12E {
		power := float32(binary.LittleEndian.Uint16(data[2:4])) / 10
		h.Power(d, power)

		log.Printf("Device %d, type %s sent extended status message: status %s, temp %dC, power %.1fW (battery %s, signal %s)\n",
			d.serialNumber, switchName(d.subtype), statusName, internalTemperature, power, d.battery, d.rssi)
	} else {
		log.Printf("Device %d, type %s sent extended status message: status %s, temp %dC (battery %s, signal %s)\n",
			d.serialNumber, switchName(d.subtype), statusName, internalTemperature, d.battery, d.rssi)
	}

	for _, dp := range d.datapoints {
		if dp.channel == 0 {
			// Status channel is always 0
			switch status {
			case CSAX_OFF, CSAX_OFF_LOCKED, CSAX_OFF_OVERTEMPERATURE, CSAX_OFF_OVERLOAD:
				h.StatusBool(dp, false)
			case CSAX_ON, CSAX_ON_LOCKED, CSAX_BLINKING:
				h.StatusBool(dp, true)
			default:
				log.Printf("unknown status %d for switching actuator; ignoring\n", status)
			}
		}
	}
}
