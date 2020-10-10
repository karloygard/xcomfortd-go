package main

import (
	"context"
	"fmt"
	"log"

	"github.com/karloygard/xcomfortd-go/pkg/xc"

	"github.com/google/gousb"
	"github.com/pkg/errors"
)

func Usb(ctx context.Context, number int, x *xc.Interface) error {
	goCtx := gousb.NewContext()
	defer goCtx.Close()

	// Open any device with a given VID/PID using a convenience function.
	dev, err := goCtx.OpenDeviceWithVIDPID(0x188a, 0x1101)
	if err != nil {
		return errors.WithStack(err)
	}
	if dev == nil {
		return fmt.Errorf("Couldn't find USB stick")
	}

	defer dev.Close()

	if err := dev.SetAutoDetach(true); err != nil {
		return errors.WithStack(err)
	}

	intf, done, err := dev.DefaultInterface()
	if err != nil {
		return errors.WithStack(err)
	}
	defer done()

	inEp, err := intf.InEndpoint(1)
	if err != nil {
		return errors.WithStack(err)
	}

	outEp, err := intf.OutEndpoint(2)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Println("Opened USB device")

	return x.Run(ctx, inEp, outEp)
}
