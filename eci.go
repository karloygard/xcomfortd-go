package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/karloygard/xcomfortd-go/pkg/xc"
)

const eciPort = 7153

func Eci(ctx context.Context, host string, x *xc.Interface) error {
	hostPort := fmt.Sprintf("%s:%d", host, eciPort)
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return err
	}

	log.Printf("Connected to ECI (%s)", hostPort)

	stream := xc.StartStopWrap(conn)

	return x.Run(ctx, stream, stream)
}
