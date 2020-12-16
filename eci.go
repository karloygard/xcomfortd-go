package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/karloygard/xcomfortd-go/pkg/xc"

	"github.com/pkg/errors"
)

const eciPort = 7153

func openEciDevices(ctx context.Context, hosts []string) (devices []io.ReadWriteCloser, err error) {
	var dialer net.Dialer

	for i := range hosts {
		var device io.ReadWriteCloser
		hostPort := fmt.Sprintf("%s:%d", hosts[i], eciPort)

		if device, err = dialer.DialContext(ctx, "tcp", hostPort); err != nil {
			err = errors.WithStack(err)
			return
		}

		log.Printf("Connected to ECI (%s)", hostPort)

		devices = append(devices, xc.StartStopWrap(device))
	}

	return
}
