package xc

import (
	"encoding/binary"
)

func (i *Interface) Serial() (uint32, error) {
	data, err := i.sendConfigCommand([]byte{CONF_SERIAL, CF_DATA_GET})
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(data[1:]), nil
}

func (i *Interface) SetOKMRF() error {
	_, err := i.sendConfigCommand([]byte{CONF_SEND_OK_MRF, CF_DATA_SET})
	return err
}

func (i *Interface) SetRfSeqNo() error {
	_, err := i.sendConfigCommand([]byte{CONF_SEND_RFSEQNO, CF_DATA_SET})
	return err
}

func (i *Interface) GetCounterRx() (uint32, error) {
	data, err := i.sendConfigCommand([]byte{CONF_COUNTER_RX, CF_DATA_GET})
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(data[1:]), nil
}

func (i *Interface) GetCounterTx() (uint32, error) {
	data, err := i.sendConfigCommand([]byte{CONF_COUNTER_TX, CF_DATA_GET})
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(data[1:]), nil
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
