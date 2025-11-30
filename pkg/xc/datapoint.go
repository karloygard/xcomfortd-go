package xc

import (
	"context" 

	"encoding/binary"
	"fmt"
	"log"
	"math"
)

type Datapoint struct {
	device  *Device
	name    string
	number  byte
	channel int
	mode    int
	sensor  bool
	queue   Queue
}

func (dp *Datapoint) Number() int {
	return int(dp.number)
}

func (dp *Datapoint) Device() *Device {
	return dp.device
}

func (dp *Datapoint) Id() string {
	return fmt.Sprintf("%d_ch%d", dp.Device().SerialNumber(), dp.Channel())
}

func (dp *Datapoint) Name() string {
	return dp.name
}

func (dp *Datapoint) fullname() string {
	if dp.name == "" {
		return dp.device.Name()
	}

	return fmt.Sprintf("%s (%s)", dp.device.Name(), dp.name)
}

func (dp *Datapoint) Channel() int {
	return dp.channel
}

func (dp *Datapoint) Mode() int {
	return dp.mode
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

func (dp *Datapoint) rx(h Handler, data []byte, ctx context.Context) (err error) {
	description := "unknown"

	dp.device.setRssi(h, SignalStrength(data[7]))
	dp.device.setBattery(h, BatteryState(data[8]&0x1f))

	cyclic := (data[8] & 0x20) == 0x20

	if data[0] == RX_EVENT_STATUS {
		description, err = dp.status(h, data[2])
	} else {
		event, exists := rxEventMap[data[0]]
		if !exists {
			log.Printf("unexpected event %d; ignoring", data[0])
			err = errMsgNotHandled
		} else {
			description, err = dp.event(h, event, data[1:], ctx)
		}
	}
	log.Printf("Device %d (channel %d-'%s') sent message (battery %s, signal %s, cyclic %v) %s",
		dp.device.serialNumber, dp.channel, dp.fullname(),
		dp.device.battery, dp.device.rssi, cyclic, description)

	return err
}

func (dp *Datapoint) status(h Handler, status byte) (string, error) {
	switch {
	case dp.device.IsSwitchingActuator():
		switch status {
		case RX_IS_OFF, RX_IS_OFF_NG:
			h.StatusBool(dp, false)
			return "status switched off", nil
		case RX_IS_ON, RX_IS_ON_NG:
			h.StatusBool(dp, true)
			return "status switched on", nil
		default:
			log.Printf("unknown switching actuator status %d\n", status)
		}

	case dp.device.IsDimmingActuator():
		h.StatusValue(dp, int(status))
		return fmt.Sprintf("value %d\n", status), nil

	case dp.device.IsShutter():
		return dp.shutterStatus(h, status)

	default:
		log.Printf("unknown status %d for unsupported device %d\n", status, dp.device.deviceType)
	}

	return "unknown", errMsgNotHandled
}

func (dp *Datapoint) event(h Handler, event Event, data []byte, ctx context.Context) (string, error) {
	var value any

	switch data[0] {
	case RX_DATA_TYPE_RC_DATA:
		value = float32(int16(binary.BigEndian.Uint16(data[2:4]))) / 10
		wheel := float32(int16(binary.BigEndian.Uint16(data[4:6]))) / 10
		h.Wheel(dp, wheel)
		log.Printf("wheel position: %.1f", wheel)
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
	case RX_DATA_TYPE_RCT_OUT:
		moisture := float32(binary.LittleEndian.Uint16(data[2:4])) / 10
		temperature := float32(binary.LittleEndian.Uint16(data[4:6])) / 10
		log.Printf("(partially decoded) temp %.1fC moisture %.1f%%", temperature, moisture)
		return "RCT OUT", errMsgNotHandled
	case RX_DATA_TYPE_RCT_REQ:
		return "RCT REQ", errMsgNotHandled
	case RX_DATA_TYPE_NO_DATA:
		h.Event(dp, event)
		return fmt.Sprintf("event '%s'\n", event), nil
	case RX_DATA_TYPE_HRV_OUT:
		status := data[2]
		if (status & MGW_HRV_ERROR_CONNECTION_LOST) != 0 {
			log.Printf("Connection lost")
		}
		if (status & MGW_HRV_ERROR_VALVE_SLUGGISH) != 0 {
			log.Printf("Valve sluggish")
		}
		if (status & MGW_HRV_ERROR_VALVE_RANGE_TOO_LARGE) != 0 {
			log.Printf("Valve range too large")
		}
		if (status & MGW_HRV_ERROR_VALVE_RANGE_TOO_SMALL) != 0 {
			log.Printf("Valve range too small")
		}
		if (status & MGW_HRV_ERROR_BATTERY_EMPTY) != 0 {
			log.Printf("Battery almost empty, valve entered save position (50%% open)")
		}
		if (status & MGW_HRV_STATUS_DEEP_SLEEP) != 0 {
			log.Printf("Device in deep sleep")
		}

		h.Valve(dp, int(data[3]))

		currTemp := (int(data[4]&0xf)*256 + int(data[5])) 
		// save current temperature as last known value
		dp.device.setCurrentTemperature(currTemp)
		value = float32(currTemp) / 10

		switch data[4] >> 4 {
		case MGW_HRV_REQ_NOTHING:
		case MGW_HRV_REQ_TSETPOINT:
			go func() {
				if dp.device.iface.verbose {
					log.Printf("Requesting temperature setpoint for device %d DP: %v", dp.device.serialNumber, dp.number)
				}
				targetTemp := float32(dp.device.TargetTemperature()) /10
				if _, err := dp.DesiredTemperature(ctx, targetTemp); err != nil {
					log.Printf("WARNING: command for device %d failed, state now unknown: %v", dp.device.serialNumber, err)
				} else {
					if dp.device.iface.verbose {
						log.Printf("Target temp successfully set")
					}			
				}
			}()				
		case MGW_HRV_REQ_TIME:
			log.Printf("Requesting time")
		case MGW_HRV_REQ_DATE:
			log.Printf("Requesting date")
		}

	default:
		log.Printf("unhandled data type %d for event '%s'", data[0], event)
		return "unknown", errMsgNotHandled
	}

	h.ValueEvent(dp, event, value)

	return fmt.Sprintf("event '%s' with value %v", event, value), nil
}

// SetTargetTemperature sets the target temperature for the device
func (dp *Datapoint) SetTargetTemperature(temp int) {
    dp.device.setTargetTemperature(temp)
}
