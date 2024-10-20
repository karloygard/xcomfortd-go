package main

import (
	"context"
	"io"
	"log"
	"sort"

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
	buf := p[:u.in.Desc.MaxPacketSize]
	return u.in.ReadContext(u.ctx, buf)
}

func (u usbDevice) Write(p []byte) (n int, err error) {
	return u.out.WriteContext(u.ctx, p)
}

func (u usbDevice) Close() error {
	u.done()
	return u.dev.Close()
}

func openUsbDevice(ctx context.Context, d *gousb.Device) (io.ReadWriteCloser, string, error) {
	if err := d.SetAutoDetach(true); err != nil {
		return nil, "", errors.WithStack(err)
	}

	serial, err := d.SerialNumber()
	if err != nil {
		return nil, "", errors.WithStack(err)
	}

	intf, done, err := d.DefaultInterface()
	if err != nil {
		return nil, "", errors.WithStack(err)
	}

	inEp, err := intf.InEndpoint(1)
	if err != nil {
		done()
		return nil, "", errors.WithStack(err)
	}

	outEp, err := intf.OutEndpoint(2)
	if err != nil {
		done()
		return nil, "", errors.WithStack(err)
	}

	log.Printf("Opened USB device '%v', serial '%s', packet size %d/%d",
		d, serial, inEp.Desc.MaxPacketSize, outEp.Desc.MaxPacketSize)

	return usbDevice{
		ctx:  ctx,
		in:   inEp,
		out:  outEp,
		dev:  d,
		done: done,
	}, serial, nil
}

type kv struct {
	serial string
	device io.ReadWriteCloser
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

	devlist := []kv{}

	for i := range devs {
		if d, serial, openErr := openUsbDevice(ctx, devs[i]); openErr != nil {
			err = openErr
		} else {
			devlist = append(devlist, kv{
				serial: serial,
				device: d,
			})
		}
	}

	// Ensure order doesn't change
	sort.Slice(devlist, func(i, j int) bool {
		return devlist[i].serial < devlist[j].serial
	})

	for _, d := range devlist {
		devices = append(devices, d.device)
	}

	return
}
