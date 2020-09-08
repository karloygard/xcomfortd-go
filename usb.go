package main

import (
	"context"
	"fmt"
	"log"

	"github.com/karloygard/xcomfortd-go/pkg/xc"

	"github.com/karalabe/hid"
	"github.com/pkg/errors"
)

func Usb(ctx context.Context, number int, x *xc.Interface) error {
	devices := hid.Enumerate(0x188a, 0x1101)
	if len(devices) < number+1 {
		return fmt.Errorf("Couldn't find USB stick")
	}

	device, err := devices[number].Open()
	if err != nil {
		return errors.WithStack(err)
	}
	defer device.Close()

	log.Printf("Opened USB device %d\n", number)

	return x.Run(ctx, device, device)
}
