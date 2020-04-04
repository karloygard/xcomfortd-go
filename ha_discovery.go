package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/karloygard/xcomfortd-go/pkg/xc"
)

// HADiscoveryAdd will send a discovery message to Home Assistant with the provided discoveryPrefix
// that will add the devices to Home Assistant.
func (r *MqttRelay) HADiscoveryAdd(discoveryPrefix string) error {
	return r.ForEachDatapoint(func(dp *xc.Datapoint) error {
		topic, addMsg, _, err := createDiscoveryMessages(discoveryPrefix, dp)
		if err != nil {
			return err
		}
		if topic != "" {
			log.Printf("Sending HA discovery add message: %s\n", topic)
			r.client.Publish(topic, 1, true, addMsg)
		}
		return nil
	})
}

// HADiscoveryRemove will send a discovery message to Home Assistant with the provided discoveryPrefix
// that will remove the devices from Home Assistant.
func (r *MqttRelay) HADiscoveryRemove(discoveryPrefix string) error {
	return r.ForEachDatapoint(func(dp *xc.Datapoint) error {
		topic, _, removeMsg, err := createDiscoveryMessages(discoveryPrefix, dp)
		if err != nil {
			return err
		}
		if topic != "" {
			log.Printf("Sending HA discovery remove message: %s\n", topic)
			r.client.Publish(topic, 1, true, removeMsg)
		}
		return nil
	})
}

func createDiscoveryMessages(discoveryPrefix string, dp *xc.Datapoint) (string, string, string, error) {
	var isActuator bool
	var isDimmable bool

	switch {
	case dp.Device().IsSwitchingActuator():
		isActuator = true
	case dp.Device().IsDimmingActuator():
		isActuator = true
		isDimmable = true
	}

	if !isActuator || dp.Channel() != 0 || dp.Number() == 0 {
		return "", "", "", nil
	}

	deviceID := fmt.Sprintf("xcomfort_%d_%s", dp.Device().SerialNumber(), stripNonAlphanumeric.ReplaceAllString(dp.Name(), "_"))
	dataPoint := dp.Number()

	config := map[string]string{
		"name":          deviceID,
		"command_topic": fmt.Sprintf("xcomfort/%d/set/switch", dataPoint),
		"state_topic":   fmt.Sprintf("xcomfort/%d/get/switch", dataPoint),
		"payload_on":    "true",
		"payload_off":   "false",
		"optimistic":    "false",
	}

	if isDimmable {
		config["brightness_command_topic"] = fmt.Sprintf("xcomfort/%d/set/dimmer", dataPoint)
		config["brightness_state_topic"] = fmt.Sprintf("xcomfort/%d/get/dimmer", dataPoint)
		config["brightness_scale"] = "100"
		config["on_command_type"] = "brightness"
	}

	addMsg, err := json.Marshal(config)
	if err != nil {
		return "", "", "", err
	}

	return fmt.Sprintf("%s/light/%s/config", discoveryPrefix, deviceID), string(addMsg), "", nil
}
