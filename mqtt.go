package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"

	"github.com/karloygard/xcomfortd-go/pkg/xc"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

var stripNonAlphanumeric = regexp.MustCompile("[^a-zA-Z0-9]+")

type MqttRelay struct {
	xc.Interface

	client mqtt.Client
	ctx    context.Context
}

func (r *MqttRelay) dimmerCallback(c mqtt.Client, msg mqtt.Message) {
	var dp, value int

	if _, err := fmt.Sscanf(msg.Topic(), "xcomfort/%d/set/dimmer", &dp); err != nil {
		log.Println(err)
		return
	}
	if _, err := fmt.Sscanf(string(msg.Payload()), "%d", &value); err != nil {
		log.Println(err)
		return
	}

	if datapoint := r.Datapoint(dp); datapoint != nil {
		log.Printf("MQTT message; topic: '%s', message: '%s'\n", msg.Topic(), string(msg.Payload()))

		if _, err := datapoint.Dim(r.ctx, value); err != nil {
			log.Printf("WARNING: command for datapoint %d failed, state now unknown: %v", dp, err)
		} else {
			r.StatusValue(datapoint, value)
			// Send bool as well, to appease HA
			r.StatusBool(datapoint, value > 0)
		}
	} else {
		log.Printf("unknown datapoint %d\n", dp)
	}
}

func (r *MqttRelay) switchCallback(c mqtt.Client, msg mqtt.Message) {
	var dp int

	if _, err := fmt.Sscanf(msg.Topic(), "xcomfort/%d/set/switch", &dp); err != nil {
		log.Println(err)
		return
	}

	if datapoint := r.Datapoint(dp); datapoint != nil {
		log.Printf("MQTT message; topic: '%s', message: '%s'\n", msg.Topic(), string(msg.Payload()))

		on := string(msg.Payload()) == "true"

		if _, err := datapoint.Switch(r.ctx, on); err != nil {
			log.Printf("WARNING: command for datapoint %d failed, state now unknown: %v", dp, err)
		} else {
			r.StatusBool(datapoint, on)
		}
	} else {
		log.Printf("unknown datapoint %d\n", dp)
	}
}

func (r *MqttRelay) shutterCallback(c mqtt.Client, msg mqtt.Message) {
	var dp int

	if _, err := fmt.Sscanf(msg.Topic(), "xcomfort/%d/set/shutter", &dp); err != nil {
		log.Println(err)
		return
	}

	if datapoint := r.Datapoint(dp); datapoint != nil {
		log.Printf("MQTT message; topic: '%s', message: '%s'\n", msg.Topic(), string(msg.Payload()))

		cmd := xc.ShutterClose

		switch string(msg.Payload()) {
		case "close":
		case "open":
			cmd = xc.ShutterOpen
		case "stop":
			cmd = xc.ShutterStop
		case "stepopen":
			cmd = xc.ShutterStepOpen
		case "stepclose":
			cmd = xc.ShutterStepClose
		default:
			log.Printf("unknown shutter command %s\n", string(msg.Payload()))
			return
		}

		if _, err := datapoint.Shutter(r.ctx, cmd); err != nil {
			log.Printf("WARNING: command for datapoint %d failed, state now unknown: %v", dp, err)
		}
	} else {
		log.Printf("unknown datapoint %d\n", dp)
	}
}

func (r *MqttRelay) StatusValue(datapoint *xc.Datapoint, value int) {
	topic := fmt.Sprintf("xcomfort/%d/get/dimmer", datapoint.Number())
	r.client.Publish(topic, 1, true, fmt.Sprint(value))
	r.StatusBool(datapoint, value > 0)
}

func (r *MqttRelay) StatusBool(datapoint *xc.Datapoint, on bool) {
	topic := fmt.Sprintf("xcomfort/%d/get/switch", datapoint.Number())
	r.client.Publish(topic, 1, true, fmt.Sprint(on))
}

func (r *MqttRelay) StatusShutter(datapoint *xc.Datapoint, status xc.ShutterStatus) {
	topic := fmt.Sprintf("xcomfort/%d/get/shutter", datapoint.Number())
	r.client.Publish(topic, 1, false, string(status))
}

func (r *MqttRelay) Event(datapoint *xc.Datapoint, event xc.Event) {
	topic := fmt.Sprintf("xcomfort/%d/event", datapoint.Number())
	r.client.Publish(topic, 1, false, string(event))
}

func (r *MqttRelay) ValueEvent(datapoint *xc.Datapoint, event xc.Event, value interface{}) {
	topic := fmt.Sprintf("xcomfort/%d/event/%s", datapoint.Number(), event)
	r.client.Publish(topic, 1, event == xc.EventValue, fmt.Sprint(value))
}

func (r *MqttRelay) Battery(device *xc.Device, percentage int) {
	topic := fmt.Sprintf("xcomfort/%d/battery", device.SerialNumber())
	r.client.Publish(topic, 1, true, fmt.Sprint(percentage))
}

func (r *MqttRelay) Rssi(device *xc.Device, dbm int) {
	topic := fmt.Sprintf("xcomfort/%d/rssi", device.SerialNumber())
	r.client.Publish(topic, 1, true, fmt.Sprint(dbm))
}

func (r *MqttRelay) InternalTemperature(device *xc.Device, temperature int) {
	topic := fmt.Sprintf("xcomfort/%d/internal_temperature", device.SerialNumber())
	r.client.Publish(topic, 1, true, fmt.Sprint(temperature))
}

func (r *MqttRelay) DPLChanged() {
	log.Printf("DPL Changed")
}

func (r *MqttRelay) connected(c mqtt.Client) {
	log.Println("Connected to broker")
}

func (r *MqttRelay) connectionLost(c mqtt.Client, err error) {
	log.Printf("Lost connection with broker: %s", err)
}

func (r *MqttRelay) Connect(ctx context.Context, clientId string, uri *url.URL) error {
	opts := mqtt.NewClientOptions()
	broker := fmt.Sprintf("tcp://%s", uri.Host)

	log.Printf("Connecting to MQTT broker '%s' with id '%s'", broker, clientId)

	opts.AddBroker(broker)
	opts.SetUsername(uri.User.Username())
	if password, set := uri.User.Password(); set {
		opts.SetPassword(password)
	}
	opts.SetClientID(clientId)
	opts.SetOnConnectHandler(r.connected)
	opts.SetConnectionLostHandler(r.connectionLost)

	r.client = mqtt.NewClient(opts)
	token := r.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return errors.WithStack(err)
	}

	r.client.Subscribe("xcomfort/+/set/dimmer", 0, r.dimmerCallback)
	r.client.Subscribe("xcomfort/+/set/switch", 0, r.switchCallback)
	r.client.Subscribe("xcomfort/+/set/shutter", 0, r.shutterCallback)

	r.ctx = ctx

	return nil
}

func (r *MqttRelay) Close() {
	r.client.Disconnect(1000)
}
