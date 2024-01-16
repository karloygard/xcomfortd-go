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

func (r *MqttRelay) hassStatusCallback(msg mqtt.Message) {
	switch string(msg.Payload()) {
	case "online":
		log.Println("HA going online, sending mqtt discovery messages")
		r.HADiscoveryAdd()
	}
}

func (r *MqttRelay) SetupHADiscovery(discoveryPrefix string, autoremove bool) {
	r.haDiscoveryPrefix = &discoveryPrefix
	r.haDiscoveryAutoremove = autoremove
}

// HADiscoveryAdd will send a discovery message to Home Assistant with the
// provided discoveryPrefix that will add the devices to Home Assistant.
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

	log.Printf("Sent MQTT autodiscover add for %d devices and %d datapoints",
		devices, datapoints)

	return nil
}

// HADiscoveryRemove will send a discovery message to Home Assistant with the
// provided discoveryPrefix that will remove the devices from Home Assistant.
// This will also wipe any alterations that the user may have made in HA, so
// this is by default turned off.
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

	log.Printf("Sent MQTT autodiscover remove for %d devices and %d datapoints",
		devices, datapoints)

	return nil
}

func createDpDiscoveryMessages(discoveryPrefix, clientId string,
	dp *xc.Datapoint, fn func(topic, addMsg, removeMsg string)) error {

	var isDimmable bool

	entityID := dp.Id()
	dataPoint := dp.Number()

	if dataPoint == 0 {
		// Ignore status report datapoint
		return nil
	}

	config := map[string]interface{}{
		"unique_id": entityID,
		"device": map[string]string{
			"identifiers":  fmt.Sprintf("%d", dp.Device().SerialNumber()),
			"name":         dp.Device().Name(),
			"manufacturer": "Eaton",
			"model":        dp.Device().Type().String(),
			"via_device":   "CI Stick",
		},
	}

	if dp.Name() != "" {
		config["name"] = dp.Name()
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

			if dp.Name() == "" {
				// HA gives us "MQTT LightEntity" if we don't set it
				config["name"] = "Light"
			}
		}

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		if dp.Type() != xc.STATUS_BOOL ||
			strings.HasPrefix(dp.Device().Name(), "LI_") {
			fn(fmt.Sprintf("%s/light/%s/config",
				discoveryPrefix, entityID), string(addMsg), "")
		} else {
			fn(fmt.Sprintf("%s/switch/%s/config",
				discoveryPrefix, entityID), string(addMsg), "")
		}

	case xc.STATUS_SHUTTER:
		config["command_topic"] = fmt.Sprintf("%s/%d/set/shutter", clientId, dataPoint)
		config["position_topic"] = fmt.Sprintf("%s/%d/get/position", clientId, dataPoint)
		config["payload_open"] = "open"
		config["payload_close"] = "close"
		config["payload_stop"] = "stop"
		config["state_topic"] = fmt.Sprintf("%s/%d/get/shutter", clientId, dataPoint)
		config["state_opening"] = xc.ShutterStateOpening
		config["state_closing"] = xc.ShutterStateClosing
		config["state_stopped"] = xc.ShutterStateStopped
		config["state_open"] = xc.ShutterStateOpen
		config["state_closed"] = xc.ShutterStateClosed

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/cover/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

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

				fn(fmt.Sprintf("%s/device_automation/%s_%s/config",
					discoveryPrefix, entityID, ev), string(addMsg), "")
			}
		}

	case xc.TEMPERATURE_SWITCH,
		xc.TEMPERATURE_WHEEL_SWITCH:
		if dp.Mode() == 0 {
			log.Printf("Datapoint %d using partially supported mode; ignoring switching commands", dataPoint)
		}

		config["unit_of_measurement"] = "°C"
		config["state_topic"] = fmt.Sprintf("%s/%d/event/+", clientId, dataPoint)
		config["device_class"] = "temperature"
		config["state_class"] = "measurement"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

		if dp.Type() == xc.TEMPERATURE_WHEEL_SWITCH {
			config["state_topic"] = fmt.Sprintf("%s/%d/wheel", clientId, dataPoint)
			config["name"] = "Temperature adjustment"
			config["unique_id"] = fmt.Sprintf("%s_wheel", entityID)

			addMsg, err := json.Marshal(config)
			if err != nil {
				return errors.WithStack(err)
			}

			fn(fmt.Sprintf("%s/sensor/%s_wheel/config",
				discoveryPrefix, entityID), string(addMsg), "")
		}

	case xc.VALUE_SWITCH:
		if dp.Mode() == 0 {
			log.Printf("Datapoint %d using partially supported mode; ignoring switching commands", dataPoint)
		}

		config["state_topic"] = fmt.Sprintf("%s/%d/event/+", clientId, dataPoint)

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

	case xc.HUMIDITY_SWITCH:
		if dp.Mode() == 0 {
			log.Printf("Datapoint %d using partially supported mode; ignoring switching commands", dataPoint)
		}

		config["unit_of_measurement"] = "%"
		config["state_topic"] = fmt.Sprintf("%s/%d/event/+", clientId, dataPoint)
		config["device_class"] = "humidity"
		config["state_class"] = "measurement"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

	case xc.SWITCH, xc.MOTION:
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

		fn(fmt.Sprintf("%s/binary_sensor/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

	case xc.POWER:
		config["unit_of_measurement"] = "W"
		config["state_topic"] = fmt.Sprintf("%s/%d/event/value", clientId, dataPoint)
		config["state_class"] = "measurement"
		config["device_class"] = "power"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

	case xc.DIMPLEX:
		config["temperature_command_topic"] = fmt.Sprintf("%s/%d/set/temperature", clientId, dataPoint)
		config["current_temperature_topic"] = fmt.Sprintf("%s/%d/get/current_temperature", clientId, dataPoint)
		config["mode_state_topic"] = fmt.Sprintf("%s/%d/get/value", clientId, dataPoint)
		config["precision"] = 0.1
		config["temp_step"] = 0.1
		config["modes"] = []string{"off", "heat"}

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/climate/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

		delete(config, "precision")
		delete(config, "temp_step")
		delete(config, "mode_state_topic")
		delete(config, "current_temperature_topic")
		delete(config, "modes")
		delete(config, "temperature_command_topic")

		config["command_topic"] = fmt.Sprintf("%s/%d/set/current_temperature", clientId, dataPoint)
		config["name"] = "Current temperature"
		config["step"] = 0.1
		config["unique_id"] = fmt.Sprintf("%s_current", entityID)

		addMsg, err = json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/number/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

	case xc.ENERGY:
		config["unit_of_measurement"] = "kWh"
		config["state_topic"] = fmt.Sprintf("%s/%d/event/value", clientId, dataPoint)
		config["device_class"] = "energy"
		config["state_class"] = "total_increasing"

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")

	case xc.PULSES:
		config["state_topic"] = fmt.Sprintf("%s/%d/event/value", clientId, dataPoint)

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s/config",
			discoveryPrefix, entityID), string(addMsg), "")
	}

	return nil
}

