package xc

import (
	"io"
)

func StartStopWrap(w io.ReadWriteCloser) io.ReadWriteCloser {
	return StartStopWrapper{w}
}

type StartStopWrapper struct {
	w io.ReadWriteCloser
}

func (s StartStopWrapper) Read(p []byte) (n int, err error) {
	if _, err = s.w.Read(p[:1]); err != nil {
		return
	}

	if p[0] != MCI_SER_START {
		return 0, errStartStopByte
	}

	if _, err = s.w.Read(p[:1]); err != nil {
		return
	}

	packetLength := int(p[0])
	if len(p) < packetLength+1 {
		return 0, io.ErrShortBuffer
	}
	if n, err = io.ReadFull(s.w, p[1:packetLength+1]); err != nil {
		return
	}

	if p[packetLength] != MCI_SER_STOP {
		return 0, errStartStopByte
	}

	return
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
