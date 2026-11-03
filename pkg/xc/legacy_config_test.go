package xc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func configTestConnection(t *testing.T) (*Interface, net.Conn) {
	t.Helper()
	var iface Interface
	iface.Init(nil, false)
	host, device := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- iface.Run(ctx, host) }()
	t.Cleanup(func() {
		cancel()
		_ = device.Close()
		_ = host.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("interface loop did not stop")
		}
	})
	return &iface, device
}

func expectConfigTestCommand(t *testing.T, device net.Conn, want []byte) {
	t.Helper()
	_ = device.SetReadDeadline(time.Now().Add(2 * time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(device, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("command %x, want %x", got, want)
	}
}

func sendConfigTestReply(t *testing.T, device net.Conn, reply []byte) {
	t.Helper()
	_ = device.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := device.Write(reply); err != nil {
		t.Fatal(err)
	}
}

func awaitConfigTestResult(t *testing.T, done <-chan error, want error) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, want) {
			t.Fatalf("got %v, want %v", err, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not complete after gateway reply")
	}
}

func TestCapturedLegacyConfigRejectionReturnsError(t *testing.T) {
	iface, device := configTestConnection(t)
	done := make(chan error, 1)
	go func() { done <- iface.SetRfSeqNo() }()
	expectConfigTestCommand(t, device, []byte{4, 0xb2, 0x1e, 1})
	// CKOZ-00/03 RF 6.00 / firmware 2.10 rejects SEND_RFSEQNO this way.
	sendConfigTestReply(t, device, []byte{8, 0xc3, 9, 1, 0x95, 0, 0, 0})
	awaitConfigTestResult(t, done, ErrUnknown)
}

func TestSuccessfulConfigReplyStillCompletesRequest(t *testing.T) {
	iface, device := configTestConnection(t)
	done := make(chan error, 1)
	go func() { done <- iface.SetOKMRF() }()
	expectConfigTestCommand(t, device, []byte{4, 0xb2, 4, 1})
	sendConfigTestReply(t, device, []byte{8, 0xc3, 0x1c, 5, 0, 0, 0, 0})
	awaitConfigTestResult(t, done, nil)
}

func TestRevisionReplyStillReturnsValues(t *testing.T) {
	iface, device := configTestConnection(t)
	done := make(chan error, 1)
	go func() {
		hw, rf, fw, err := iface.Revision()
		if err == nil && (hw != 0 || rf != 60 || fw != 650) {
			err = errors.New("incorrect revision values")
		}
		done <- err
	}()
	expectConfigTestCommand(t, device, []byte{4, 0xb2, 0x1b, 0x10})
	sendConfigTestReply(t, device, []byte{8, 0xc3, 0x1b, 0x10, 0, 60, 2, 0x8a})
	awaitConfigTestResult(t, done, nil)
}

func TestUnknownExtendedReplyStillReturnsError(t *testing.T) {
	for _, code := range []byte{ERR_T_TYPE, ERR_T_PACKET_TYPE} {
		t.Run(generalErrorString[code], func(t *testing.T) {
			iface, device := configTestConnection(t)
			done := make(chan error, 1)
			go func() { _, err := iface.sendExtendedCommand([]byte{0x20}); done <- err }()
			expectConfigTestCommand(t, device, []byte{3, 0xd1, 0x20})
			sendConfigTestReply(t, device, []byte{8, 0xc3, 9, 1, code, 0, 0, 0})
			awaitConfigTestResult(t, done, ErrUnknown)
		})
	}
}

func TestUnknownRepliesDistinguishConcurrentConfigAndExtendedRequests(t *testing.T) {
	for _, configFirst := range []bool{true, false} {
		name := "extended rejection first"
		if configFirst {
			name = "config rejection first"
		}
		t.Run(name, func(t *testing.T) {
			iface, device := configTestConnection(t)
			configDone, extendedDone := make(chan error, 1), make(chan error, 1)
			go func() { configDone <- iface.SetRfSeqNo() }()
			expectConfigTestCommand(t, device, []byte{4, 0xb2, 0x1e, 1})
			go func() { _, err := iface.sendExtendedCommand([]byte{0x20}); extendedDone <- err }()
			expectConfigTestCommand(t, device, []byte{3, 0xd1, 0x20})
			first, second := extendedDone, configDone
			firstCode, secondCode := byte(ERR_T_PACKET_TYPE), byte(ERR_T_TYPE)
			if configFirst {
				first, second = configDone, extendedDone
				firstCode, secondCode = ERR_T_TYPE, ERR_T_PACKET_TYPE
			}
			sendConfigTestReply(t, device, []byte{8, 0xc3, 9, 1, firstCode, 0, 0, 0})
			awaitConfigTestResult(t, first, ErrUnknown)
			select {
			case err := <-second:
				t.Fatalf("other request completed before its reply: %v", err)
			default:
			}
			sendConfigTestReply(t, device, []byte{8, 0xc3, 9, 1, secondCode, 0, 0, 0})
			awaitConfigTestResult(t, second, ErrUnknown)
		})
	}
}
