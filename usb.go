package main

import (
	"context"
	"io"
	"log"

	"github.com/google/gousb"
	"github.com/pkg/errors"
)

type usbDevice struct {
	ctx  context.Context
	in   *gousb.InEndpoint
	out  *gousb.OutEndpoint
	dev  *gousb.Device
	done func()
}

func (u usbDevice) Read(p []byte) (n int, err error) {
	return u.in.ReadContext(u.ctx, p)
}

func (u usbDevice) Write(p []byte) (n int, err error) {
	return u.out.WriteContext(u.ctx, p)
}

func (u usbDevice) Close() error {
	u.done()
	return u.dev.Close()
}

func openUsbDevice(ctx context.Context, d *gousb.Device) (io.ReadWriteCloser, error) {
	if err := d.SetAutoDetach(true); err != nil {
		return nil, errors.WithStack(err)
	}

	intf, done, err := d.DefaultInterface()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	inEp, err := intf.InEndpoint(1)
	if err != nil {
		done()
		return nil, errors.WithStack(err)
	}

	outEp, err := intf.OutEndpoint(2)
	if err != nil {
		done()
		return nil, errors.WithStack(err)
	}

	log.Printf("Opened USB device %v", d)

	return usbDevice{
		ctx:  ctx,
		in:   inEp,
		out:  outEp,
		dev:  d,
		done: done,
	}, nil
}

func openUsbDevices(ctx context.Context) (devices []io.ReadWriteCloser, done func() error, err error) {
	usb := gousb.NewContext()
	done = usb.Close

	devs, err := usb.OpenDevices(func(d *gousb.DeviceDesc) bool {
		if d.Vendor == gousb.ID(0x188a) && d.Product == gousb.ID(0x1101) {
			return true
		}
		return false
	})

	for i := range devs {
		if d, openErr := openUsbDevice(ctx, devs[i]); openErr != nil {
			err = openErr
		} else {
			devices = append(devices, d)
		}
	}

	return
}
