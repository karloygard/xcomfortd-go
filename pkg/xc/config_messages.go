package xc

import (
	"encoding/binary"
)

func (i *Interface) Serial() (uint32, error) {
	data, err := i.sendConfigCommand([]byte{CONF_SERIAL, CF_DATA_GET})
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(data[1:]), err
}

func (i *Interface) Release() (rf, fw float32, err error) {
	data, err := i.sendConfigCommand([]byte{CONF_RELEASE, CF_DATA_GET})
	if err != nil {
		return 0, 0, err
	}

	rf = float32(data[1]) + float32(data[2])/100.0
	fw = float32(data[3]) + float32(data[4])/100.0

	return
}

func (i *Interface) Revision() (hw, rf, fw int, err error) {
	data, err := i.sendConfigCommand([]byte{CONF_RELEASE, CF_DATA_GET_REVISION})
	if err != nil {
		return 0, 0, 0, err
	}

	hw = int(data[1])
	rf = int(data[2])
	fw = int(binary.BigEndian.Uint16(data[3:5]))

	return
}
