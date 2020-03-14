package main

import (
	"context"
	"fmt"
	"log"

	"github.com/karloygard/xcomfortd-go/xc"

	"github.com/karalabe/hid"
)

func Usb(ctx context.Context, number int, x *xc.Interface) error {
	devices := hid.Enumerate(0x188a, 0x1101)
	if len(devices) < number+1 {
		return fmt.Errorf("Couldn't find USB stick")
	}

	device, err := devices[number].Open()
	if err != nil {
		return err
	}

	log.Printf("connected to usb device\n")

	return x.Run(ctx, device, device)
}
