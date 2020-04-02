package xc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
)

var errMsgNotHandled = errors.New("unhandled message")

type Datapoint struct {
	device  *Device
	name    string
	number  byte
	channel int
	mux     sync.Mutex
}

func (dp *Datapoint) Number() int {
	return int(dp.number)
}

func (dp *Datapoint) rx(h Handler, data []byte) error {
	dp.device.setRssi(SignalStrength(data[7]))
	dp.device.setBattery(BatteryState(data[8] & 0x1f))

	fmt.Printf("Device %d (channel %d-'%s') sent message (battery %s, signal %s) ",
		dp.device.serialNumber, dp.channel, dp.name, dp.device.battery, dp.device.rssi)

	switch data[0] {
	case RX_EVENT_STATUS:
		dp.status(h, data[2])
	case RX_EVENT_ON:
		dp.on()
	case RX_EVENT_OFF:
		dp.off()
	case RX_EVENT_UP_PRESSED:
		dp.upPressed()
	case RX_EVENT_UP_RELEASED:
		dp.upReleased()
	case RX_EVENT_DOWN_PRESSED:
		dp.downPressed()
	case RX_EVENT_DOWN_RELEASED:
		dp.downReleased()
	case RX_EVENT_VALUE:
		dp.value(data[1:])
	case RX_EVENT_SWITCH_ON,
		RX_EVENT_SWITCH_OFF,
		RX_EVENT_FORCED,
		RX_EVENT_SINGLE_ON,
		RX_EVENT_TOO_WARM,
		RX_EVENT_TOO_COLD,
		RX_EVENT_BASIC_MODE:
		return errMsgNotHandled
	}

	return nil
}

func (dp *Datapoint) status(h Handler, status byte) {
	fmt.Printf("status ")

	switch dp.device.deviceType {
	case DT_CSAx_01, DT_CSAU_0101, DT_CBEU_0201:
		switch status {
		case RX_IS_OFF, RX_IS_OFF_NG:
			fmt.Println("switched off")
			h.StatusBool(dp, false)
		case RX_IS_ON, RX_IS_ON_NG:
			fmt.Println("switched on")
			h.StatusBool(dp, true)
		default:
			fmt.Println("unknown")
		}

	case DT_CDAx_01, DT_CDAx_01NG, DT_CAAE_01:
		fmt.Println(status)
		h.StatusValue(dp, int(status))

	case DT_CJAU_0101, DT_CJAU_0102:
		switch status {
		case RX_IS_STOP:
			fmt.Println("stop")
		case RX_IS_OPEN:
			fmt.Println("open")
		case RX_IS_CLOSE:
			fmt.Println("close")
		default:
			fmt.Println("unknown")
		}

	default:
		fmt.Println(" unsupported device")
	}
}

func (dp *Datapoint) on() {
	fmt.Println("ON")
}

func (dp *Datapoint) off() {
	fmt.Println("OFF")
}

func (dp *Datapoint) upPressed() {
	fmt.Println("UP PRESSED")
}

func (dp *Datapoint) upReleased() {
	fmt.Println("UP RELEASED")
}

func (dp *Datapoint) downPressed() {
	fmt.Println("DOWN PRESSED")
}

func (dp *Datapoint) downReleased() {
	fmt.Println("DOWN RELEASED")
}

func (dp *Datapoint) value(data []byte) error {
	fmt.Printf("value ")
	switch data[0] {
	case RX_DATA_TYPE_NO_DATA:
	case RX_DATA_TYPE_PERCENT:
	case RX_DATA_TYPE_UINT8:
	case RX_DATA_TYPE_INT16_1POINT:
	case RX_DATA_TYPE_FLOAT:
	case RX_DATA_TYPE_UINT16:
	case RX_DATA_TYPE_UINT16_1POINT:
		fmt.Println(float32(binary.BigEndian.Uint16(data[2:4])) / 10)
	case RX_DATA_TYPE_UINT16_2POINT:
	case RX_DATA_TYPE_UINT16_3POINT:
	case RX_DATA_TYPE_UINT32:
	case RX_DATA_TYPE_UINT32_1POINT:
	case RX_DATA_TYPE_UINT32_2POINT:
	case RX_DATA_TYPE_UINT32_3POINT:
	case RX_DATA_TYPE_RC_DATA:
	case RX_DATA_TYPE_RM_TIME:
	case RX_DATA_TYPE_RM_DATE:
	case RX_DATA_TYPE_ROSETTA:
	case RX_DATA_TYPE_HRV_OUT:
	case RX_DATA_TYPE_SERIAL_NUMBER:
	}
	return nil
}
