package xc

import "io"

func StartStopWrap(w io.ReadWriter) io.ReadWriter {
	return StartStopWrapper{w}
}

type StartStopWrapper struct {
	w io.ReadWriter
}

func (s StartStopWrapper) Read(p []byte) (n int, err error) {
	if n, err = s.w.Read(p); err != nil {
		return 0, err
	}

	if n < 3 {
		return 0, ErrShortPacket
	}

	packetLength := int(p[1])
	if n < packetLength+2 {
		return 0, ErrShortPacket
	}

	if p[0] != MCI_SER_START ||
		p[packetLength+1] != MCI_SER_STOP {
		return 0, ErrStartStopByte
	}

	copy(p, p[1:packetLength+1])
	return n - 2, nil
}

func (s StartStopWrapper) Write(p []byte) (int, error) {
	return s.w.Write(append(append([]byte{MCI_SER_START}, p...), MCI_SER_STOP))
}

type prependLength struct {
	w io.Writer
}

func (w prependLength) Write(p []byte) (int, error) {
	return w.w.Write(append([]byte{byte(len(p) + 1)}, p...))
}
