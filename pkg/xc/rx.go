package xc

import (
	"encoding/binary"
	"log"
)

func (i *Interface) rx(data []byte) error {
	if data[1] == RX_EVENT_STATUS_EXT {
		i.extendedStatus(data[2:])
	} else {
		if dp, found := i.datapoints[data[0]]; found {
			return dp.rx(i.handler, data[1:])
		} else {
			log.Printf("Received message from unknown datapoint %d", data[0])
		}
	}
	return nil
}

func (i *Interface) extendedStatus(data []byte) {
	switch data[0] {
	case RX_DATA_TYPE_SERIAL_NUMBER:
		serial := int(binary.LittleEndian.Uint32(data[2:6]))
		if device, found := i.devices[serial]; found {
			device.extendedStatus(i.handler, data[6:])
		} else {
			log.Printf("Received extended status message from unknown device %d", serial)
		}
	default:
		log.Println("Unhandled extended status message")
	}
}
