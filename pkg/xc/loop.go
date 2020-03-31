package xc

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"log"
)

var (
	ErrTerminal          = errors.New("terminal error")
	ErrGeneral           = errors.New("general error")
	ErrUnknown           = errors.New("unknown error")
	ErrDpOutOfRange      = errors.New("dp out of range")
	ErrBusyMRF           = errors.New("rf busy, tx msg lost")
	ErrBusyMRFRX         = errors.New("rf busy, rx in progress")
	ErrTxMsgLost         = errors.New("tx lost, buffer full")
	ErrNoAck             = errors.New("no ack")
	ErrUnrecognisedError = errors.New("unknown error")
)

func errorMessage(data []byte) error {
	switch data[0] {
	case STATUS_GENERAL:
		return ErrGeneral
	case STATUS_UNKNOWN:
		return ErrUnknown
	case STATUS_DP_OOR:
		return ErrDpOutOfRange
	case STATUS_BUSY_MRF:
		return ErrBusyMRF
	case STATUS_BUSY_MRF_RX:
		return ErrBusyMRFRX
	case STATUS_TX_MSG_LOST:
		return ErrTxMsgLost
	case STATUS_NO_ACK:
		return ErrNoAck
	default:
		return ErrUnrecognisedError
	}
}

// Run starts the event loop, dispatching TX and CONFIG commands,
// and returning the results to the requesters.
func (i *Interface) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	input := make(chan []byte)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		buf := make([]byte, 32)
		for {
			if _, err := in.Read(buf); err != nil {
				log.Printf("read failed: %s\n", err)
				cancel()
				return
			}
			input <- buf[1:buf[0]]
		}
	}()

	var txWaiters waithandler
	defer txWaiters.Close()

	configWaiter := make(chan []byte)

	for {
		select {
		case o := <-i.txCommandQueue:
			// Send TX command
			seq := txWaiters.Add(o.responseCh)
			data := append([]byte{byte(len(o.command) + 2)}, o.command...)

			if _, err := out.Write(append(data, byte(seq<<4))); err != nil {
				return err
			}

		case o := <-i.configCommandQueue:
			// Send CONFIG command
			configWaiter = o.responseCh
			if _, err := out.Write(append([]byte{byte(len(o.command) + 1)}, o.command...)); err != nil {
				return err
			}

		case in := <-input:
			switch in[0] {
			case MGW_PT_RX:
				if i.rx(in[1:]) == errMsgNotHandled {
					log.Printf("Message not handled [%s]\n", hex.EncodeToString(in))
				}
			case MGW_PT_STATUS:
				switch in[1] {
				case STATUS_TYPE_ERROR:
					seqPos := 3
					if in[2] == STATUS_GENERAL || in[2] == STATUS_DATA {
						seqPos = 4
					}
					txWaiters.Resume(in[1:], int(in[seqPos]>>4))
				case STATUS_TYPE_OK:
					switch in[2] {
					case STATUS_OK_MRF:
						switch in[4] {
						case STATUS_DATA_OKMRF_ACK_DIRECT:
							log.Printf("ack direct %d\n", in[3]>>4)
						case STATUS_DATA_OKMRF_ACK_ROUTED:
							log.Printf("ack routed %d\n", in[3]>>4)
						default:
							continue
						}
						txWaiters.Resume(in[1:], int(in[3]>>4))
					case STATUS_OK_CONFIG:
						log.Printf("ok\n")
					}
				case STATUS_TYPE_SERIAL,
					STATUS_TYPE_TIMEACCOUNT,
					STATUS_TYPE_RELEASE:
					configWaiter <- in[2:]
					configWaiter = nil
				default:
					log.Printf("<- %s\n", hex.EncodeToString(in))
				}
			default:
				log.Printf("Unknown message received: %08x\n", in[0])
			}

		case <-ctx.Done():
			log.Println("Exiting")
			return nil
		}
	}
}

func (i *Interface) sendTxCommand(ctx context.Context, command []byte) ([]byte, error) {
	if err := i.txSemaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer i.txSemaphore.Release(1)

	waitCh := make(chan []byte)
	i.txCommandQueue <- request{append([]byte{byte(MGW_PT_TX)}, command...), waitCh}
	res := <-waitCh

	if len(res) == 0 {
		return nil, ErrTerminal
	}

	switch res[0] {
	case STATUS_TYPE_ERROR:
		return nil, errorMessage(res[1:])
	case STATUS_TYPE_OK:
		return res[1:], nil
	default:
		return nil, ErrTerminal
	}
}

func (i *Interface) sendConfigCommand(command []byte) ([]byte, error) {
	i.configMutex.Lock()
	defer i.configMutex.Unlock()

	waitCh := make(chan []byte)
	i.configCommandQueue <- request{append([]byte{byte(MGW_PT_CONFIG)}, command...), waitCh}
	res := <-waitCh

	if len(res) == 0 {
		return nil, ErrTerminal
	}

	return res, nil
}
