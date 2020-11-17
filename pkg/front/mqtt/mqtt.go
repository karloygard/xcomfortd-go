package mqtt

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"

	evbus "github.com/asaskevich/EventBus"
	"github.com/karloygard/xcomfortd-go/pkg/bus"
	"github.com/karloygard/xcomfortd-go/pkg/xc"

	mqttclient "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)


var stripNonAlphanumeric = regexp.MustCompile("[^a-zA-Z0-9]+")

type MqttRelay struct {
	ctx    context.Context
	client mqttclient.Client

	haDiscoveryPrefix *string
}

func (r *MqttRelay) Close() {
	r.client.Disconnect(1000)
}

func CreateMqttRelay(ctx context.Context, clientId string, uri *url.URL, b evbus.Bus) (*MqttRelay, error) {
	r := &MqttRelay{}

	opts := mqttclient.NewClientOptions()
	broker := fmt.Sprintf("tcp://%s", uri.Host)

	log.Printf("Connecting to MQTT broker '%s' with id '%s'", broker, clientId)

	opts.AddBroker(broker)
	opts.SetUsername(uri.User.Username())
	if password, set := uri.User.Password(); set {
		opts.SetPassword(password)
	}
	opts.SetClientID(clientId)
	opts.SetOnConnectHandler(func (c mqttclient.Client) {
		log.Println("Connected to broker")
	})
	opts.SetConnectionLostHandler(func (c mqttclient.Client, err error) {
		log.Printf("Lost connection with broker: %s", err)
	})

	r.client = mqttclient.NewClient(opts)
	token := r.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return nil,errors.WithStack(err)
	}

	r.client.Subscribe("xcomfort/+/set/dimmer", 0, func(_ mqttclient.Client, msg mqttclient.Message) {
		handleCommand(msg, "xcomfort/%d/set/dimmer", func(dp int) error {
			var value int
			if _, err := fmt.Sscanf(string(msg.Payload()), "%d", &value); err != nil {
				return err
			}
			b.Publish(bus.TOPIC_COMMAND_DIMMER, dp, value)
			return nil
		});
	})

	r.client.Subscribe("xcomfort/+/set/switch", 0, func(_ mqttclient.Client, msg mqttclient.Message) {
		handleCommand(msg, "xcomfort/%d/set/switch", func(dp int) error {
			b.Publish(bus.TOPIC_COMMAND_SWITCH, dp, string(msg.Payload()) == "true")
			return nil
		});
	})

	r.client.Subscribe("xcomfort/+/set/shutter", 0, func(_ mqttclient.Client, msg mqttclient.Message) {
		handleCommand(msg, "xcomfort/%d/set/shutter", func(dp int) error {
			var cmd xc.ShutterCommand
			
			switch string(msg.Payload()) {
			case "close":
				cmd = xc.ShutterClose
			case "open":
				cmd = xc.ShutterOpen
			case "stop":
				cmd = xc.ShutterStop
			case "stepopen":
				cmd = xc.ShutterStepOpen
			case "stepclose":
				cmd = xc.ShutterStepClose
			default:
				return errors.Errorf("unknown shutter command %s\n", string(msg.Payload()))
			}
			
			b.Publish(bus.TOPIC_COMMAND_SWITCH, dp, cmd)
			return nil
		});
	})

	b.SubscribeAsync(bus.TOPIC_EVENT_DP_STATUS_VALUE, func (dp int, brigthness int) {
		r.sendEvent(dp, "xcomfort/%d/get/dimmer", 1, true, brigthness)
	}, false);

	b.SubscribeAsync(bus.TOPIC_EVENT_DP_STATUS_BOOL, func (dp int, state bool) {
		r.sendEvent(dp, "xcomfort/%d/get/switch", 1, true, state)
	}, false)

	b.SubscribeAsync(bus.TOPIC_EVENT_DP_STATUS_SHUTTER, func (dp int, command xc.ShutterStatus) {
		r.sendEvent(dp, "xcomfort/%d/get/shutter", 1, false, command)
	}, false)

	b.SubscribeAsync(bus.TOPIC_EVENT_DP_EVENT, func (dp int, event xc.Event, value interface{}) {
		if value == nil {
			r.sendEvent(dp, "xcomfort/%d/event", 1, false, event)
		} else {
			r.sendEvent(dp, "xcomfort/%d/event/" + string(event) , 1, event == xc.EventValue, value)
		}
	}, false)

	b.SubscribeAsync(bus.TOPIC_EVENT_DEV_BATTERY, func(dev int, percentage int) {
		r.sendEvent(dev, "xcomfort/%d/battery", 1, true, percentage)
	}, false)

	b.SubscribeAsync(bus.TOPIC_EVENT_DEV_RSSI, func(dev int, rssi int) {
		r.sendEvent(dev, "xcomfort/%d/rssi", 1, true, rssi)
	}, false)

	b.SubscribeAsync(bus.TOPIC_EVENT_DEV_TEMP, func(dev int, temp int) {
		r.sendEvent(dev, "xcomfort/%d/internal_temperature", 1, true, temp)
	}, false)

	b.SubscribeAsync(bus.TOPIC_EVENT_DPL_CHANGED, func() {
		log.Printf("DPL Changed\n")
	}, false)

	r.ctx = ctx

	return r, nil
}

func handleCommand(msg mqttclient.Message, topic string, handler func(int) error ) {
	var dp int
	if _, err := fmt.Sscanf(msg.Topic(), topic, &dp); err != nil {
		log.Println(err)
		return
	}
	if err := handler(dp); err != nil {
		log.Println(err)
	}
}

func (r *MqttRelay) sendEvent(id int, topic string, qos byte, retained bool, payload interface{}) {
	pl := ""
	if payload != nil {
		pl = fmt.Sprint(payload)
	}
	r.client.Publish(fmt.Sprintf(topic, id), qos, retained, pl)
}