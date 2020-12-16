package main

import (
	"io"
	"log"

	"github.com/karalabe/hid"
	"github.com/pkg/errors"
)

func openHidDevices() (devices []io.ReadWriteCloser, err error) {
	devs := hid.Enumerate(0x188a, 0x1101)

	for i := range devs {
		var device io.ReadWriteCloser

		if device, err = devs[i].Open(); err != nil {
			err = errors.WithStack(err)
			return
		}

		log.Printf("Opened HID device %d\n", i)

		devices = append(devices, device)
	}

	return
}
