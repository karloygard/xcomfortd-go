package busproxy

import (
	evbus "github.com/asaskevich/EventBus"
	"github.com/karloygard/xcomfortd-go/pkg/bus"
)

func CreateHaSync(b evbus.Bus) {
	b.SubscribeAsync(bus.TOPIC_COMMAND_DIMMER, func (dp int, brightness int) {
		b.Publish(bus.TOPIC_EVENT_DP_STATUS_BOOL, dp, brightness > 0)
	}, false)
}
