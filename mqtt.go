package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"time"

	"github.com/karloygard/xcomfortd-go/pkg/xc"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

var stripNonAlphanumeric = regexp.MustCompile("[^a-zA-Z0-9]+")

type MqttRelay struct {
	xc.Interface

	client mqtt.Client
	ctx    context.Context

	haDiscoveryPrefix     *string
	haDiscoveryAutoremove bool
	clientId              string
}

func (r *MqttRelay) dimmerCallback(c mqtt.Client, msg mqtt.Message) {
	var dp, value int

	if _, err := fmt.Sscanf(msg.Topic(), fmt.Sprintf("%s/%%d/set/dimmer", r.clientId), &dp); err != nil {
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

	if _, err := fmt.Sscanf(msg.Topic(), fmt.Sprintf("%s/%%d/set/switch", r.clientId), &dp); err != nil {
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

	if _, err := fmt.Sscanf(msg.Topic(), fmt.Sprintf("%s/%%d/set/shutter", r.clientId), &dp); err != nil {
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
	topic := fmt.Sprintf("%s/%d/get/dimmer", r.clientId, datapoint.Number())
	r.client.Publish(topic, 1, true, fmt.Sprint(value))
	r.StatusBool(datapoint, value > 0)
}

func (r *MqttRelay) StatusBool(datapoint *xc.Datapoint, on bool) {
	topic := fmt.Sprintf("%s/%d/get/switch", r.clientId, datapoint.Number())
	r.client.Publish(topic, 1, true, fmt.Sprint(on))
}

func (r *MqttRelay) StatusShutter(datapoint *xc.Datapoint, status xc.ShutterStatus) {
	topic := fmt.Sprintf("%s/%d/get/shutter", r.clientId, datapoint.Number())
	r.client.Publish(topic, 1, false, string(status))
}

func (r *MqttRelay) Event(datapoint *xc.Datapoint, event xc.Event) {
	topic := fmt.Sprintf("%s/%d/event", r.clientId, datapoint.Number())
	r.client.Publish(topic, 1, false, string(event))
}

func (r *MqttRelay) ValueEvent(datapoint *xc.Datapoint, event xc.Event, value interface{}) {
	topic := fmt.Sprintf("%s/%d/event/%s", r.clientId, datapoint.Number(), event)
	r.client.Publish(topic, 1, event == xc.EventValue, fmt.Sprint(value))
}

func (r *MqttRelay) Battery(device *xc.Device, percentage int) {
	topic := fmt.Sprintf("%s/%d/battery", r.clientId, device.SerialNumber())
	r.client.Publish(topic, 1, true, fmt.Sprint(percentage))
}

func (r *MqttRelay) Rssi(device *xc.Device, dbm int) {
	topic := fmt.Sprintf("%s/%d/rssi", r.clientId, device.SerialNumber())
	r.client.Publish(topic, 1, true, fmt.Sprint(dbm))
}

func (r *MqttRelay) InternalTemperature(device *xc.Device, temperature int) {
	topic := fmt.Sprintf("%s/%d/internal_temperature", r.clientId, device.SerialNumber())
	r.client.Publish(topic, 1, true, fmt.Sprint(temperature))
}

func (r *MqttRelay) DPLChanged() {
	log.Printf("DPL Changed")

	err := r.HADiscoveryRemove()
	if err == nil {
		err = r.HADiscoveryAdd()
	}

	if err != nil {
		log.Println(err)
	}
}

func (r *MqttRelay) connected(c mqtt.Client) {
	log.Println("Connected to broker")
}

func (r *MqttRelay) connectionLost(c mqtt.Client, err error) {
	log.Printf("Lost connection with broker: %s", err)
}

func (r *MqttRelay) Connect(ctx context.Context, clientId string, uri *url.URL, id int) error {
	opts := mqtt.NewClientOptions()
	broker := fmt.Sprintf("tcp://%s", uri.Host)

	if id > 0 {
		r.clientId = fmt.Sprintf("%s-%d", clientId, id)
	} else {
		r.clientId = fmt.Sprintf("%s", clientId)
	}
	log.Printf("Connecting to MQTT broker '%s' with id '%s'", broker, r.clientId)

	opts.AddBroker(broker).
		SetClientID(r.clientId).
		SetOnConnectHandler(r.connected).
		SetConnectionLostHandler(r.connectionLost).
		SetOrderMatters(false).
		SetKeepAlive(30 * time.Second).
		SetUsername(uri.User.Username())
	if password, set := uri.User.Password(); set {
		opts.SetPassword(password)
	}

	r.client = mqtt.NewClient(opts)
	token := r.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return errors.WithStack(err)
	}

	r.client.Subscribe(fmt.Sprintf("%s/+/set/dimmer", r.clientId), 0, r.dimmerCallback)
	r.client.Subscribe(fmt.Sprintf("%s/+/set/switch", r.clientId), 0, r.switchCallback)
	r.client.Subscribe(fmt.Sprintf("%s/+/set/shutter", r.clientId), 0, r.shutterCallback)

	r.ctx = ctx

	return nil
}

func (r *MqttRelay) Close() {
	r.client.Disconnect(1000)
}
