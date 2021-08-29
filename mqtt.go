package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/karloygard/xcomfortd-go/pkg/xc"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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

func (r *MqttRelay) desiredTemperatureCallback(c mqtt.Client, msg mqtt.Message) {
	var dp int
	var value float32

	if _, err := fmt.Sscanf(msg.Topic(), fmt.Sprintf("%s/%%d/set/temperature", r.clientId), &dp); err != nil {
		log.Println(err)
		return
	}
	if _, err := fmt.Sscanf(string(msg.Payload()), "%f", &value); err != nil {
		log.Println(err)
		return
	}

	if datapoint := r.Datapoint(dp); datapoint != nil {
		log.Printf("MQTT message; topic: '%s', message: '%s'\n", msg.Topic(), string(msg.Payload()))

		if _, err := datapoint.DesiredTemperature(r.ctx, value); err != nil {
			log.Printf("WARNING: command for datapoint %d failed, state now unknown: %v", dp, err)
		} /*else {
			// Required?
			r.Temperature(datapoint, value)
		}*/
	} else {
		log.Printf("unknown datapoint %d\n", dp)
	}
}

func (r *MqttRelay) currentTemperatureCallback(c mqtt.Client, msg mqtt.Message) {
	var dp int
	var value float32

	if _, err := fmt.Sscanf(msg.Topic(), fmt.Sprintf("%s/%%d/set/current", r.clientId), &dp); err != nil {
		log.Println(err)
		return
	}
	if _, err := fmt.Sscanf(string(msg.Payload()), "%f", &value); err != nil {
		log.Println(err)
		return
	}

	if datapoint := r.Datapoint(dp); datapoint != nil {
		log.Printf("MQTT message; topic: '%s', message: '%s'\n", msg.Topic(), string(msg.Payload()))

		if _, err := datapoint.CurrentTemperature(r.ctx, value); err != nil {
			log.Printf("WARNING: command for datapoint %d failed, state now unknown: %v", dp, err)
		} else {
			topic := fmt.Sprintf("%s/%d/get/current_temperature", r.clientId, datapoint.Number())
			r.publish(topic, fmt.Sprint(value))
		}
	} else {
		log.Printf("unknown datapoint %d\n", dp)
	}
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
	r.publish(topic, fmt.Sprint(value))
	r.StatusBool(datapoint, value > 0)
}

func (r *MqttRelay) Value(datapoint *xc.Datapoint, value interface{}) {
	topic := fmt.Sprintf("%s/%d/get/value", r.clientId, datapoint.Number())
	r.publish(topic, fmt.Sprint(value))
}

func (r *MqttRelay) StatusBool(datapoint *xc.Datapoint, on bool) {
	topic := fmt.Sprintf("%s/%d/get/switch", r.clientId, datapoint.Number())
	r.publish(topic, fmt.Sprint(on))
}

func (r *MqttRelay) StatusShutter(datapoint *xc.Datapoint, status xc.ShutterStatus) {
	topic := fmt.Sprintf("%s/%d/get/shutter", r.clientId, datapoint.Number())
	r.publish(topic, string(status))
}

func (r *MqttRelay) Event(datapoint *xc.Datapoint, event xc.Event) {
	topic := fmt.Sprintf("%s/%d/event", r.clientId, datapoint.Number())
	r.publish(topic, string(event))
}

func (r *MqttRelay) ValueEvent(datapoint *xc.Datapoint, event xc.Event, value interface{}) {
	topic := fmt.Sprintf("%s/%d/event/%s", r.clientId, datapoint.Number(), event)
	r.publish(topic, fmt.Sprint(value))
}

func (r *MqttRelay) Wheel(datapoint *xc.Datapoint, value interface{}) {
	topic := fmt.Sprintf("%s/%d/wheel", r.clientId, datapoint.Number())
	r.publish(topic, fmt.Sprint(value))
}

func (r *MqttRelay) Battery(device *xc.Device, percentage int) {
	topic := fmt.Sprintf("%s/%d/battery", r.clientId, device.SerialNumber())
	r.publish(topic, fmt.Sprint(percentage))
}

func (r *MqttRelay) Rssi(device *xc.Device, dbm int) {
	topic := fmt.Sprintf("%s/%d/rssi", r.clientId, device.SerialNumber())
	r.publish(topic, fmt.Sprint(dbm))
}

func (r *MqttRelay) InternalTemperature(device *xc.Device, temperature int) {
	topic := fmt.Sprintf("%s/%d/internal_temperature", r.clientId, device.SerialNumber())
	r.publish(topic, fmt.Sprint(temperature))
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

func (r *MqttRelay) Connect(ctx context.Context, clientId string, uri *url.URL, id int) error {
	opts := mqtt.NewClientOptions()
	broker := fmt.Sprintf("tcp://%s", uri.Host)

	if id > 0 {
		r.clientId = fmt.Sprintf("%s-%d", clientId, id)
	} else {
		r.clientId = fmt.Sprintf("%s", clientId)
	}
	log.Printf("Connecting to MQTT broker '%s' with id '%s'", broker, r.clientId)

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)

	opts.AddBroker(broker).
		SetClientID(r.clientId).
		SetConnectRetry(true).
		SetOnConnectHandler(r.connected).
		SetConnectionLostHandler(r.connectionLost).
		SetKeepAlive(30 * time.Second).
		SetUsername(uri.User.Username())
	if password, set := uri.User.Password(); set {
		opts.SetPassword(password)
	}

	r.client = mqtt.NewClient(opts)
	t := r.client.Connect()
	go func() {
		<-t.Done()
		if t.Error() != nil {
			log.Println(t.Error())
		}
	}()

	r.ctx = ctx

	return nil
}

func (r *MqttRelay) Close() {
	r.client.Disconnect(1000)
}

func (r *MqttRelay) connected(c mqtt.Client) {
	subscriptions := map[string]func(c mqtt.Client, m mqtt.Message){
		"dimmer":              r.dimmerCallback,
		"switch":              r.switchCallback,
		"shutter":             r.shutterCallback,
		"temperature":         r.desiredTemperatureCallback,
		"current_temperature": r.currentTemperatureCallback,
	}

	for k, c := range subscriptions {
		cb := c
		r.subscribe(fmt.Sprintf("%s/+/set/%s", r.clientId, k),
			func(c mqtt.Client, m mqtt.Message) { go cb(c, m) })
	}

	if r.haDiscoveryPrefix != nil {
		r.client.Subscribe(*r.haDiscoveryPrefix+"/status", 0,
			func(c mqtt.Client, m mqtt.Message) { go r.hassStatusCallback(c, m) })
	}

	log.Println("Connected to broker")
}

func (r *MqttRelay) connectionLost(c mqtt.Client, err error) {
	log.Printf("Lost connection with broker: %s", err)
}

func (r *MqttRelay) publish(topic string, msg string) {
	t := r.client.Publish(topic, 1, true, msg)
	go func() {
		<-t.Done()
		if t.Error() != nil {
			log.Println(t.Error())
		}
	}()
}

func (r *MqttRelay) subscribe(topic string, callback mqtt.MessageHandler) {
	t := r.client.Subscribe(topic, 1, callback)
	go func() {
		<-t.Done()
		if t.Error() != nil {
			log.Println(t.Error())
		}
	}()
}
