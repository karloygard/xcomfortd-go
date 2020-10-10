package xc

import (
	"context"
	"encoding/hex"
	"io"
	"log"

	"github.com/pkg/errors"
)

const commandRetries = 2

// Run starts the event loop, dispatching TX and CONFIG commands,
// and returning the results to the requesters.
func (i *Interface) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	input := make(chan []byte)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer cancel()
		buf := make([]byte, 32)
		for {
			if _, err := in.Read(buf); err != nil {
				log.Printf("read failed: %+v\n", errors.WithStack(err))
				return
			}
			input <- buf[1:buf[0]]
		}
	}()

	var txWaiters waithandler
	var configWaiter, extendedWaiter chan []byte

	defer func() {
		txWaiters.Close()
		if configWaiter != nil {
			configWaiter <- nil
		}
		if extendedWaiter != nil {
			extendedWaiter <- nil
		}
	}()

	for {
		select {
		case o := <-i.setupChan:
			i.devices = o.devices
			i.datapoints = o.datapoints
			o.done <- true

		case o := <-i.txCommandChan:
			// Send TX command
			seq := txWaiters.Add(o.responseCh)
			data := append([]byte{byte(len(o.command) + 2)}, o.command...)

			if _, err := out.Write(append(data, byte(seq<<4))); err != nil {
				return errors.WithStack(err)
			}

		case o := <-i.configCommandChan:
			// Send CONFIG command
			configWaiter = o.responseCh
			msg := append([]byte{byte(len(o.command) + 1)}, o.command...)
			if i.verbose {
				log.Printf("CONFIG: [%s]\n", hex.EncodeToString(msg))
			}
			if _, err := out.Write(msg); err != nil {
				return errors.WithStack(err)
			}

		case o := <-i.extendedCommandChan:
			// Send EXTENDED command
			extendedWaiter = o.responseCh
			if _, err := out.Write(append([]byte{byte(len(o.command) + 1)}, o.command...)); err != nil {
				return errors.WithStack(err)
			}

		case in := <-input:
			switch in[0] {
			case MCI_PT_RX:
				if i.verbose {
					log.Printf("RX: [%s]\n", hex.EncodeToString(in))
				}

				if err := i.rx(in[1:]); err != nil {
					if errors.Is(err, errMsgNotHandled) {
						log.Printf("Message not handled [%s]\n", hex.EncodeToString(in))
					} else {
						return err
					}
				}
			case MCI_PT_STATUS:
				if i.verbose {
					log.Printf("STATUS: [%s]\n", hex.EncodeToString(in))
				}

				switch in[1] {
				case MCI_STT_ERROR:
					seqPos := 3
					if in[2] == MCI_STS_GENERAL || in[2] == STATUS_DATA {
						seqPos = 4
					}
					txWaiters.Resume(in[1:], int(in[seqPos]>>4))
				case MGW_STT_OK:
					switch in[2] {
					case STATUS_OK_MRF:
						switch in[4] {
						case STATUS_DATA_OKMRF_ACK_DIRECT, STATUS_DATA_OKMRF_ACK_ROUTED:
							txWaiters.Resume(in[1:], int(in[3]>>4))
						}
					case STATUS_OK_CONFIG:
						// doesn't matter what we return here
						configWaiter <- in[2:]
						configWaiter = nil
					}
				case MCI_STT_TIMEACCOUNT:
					if in[2] != STATUS_DATA {
						break
					}
					fallthrough
				case MGW_STT_SERIAL,
					MGW_STT_RELEASE:
					configWaiter <- in[2:]
					configWaiter = nil
				default:
					log.Printf("<- %s", hex.EncodeToString(in))
				}
			case MCI_PT_EXTENDED:
				if i.verbose {
					log.Printf("EPROM: [%s]\n", hex.EncodeToString(in))
				}

				if in[1] == MCI_ET_DPL_CHANGED {
					i.handler.DPLChanged()
					go i.RequestDPL(ctx)
				} else {
					extendedWaiter <- in[1:]
					extendedWaiter = nil
				}

			default:
				log.Printf("Unknown message received: %08x", in[0])
			}

		case <-ctx.Done():
			log.Println("Exiting")
			return nil
		}
	}
}

func (i *Interface) sendTxCommand(ctx context.Context, command []byte) ([]byte, error) {
	if err := i.txSemaphore.Acquire(ctx, 1); err != nil {
		return nil, errors.WithStack(err)
	}
	defer i.txSemaphore.Release(1)

	for retry := 0; ; retry++ {
		waitCh := make(chan []byte)
		i.txCommandChan <- request{append([]byte{byte(MCI_PT_TX)}, command...), waitCh}
		res := <-waitCh

		if i.verbose {
			log.Printf("RX: [%s]\n", hex.EncodeToString(res))
		}

		if len(res) > 0 {
			switch res[0] {
			case MCI_STT_ERROR:
				err := errorMessage(res[1:])
				if retryableError(err) && retry < commandRetries {
					log.Printf("TX command failed, retrying (%d/%d): %v", retry+1, commandRetries, err)
					continue
				}
				return nil, errors.WithStack(err)
			case MGW_STT_OK:
				return res[1:], nil
			}
		}

		return nil, errors.WithStack(ErrTerminal)
	}
}

func (i *Interface) sendConfigCommand(command []byte) ([]byte, error) {
	i.configMutex.Lock()
	defer i.configMutex.Unlock()

	waitCh := make(chan []byte)
	i.configCommandChan <- request{append([]byte{byte(MCI_PT_CONFIG)}, command...), waitCh}
	res := <-waitCh

	if len(res) == 0 {
		return nil, errors.WithStack(ErrTerminal)
	}

	return res, nil
}

func (i *Interface) sendExtendedCommand(command []byte) ([]byte, error) {
	waitCh := make(chan []byte)
	i.extendedCommandChan <- request{append([]byte{byte(MCI_PT_EXTENDED)}, command...), waitCh}
	res := <-waitCh

	if len(res) == 0 {
		return nil, errors.WithStack(ErrTerminal)
	}

	return res, nil
}
