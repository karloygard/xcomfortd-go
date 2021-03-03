package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/karloygard/xcomfortd-go/pkg/xc"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

func (r *MqttRelay) addDevice(topic, addMsg, removeMsg string) {
	r.client.Publish(topic, 1, false, addMsg)
}

func (r *MqttRelay) removeDevice(topic, addMsg, removeMsg string) {
	token := r.client.Publish(topic, 1, false, removeMsg)
	token.Wait()
}

func (r *MqttRelay) hassStatusCallback(c mqtt.Client, msg mqtt.Message) {
	switch string(msg.Payload()) {
	case "online":
		log.Println("HA going online, sending mqtt discovery messages")
		r.HADiscoveryAdd()
	}
}

// HADiscoveryAdd will send a discovery message to Home Assistant with the provided discoveryPrefix
// that will add the devices to Home Assistant.
func (r *MqttRelay) SetupHADiscovery(discoveryPrefix string, autoremove bool) error {
	r.haDiscoveryPrefix = &discoveryPrefix
	r.haDiscoveryAutoremove = autoremove

	return r.HADiscoveryAdd()
}

// HADiscoveryAdd will send a discovery message to Home Assistant with the provided discoveryPrefix
// that will add the devices to Home Assistant.
func (r *MqttRelay) HADiscoveryAdd() error {
	var devices, datapoints int

	if r.haDiscoveryPrefix == nil {
		return nil
	}

	if err := r.ForEachDevice(func(device *xc.Device) error {
		if err := createDeviceDiscoveryMessages(*r.haDiscoveryPrefix, r.clientId, device, r.addDevice); err != nil {
			return err
		}
		devices++
		return nil
	}); err != nil {
		return err
	}

	if err := r.ForEachDatapoint(func(dp *xc.Datapoint) error {
		if err := createDpDiscoveryMessages(*r.haDiscoveryPrefix, r.clientId, dp, r.addDevice); err != nil {
			return err
		}
		datapoints++
		return nil
	}); err != nil {
		return err
	}

	log.Printf("Sent MQTT autodiscover add for %d devices and %d datapoints", devices, datapoints)

	return nil
}

// HADiscoveryRemove will send a discovery message to Home Assistant with the provided discoveryPrefix
// that will remove the devices from Home Assistant.  This will also wipe any alterations that the user
// may have made in HA, so this is by default turned off.
func (r *MqttRelay) HADiscoveryRemove() error {
	var devices, datapoints int

	if r.haDiscoveryPrefix == nil || !r.haDiscoveryAutoremove {
		return nil
	}

	if err := r.ForEachDevice(func(device *xc.Device) error {
		if err := createDeviceDiscoveryMessages(*r.haDiscoveryPrefix, r.clientId, device, r.removeDevice); err != nil {
			return err
		}
		devices++
		return nil
	}); err != nil {
		return err
	}

	if err := r.ForEachDatapoint(func(dp *xc.Datapoint) error {
		if err := createDpDiscoveryMessages(*r.haDiscoveryPrefix, r.clientId, dp, r.removeDevice); err != nil {
			return err
		}
		datapoints++
		return nil
	}); err != nil {
		return err
	}

	log.Printf("Sent MQTT autodiscover remove for %d devices and %d datapoints", devices, datapoints)

	return nil
}

