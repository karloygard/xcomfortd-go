package main

import (
	"bytes"
	"io"
)

type logRedacter struct {
	io.Writer
}

var interruptedError = []byte("libusb: interrupted [code -10]")

func (l logRedacter) Write(data []byte) (int, error) {
	if bytes.Contains(data, interruptedError) {
		return len(data), nil
	}
	return l.Write(data)
}
