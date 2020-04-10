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

func (dp *Datapoint) Type() channelType {
	info, exists := names[dp.device.deviceType]
	if !exists {
		return UNKNOWN
	}
	if len(info.channels) < dp.channel {
		return UNKNOWN
	}
	return info.channels[dp.channel]
}

func (dp *Datapoint) rx(h Handler, data []byte) error {
	dp.device.setRssi(h, SignalStrength(data[7]))
	dp.device.setBattery(h, BatteryState(data[8]&0x1f))

	fmt.Printf("Device %d (channel %d-'%s') sent message (battery %s, signal %s) ",
		dp.device.serialNumber, dp.channel, dp.name, dp.device.battery, dp.device.rssi)

	switch data[0] {
	case RX_EVENT_STATUS:
		dp.status(h, data[2])
	case RX_EVENT_VALUE:
		dp.value(h, data[1:])
	default:
		if event, exists := rxEventMap[data[0]]; exists {
			dp.event(h, event)
		} else {
			fmt.Println(" unexpected event; ignoring")
			return errMsgNotHandled
		}
	}

	return nil
}

func (dp *Datapoint) status(h Handler, status byte) {
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
			fmt.Println("unknown")
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
			fmt.Println("unknown")
		}

	default:
		fmt.Println(" unsupported device")
	}
}

func (dp *Datapoint) event(h Handler, event Event) {
	fmt.Println(event)
	h.Event(dp, event)
}

func (dp *Datapoint) value(h Handler, data []byte) error {
	fmt.Printf("value ")
	switch data[0] {
	case RX_DATA_TYPE_UINT16_1POINT:
		fmt.Println(float32(binary.BigEndian.Uint16(data[2:4])) / 10)
		h.Value(dp, float32(binary.BigEndian.Uint16(data[2:4]))/10)
		return nil
	default:
		return errMsgNotHandled
	}
}
