package xc

import (
	"encoding/hex"
	"io"
	"log"
)

func StartStopWrap(w io.ReadWriteCloser) io.ReadWriteCloser {
	return StartStopWrapper{w}
}

type StartStopWrapper struct {
	w io.ReadWriteCloser
}

func (s StartStopWrapper) Read(p []byte) (n int, err error) {
	if n, err = s.w.Read(p); err != nil {
		return 0, err
	}

	if n < 3 {
		log.Printf("Received short packet: %s, buffer length %d",
			hex.EncodeToString(p[:n]), len(p))
		return 0, errShortPacket
	}

	packetLength := int(p[1])
	if n < packetLength+2 {
		log.Printf("Received incomplete or garbage packet: %s, buffer length %d",
			hex.EncodeToString(p[:n]), len(p))
		return 0, errShortPacket
	}

	if p[0] != MCI_SER_START ||
		p[packetLength+1] != MCI_SER_STOP {
		return 0, errStartStopByte
	}

	copy(p, p[1:packetLength+1])
	return n - 2, nil
}

func (s StartStopWrapper) Write(p []byte) (int, error) {
	return s.w.Write(append(append([]byte{MCI_SER_START}, p...), MCI_SER_STOP))
}

func (s StartStopWrapper) Close() error {
	return s.w.Close()
}

type prependLength struct {
	w io.Writer
}

func (w prependLength) Write(p []byte) (int, error) {
	return w.w.Write(append([]byte{byte(len(p) + 1)}, p...))
}
