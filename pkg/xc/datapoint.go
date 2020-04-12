package xc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
)

var errMsgNotHandled = errors.New("unhandled message")

type Datapoint struct {
	device  *Device
	name    string
	number  byte
	channel int
	mode    int
	sensor  bool
	mux     sync.Mutex
}

func (dp *Datapoint) Number() int {
	return int(dp.number)
}

func (dp *Datapoint) Device() *Device {
	return dp.device
}

func (dp *Datapoint) Name() string {
	return dp.name
}

func (dp *Datapoint) Channel() int {
	return dp.channel
}

func (dp *Datapoint) Mode() int {
	return dp.channel
}

func (dp *Datapoint) Type() channelType {
	info, exists := names[dp.device.deviceType]
	if !exists {
		log.Printf("Unknown device type %d\n", dp.device.deviceType)
		return UNKNOWN
	}
	if len(info.channels) <= dp.channel {
		log.Printf("Unknown channel %d for device %s\n", dp.channel, info.name)
		return UNKNOWN
	}
	return info.channels[dp.channel]
}

func (dp *Datapoint) rx(h Handler, data []byte) error {
	dp.device.setRssi(h, SignalStrength(data[7]))
	dp.device.setBattery(h, BatteryState(data[8]&0x1f))

	fmt.Printf("Device %d (channel %d-'%s') sent message (battery %s, signal %s) ",
		dp.device.serialNumber, dp.channel, dp.name, dp.device.battery, dp.device.rssi)

	if data[0] == RX_EVENT_STATUS {
		return dp.status(h, data[2])
	}

	event, exists := rxEventMap[data[0]]
	if !exists {
		fmt.Printf("unexpected event %d; ignoring", data[0])
		return errMsgNotHandled
	}

	return dp.event(h, event, data[1:])
}

func (dp *Datapoint) status(h Handler, status byte) error {
	fmt.Printf("status ")

	switch {
	case dp.device.IsSwitchingActuator():
		switch status {
		case RX_IS_OFF, RX_IS_OFF_NG:
			fmt.Println("switched off")
			h.StatusBool(dp, false)
		case RX_IS_ON, RX_IS_ON_NG:
			fmt.Println("switched on")
			h.StatusBool(dp, true)
		default:
			fmt.Printf("unknown status %d\n", status)
			return errMsgNotHandled
		}

	case dp.device.IsDimmingActuator():
		fmt.Println(status)
		h.StatusValue(dp, int(status))

	case dp.device.IsShutter():
		switch status {
		case RX_IS_STOP:
			fmt.Println("stop")
		case RX_IS_OPEN:
			fmt.Println("open")
		case RX_IS_CLOSE:
			fmt.Println("close")
		default:
			fmt.Printf("unknown status %d\n", status)
			return errMsgNotHandled
		}

	default:
		fmt.Printf(" unsupported device %d\n", dp.device.deviceType)
		return errMsgNotHandled
	}

	return nil
}

func (dp *Datapoint) event(h Handler, event Event, data []byte) error {
	var value interface{}

	switch data[0] {
	case RX_DATA_TYPE_RC_DATA:
		value = float32(int16(binary.BigEndian.Uint16(data[2:4]))) / 10
	case RX_DATA_TYPE_UINT16_1POINT:
		value = float32(binary.BigEndian.Uint16(data[2:4])) / 10
	case RX_DATA_TYPE_INT16_1POINT:
		value = float32(int16(binary.BigEndian.Uint16(data[2:4]))) / 10
	case RX_DATA_TYPE_UINT16_2POINT:
		value = float32(binary.BigEndian.Uint16(data[2:4])) / 100
	case RX_DATA_TYPE_UINT16_3POINT:
		value = float32(binary.BigEndian.Uint16(data[2:4])) / 1000
	case RX_DATA_TYPE_UINT32_3POINT:
		value = float32(binary.BigEndian.Uint32(data[2:6])) / 1000
	case RX_DATA_TYPE_UINT32:
		value = binary.BigEndian.Uint32(data[2:6])
	case RX_DATA_TYPE_UINT16:
		value = binary.BigEndian.Uint16(data[2:4])
	case RX_DATA_TYPE_UINT8:
		value = data[2]
	case RX_DATA_TYPE_FLOAT:
		value = math.Float32frombits(binary.BigEndian.Uint32(data[2:6]))
	case RX_DATA_TYPE_PERCENT:
		value = float32(data[2]) * 100 / 255
	case RX_DATA_TYPE_NO_DATA:
		fmt.Printf("event '%s'\n", event)
		h.Event(dp, event)
		return nil
	default:
		fmt.Printf("unhandled data type %d for event '%s'\n", data[0], event)
		return errMsgNotHandled
	}

	fmt.Printf("event '%s' with value %v\n", event, value)
	h.ValueEvent(dp, event, value)

	return nil
}
