package xc

import (
	"encoding/binary"
	"log"
)

func (i *Interface) rx(data []byte) error {
	if data[1] == RX_EVENT_STATUS_EXT {
		return i.extendedStatus(data[2:])
	}

	if dp, found := i.datapoints[data[0]]; found {
		return dp.rx(i.handler, data[1:])
	}

	log.Printf("Received message from unknown datapoint %d", data[0])
	return errMsgNotHandled
}

func (i *Interface) extendedStatus(data []byte) error {
	switch data[0] {
	case RX_DATA_TYPE_SERIAL_NUMBER:
		serial := int(binary.LittleEndian.Uint32(data[2:6]))
		if device, found := i.devices[serial]; found {
			return device.extendedStatus(i.handler, data[6:])
		} else {
			log.Printf("Received extended status message from unknown device %d", serial)
			return errMsgNotHandled
		}
	default:
		log.Println("Unhandled extended status message")
		return errMsgNotHandled
	}
}
