package bus

import (
	"context"

	evbus "github.com/asaskevich/EventBus"
	"github.com/karloygard/xcomfortd-go/pkg/xc"
)

func CreateMessageBus(ctx context.Context, verbose bool) (*xc.Interface, evbus.Bus) {
	bus := evbus.New()

	eventHandler := &eventHandler{bus: bus}
	eventHandler.Init(eventHandler, verbose)

	createCommandHandler(ctx, &eventHandler.Interface, bus)

	return &eventHandler.Interface, bus
}