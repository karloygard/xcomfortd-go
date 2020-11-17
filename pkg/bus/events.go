package bus

import (
	evbus "github.com/asaskevich/EventBus"
	"github.com/karloygard/xcomfortd-go/pkg/xc"
)

const (
	TOPIC_EVENT_DP_STATUS_VALUE = "event.dp.status_value"
	TOPIC_EVENT_DP_STATUS_BOOL = "event.dp.status_bool"
	TOPIC_EVENT_DP_STATUS_SHUTTER = "event.dp.status_shutter"
	TOPIC_EVENT_DP_EVENT = "event.dp.event"
	TOPIC_EVENT_DEV_BATTERY = "event.dev.battery"
	TOPIC_EVENT_DEV_TEMP = "event.dev.temperature"
	TOPIC_EVENT_DEV_RSSI = "event.dev.rssi"
	TOPIC_EVENT_DPL_CHANGED = "event.dpl.changed"
)

type eventHandler struct {
	xc.Interface
	
	bus evbus.Bus
}

func (e *eventHandler) StatusValue(datapoint *xc.Datapoint, value int) {
	e.bus.Publish(TOPIC_EVENT_DP_STATUS_VALUE, datapoint.Number(), value)
}

func (e *eventHandler) StatusBool(datapoint *xc.Datapoint, state bool) {
	e.bus.Publish(TOPIC_EVENT_DP_STATUS_BOOL, datapoint.Number(), state)
}

func (e *eventHandler) StatusShutter(datapoint *xc.Datapoint, status xc.ShutterStatus) {
	e.bus.Publish(TOPIC_EVENT_DP_STATUS_SHUTTER, datapoint.Number(), datapoint.Number(), status)
}

func (e *eventHandler) Event(datapoint *xc.Datapoint, event xc.Event) {
	e.bus.Publish(TOPIC_EVENT_DP_EVENT, datapoint.Number(), event, nil)
}

func (e *eventHandler) ValueEvent(datapoint *xc.Datapoint, event xc.Event, value interface{}) {
	e.bus.Publish(TOPIC_EVENT_DP_EVENT, datapoint.Number(), event, value)
}

func (e *eventHandler) Battery(device *xc.Device, percentage int) {
	e.bus.Publish(TOPIC_EVENT_DEV_BATTERY, device.SerialNumber(), percentage)
}

func (e *eventHandler) InternalTemperature(device *xc.Device, centigrade int) {
	e.bus.Publish(TOPIC_EVENT_DEV_TEMP, device.SerialNumber(), centigrade)
}

func (e *eventHandler) Rssi(device *xc.Device, rssi int) {
	e.bus.Publish(TOPIC_EVENT_DEV_RSSI, device.SerialNumber(), rssi)
}

func (e *eventHandler) DPLChanged() {
	e.bus.Publish(TOPIC_EVENT_DPL_CHANGED)
}