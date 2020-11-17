package bus

import (
	"context"
	"log"

	evbus "github.com/asaskevich/EventBus"

	"github.com/karloygard/xcomfortd-go/pkg/xc"
)

const (
	TOPIC_COMMAND_SWITCH = "command.switch"
	TOPIC_COMMAND_DIMMER = "command.dimmer"
	TOPIC_COMMAND_SHUTTER = "command.shutter"
)

func createCommandHandler(ctx context.Context, xci *xc.Interface, bus evbus.Bus) {
	bus.SubscribeAsync(TOPIC_COMMAND_SWITCH, func (dp int, on bool) {
		if datapoint := xci.Datapoint(dp); datapoint != nil {
			if _, err := datapoint.Switch(ctx, on); err != nil {
				log.Printf("WARNING: command for datapoint %d failed, state now unknown: %v\n", dp, err)
			} else {
				bus.Publish(TOPIC_EVENT_DP_STATUS_BOOL, dp, on)
			}
		} else {
			log.Printf("unknown datapoint %d\n", dp)
		}
	}, false)

	bus.SubscribeAsync(TOPIC_COMMAND_DIMMER, func (dp int, brightness int) {
		if datapoint := xci.Datapoint(dp); datapoint != nil {
			if _, err := datapoint.Dim(ctx, brightness); err != nil {
				log.Printf("WARNING: command for datapoint %d failed, state now unknown: %v\n", dp, err)
			} else {
				bus.Publish(TOPIC_EVENT_DP_STATUS_VALUE, dp, brightness)
			}
		} else {
			log.Printf("unknown datapoint %d\n", dp)
		}
	}, false)

	bus.SubscribeAsync(TOPIC_COMMAND_SHUTTER, func (dp int, command xc.ShutterCommand) {
		if datapoint := xci.Datapoint(dp); datapoint != nil {
			if _, err := datapoint.Shutter(ctx, command); err != nil {
				log.Printf("WARNING: command for datapoint %d failed, state now unknown: %v\n", dp, err)
			}
		} else {
			log.Printf("unknown datapoint %d\n", dp)
		}
	}, false)
}