func createDeviceDiscoveryMessages(discoveryPrefix, clientId string,
	device *xc.Device, fn func(topic, addMsg, removeMsg string)) error {

	deviceID := fmt.Sprintf("xcomfort_%d", device.SerialNumber())

	config := map[string]interface{}{
		"device": map[string]string{
			"identifiers":  fmt.Sprintf("%d", device.SerialNumber()),
			"name":         device.Name(),
			"manufacturer": "Eaton",
			"model":        device.Type().String(),
			"via_device":   "CI Stick",
		},
	}

	config["state_class"] = "measurement"

	if device.Type() == xc.DT_CSAU_0101 ||
		device.Type() == xc.DT_CDAx_01NG {
		config["state_topic"] = fmt.Sprintf("%s/%d/internal_temperature", clientId, device.SerialNumber())
		config["device_class"] = "temperature"
		config["unit_of_measurement"] = "°C"
		config["unique_id"] = fmt.Sprintf("%d_temperature", device.SerialNumber())

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s_internal_temperature/config",
			discoveryPrefix, deviceID), string(addMsg), "")
	}

	if device.IsBatteryOperated() {
		config["state_topic"] = fmt.Sprintf("%s/%d/battery", clientId, device.SerialNumber())
		config["device_class"] = "battery"
		config["unit_of_measurement"] = "%"
		config["unique_id"] = fmt.Sprintf("%d_battery", device.SerialNumber())

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s_battery/config",
			discoveryPrefix, deviceID), string(addMsg), "")
	}

	if device.ReportsPower() {
		config["state_topic"] = fmt.Sprintf("%s/%d/power", clientId, device.SerialNumber())
		config["device_class"] = "power"
		config["unit_of_measurement"] = "W"
		config["unique_id"] = fmt.Sprintf("%d_power", device.SerialNumber())

		addMsg, err := json.Marshal(config)
		if err != nil {
			return errors.WithStack(err)
		}

		fn(fmt.Sprintf("%s/sensor/%s_power/config",
			discoveryPrefix, deviceID), string(addMsg), "")
	}

	config["state_topic"] = fmt.Sprintf("%s/%d/rssi", clientId, device.SerialNumber())
	config["device_class"] = "signal_strength"
	config["unit_of_measurement"] = "dBm"
	config["unique_id"] = fmt.Sprintf("%d_rssi", device.SerialNumber())

	addMsg, err := json.Marshal(config)
	if err != nil {
		return errors.WithStack(err)
	}

	fn(fmt.Sprintf("%s/sensor/%s_rssi/config",
		discoveryPrefix, deviceID), string(addMsg), "")

	return nil
}