func createDpDiscoveryMessages(discoveryPrefix, clientId string, dp *xc.Datapoint, fn func(topic, addMsg, removeMsg string)) error {
	var isDimmable bool

	deviceID := fmt.Sprintf("xcomfort_%d_%s", dp.Device().SerialNumber(), stripNonAlphanumeric.ReplaceAllString(dp.Name(), "_"))
	dataPoint := dp.Number()

	if dataPoint == 0 {
		// Ignore status report datapoint
		return nil
	}

	config := map[string]interface{}{
		"name":      dp.Name(),
		"unique_id": fmt.Sprintf("%d_ch%d", dp.Device().SerialNumber(), dp.Channel()),
		"device": map[string]string{
			"identifiers":  fmt.Sprintf("%d", dp.Device().SerialNumber()),
			"name":         fmt.Sprintf("%d", dp.Device().SerialNumber()),
			"manufacturer": "Eaton",
			"model":        dp.Device().Type().String(),
			"via_device":   "CI Stick",
		},
	}

	switch dp.Type() {
	case xc.STATUS_PERCENT:
		isDimmable = true
		fallthrough
	case xc.STATUS_BOOL:
		config["command_topic"] = fmt.Sprintf("%s/%d/set/switch", clientId, dataPoint)
		config["state_topic"] = fmt.Sprintf("%s/%d/get/switch", clientId, dataPoint)
		config["payload_on"] = "true"
		config["payload_off"] = "false"
		config["optimistic"] = "false"

		if isDimmable {
			config["brightness_command_topic"] = fmt.Sprintf("%s/%d/set/dimmer", clientId, dataPoint)
			config["brightness_state_topic"] = fmt.Sprintf("%s/%d/get/dimmer", clientId, dataPoint)
			config["brightness_scale"] = "100"
			config["on_command_type"] = "brightness"
		}

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		if dp.Type() != xc.STATUS_BOOL ||
			strings.HasPrefix(dp.Name(), "LI_") {
			fn(fmt.Sprintf("%s/light/%s/config", discoveryPrefix, deviceID), string(addMsg), "")
		} else {
			fn(fmt.Sprintf("%s/switch/%s/config", discoveryPrefix, deviceID), string(addMsg), "")
		}

	case xc.STATUS_SHUTTER:
		config["command_topic"] = fmt.Sprintf("%s/%d/set/shutter", clientId, dataPoint)
		config["payload_open"] = "open"
		config["payload_close"] = "close"
		config["payload_stop"] = "stop"
		config["state_topic"] = fmt.Sprintf("%s/%d/get/shutter", clientId, dataPoint)
		config["state_opening"] = "opening"
		config["state_closing"] = "closing"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/cover/%s/config", discoveryPrefix, deviceID), string(addMsg), "")

	case xc.PUSHBUTTON:
		delete(config, "name")
		delete(config, "unique_id")

		for i, a := range []map[xc.Event]string{
			{
				xc.EventOn:         "button_short_release",
				xc.EventUpPressed:  "button_long_press",
				xc.EventUpReleased: "button_long_release",
			},
			{
				xc.EventOff:          "button_short_release",
				xc.EventDownPressed:  "button_long_press",
				xc.EventDownReleased: "button_long_release",
			},
		} {
			config["subtype"] = fmt.Sprintf("button_%d", (dp.Channel()*2)+i+1)

			for ev, t := range a {
				config["topic"] = fmt.Sprintf("%s/%d/event", clientId, dataPoint)
				config["type"] = t
				config["payload"] = ev.String()
				config["automation_type"] = "trigger"

				addMsg, err := json.Marshal(config)
				if err != nil {
					return errors.WithStack(err)
				}

				fn(fmt.Sprintf("%s/device_automation/%s_%s/config", discoveryPrefix, deviceID, ev), string(addMsg), "")
			}
		}

	case xc.TEMPERATURE_SWITCH:
		if dp.Mode() == 0 {
			log.Printf("Datapoint %d using partially supported mode; ignoring switching commands", dataPoint)
		}

		config["unit_of_measurement"] = "C"
		config["state_topic"] = fmt.Sprintf("%s/%d/event/+", clientId, dataPoint)
		config["device_class"] = "temperature"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config", discoveryPrefix, deviceID), string(addMsg), "")

	case xc.HUMIDITY_SWITCH:
		if dp.Mode() == 0 {
			log.Printf("Datapoint %d using partially supported mode; ignoring switching commands", dataPoint)
		}

		config["unit_of_measurement"] = "%"
		config["state_topic"] = fmt.Sprintf("%s/%d/event/+", clientId, dataPoint)
		config["device_class"] = "humidity"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config", discoveryPrefix, deviceID), string(addMsg), "")

	case xc.SWITCH, xc.MOTION:
		if dp.Mode() != 1 {
			log.Printf("Datapoint %d using currently unsupported mode; ignoring", dataPoint)
		} else {
			config["state_topic"] = fmt.Sprintf("%s/%d/event", clientId, dataPoint)
			config["payload_on"] = xc.EventSwitchOn
			config["payload_off"] = xc.EventSwitchOff

			if dp.Type() == xc.MOTION {
				config["device_class"] = "motion"
			}

			addMsg, err := json.Marshal(config)
			if err != nil {
				return errors.WithStack(err)
			}

			fn(fmt.Sprintf("%s/binary_sensor/%s/config", discoveryPrefix, deviceID), string(addMsg), "")
		}

	case xc.POWER:
		config["unit_of_measurement"] = "W"
		config["state_topic"] = fmt.Sprintf("%s/%d/event/value", clientId, dataPoint)
		config["device_class"] = "power"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config", discoveryPrefix, deviceID), string(addMsg), "")

	case xc.ENERGY:
		config["unit_of_measurement"] = "kWh"
		config["state_topic"] = fmt.Sprintf("%s/%d/event/value", clientId, dataPoint)
		config["device_class"] = "energy"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config", discoveryPrefix, deviceID), string(addMsg), "")
	}

	return nil
}

func createDeviceDiscoveryMessages(discoveryPrefix, clientId string, device *xc.Device, fn func(topic, addMsg, removeMsg string)) error {
	deviceID := fmt.Sprintf("xcomfort_%d", device.SerialNumber())

	config := map[string]interface{}{
		"device": map[string]string{
			"identifiers":  fmt.Sprintf("%d", device.SerialNumber()),
			"name":         fmt.Sprintf("%d", device.SerialNumber()),
			"manufacturer": "Eaton",
			"model":        device.Type().String(),
			"via_device":   "CI Stick",
		},
	}

	if device.Type() == xc.DT_CSAU_0101 ||
		device.Type() == xc.DT_CDAx_01NG {
		config["state_topic"] = fmt.Sprintf("%s/%d/internal_temperature", clientId, device.SerialNumber())
		config["device_class"] = "temperature"
		config["unit_of_measurement"] = "C"
		config["name"] = "Temperature"
		config["unique_id"] = fmt.Sprintf("%d_temperature", device.SerialNumber())

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s_internal_temperature/config", discoveryPrefix, deviceID), string(addMsg), "")
	}

	if device.IsBatteryOperated() {
		config["state_topic"] = fmt.Sprintf("%s/%d/battery", clientId, device.SerialNumber())
		config["device_class"] = "battery"
		config["unit_of_measurement"] = "%"
		config["name"] = "Battery"
		config["unique_id"] = fmt.Sprintf("%d_battery", device.SerialNumber())

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s_battery/config", discoveryPrefix, deviceID), string(addMsg), "")
	}

	config["state_topic"] = fmt.Sprintf("%s/%d/rssi", clientId, device.SerialNumber())
	config["device_class"] = "signal_strength"
	config["unit_of_measurement"] = "-dBm"
	config["name"] = "Signal strength"
	config["unique_id"] = fmt.Sprintf("%d_rssi", device.SerialNumber())

	addMsg, err := json.Marshal(config)
	if err != nil {
		return errors.WithStack(err)
	}

	fn(fmt.Sprintf("%s/sensor/%s_rssi/config", discoveryPrefix, deviceID), string(addMsg), "")

	return nil
}
