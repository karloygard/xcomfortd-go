package xc

import (
	"context"
	"encoding/hex"
	"io"
	"log"
	"time"

	"github.com/pkg/errors"
)

const (
	commandRetries     = 2
	lostCommandTimeout = 10
)

// Run starts the event loop, dispatching TX and CONFIG commands,
// and returning the results to the requesters.
func (i *Interface) Run(ctx context.Context, conn io.ReadWriter) error {
	input := make(chan []byte)
	ctx, cancel := context.WithCancel(ctx)

	out := prependLength{conn}

	go func() {
		defer cancel()
		buf := make([]byte, 256)
		for {
			if n, err := conn.Read(buf); err != nil {
				log.Printf("read failed: %+v", errors.WithStack(err))
				return
			} else if n > 0 {
				if buf[0] > 0 {
					input <- buf[1:buf[0]]
				} else {
					log.Printf("Ignoring unexpected input [%s]", hex.EncodeToString(buf[:n]))
				}
			}
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
			seq, waiters := txWaiters.Add(o.responseCh)
			tx := append(o.command, byte(seq<<4))
			if i.verbose {
				log.Printf("TX (seq %x, %d parallel): [%s] ",
					seq, waiters, hex.EncodeToString(tx))
			}
			if _, err := out.Write(tx); err != nil {
				return errors.WithStack(err)
			}

		case o := <-i.configCommandChan:
			// Send CONFIG command
			configWaiter = o.responseCh
			if i.verbose {
				log.Printf("CONFIG: [%s]", hex.EncodeToString(o.command))
			}
			if _, err := out.Write(o.command); err != nil {
				return errors.WithStack(err)
			}

		case o := <-i.extendedCommandChan:
			// Send EXTENDED command
			extendedWaiter = o.responseCh
			if i.verbose {
				log.Printf("EXTENDED: [%s]", hex.EncodeToString(o.command))
			}
			if _, err := out.Write(o.command); err != nil {
				return errors.WithStack(err)
			}

		case in := <-input:
			switch in[0] {
			case MCI_PT_RX:
				if i.verbose {
					log.Printf("RX: [%s]", hex.EncodeToString(in))
				}

				if err := i.rx(in[1:]); err != nil {
					if errors.Is(err, errMsgNotHandled) {
						log.Printf("Message not handled [%s]",
							hex.EncodeToString(in))
					} else {
						return err
					}
				}
			case MCI_PT_STATUS:
				if i.verbose {
					log.Printf("STATUS: [%s]", hex.EncodeToString(in))
				}

				switch in[1] {
				case MCI_STT_ERROR:
					seqPos := 3

					switch in[2] {
					case MCI_STS_UNKNOWN:
						// If this fails, allocate seqno 0 to extended cmds only
						if extendedWaiter != nil {
							extendedWaiter <- in[1:]
							extendedWaiter = nil
						}
					case MCI_STS_GENERAL:
						seqPos = 4
						fallthrough
					default:
						txWaiters.Resume(in[1:], int(in[seqPos]>>4))
					}
				case MGW_STT_OK:
					switch in[2] {
					case STATUS_OK_MRF:
						switch in[4] {
						case STATUS_DATA_OKMRF_NOINFO,
							STATUS_DATA_OKMRF_ACK_DIRECT,
							STATUS_DATA_OKMRF_ACK_ROUTED:
							txWaiters.Resume(in[1:], int(in[3]>>4))
						}
					case STATUS_OK_CONFIG:
						// doesn't matter what we return here
						if configWaiter != nil {
							configWaiter <- in[2:]
							configWaiter = nil
						}
					}
				case MCI_STT_TIMEACCOUNT:
					switch in[2] {
					case STATUS_DATA:
						log.Printf("Timeaccount %d%%", in[3])
					case STATUS_IS_0:
						log.Printf("Timeaccount zero, no more transmission possible")
					case STATUS_LESS_10:
						log.Printf("Timeaccount fell below 10%%")
					case STATUS_MORE_15:
						log.Printf("Timeaccount climbed above 15%%")
					}
				case MGW_STT_SERIAL,
					MGW_STT_RELEASE,
					MGW_STT_SEND_OK_MRF,
					MCI_STT_COUNTER_RX,
					MCI_STT_COUNTER_TX:
					if configWaiter != nil {
						configWaiter <- in[2:]
						configWaiter = nil
					}
				default:
					log.Printf("<- %s", hex.EncodeToString(in))
				}
			case MCI_PT_EXTENDED:
				if i.verbose {
					log.Printf("EPROM: [%s]", hex.EncodeToString(in))
				}

				switch in[1] {
				case MCI_ET_DPL_CHANGED:
					go func() {
						if err := i.RequestDPL(ctx); err != nil {
							log.Println(err)
						} else {
							i.handler.DPLChanged()
						}
					}()

				case MCI_ET_REPLY, MCI_ET_SEND_DPL:
					if extendedWaiter != nil {
						extendedWaiter <- in[1:]
						extendedWaiter = nil
					}

				case MCI_ET_STL_CHANGED, MCI_ET_SEND_STL:
					log.Printf("Status list messages currently ignored: %02x", in[1])

				default:
					log.Printf("Unknown extended message received: %02x", in[1])
				}

			default:
				log.Printf("Unknown message received: %s", hex.EncodeToString(in))
			}

		case <-txWaiters.OldestExpiring(lostCommandTimeout):
			log.Println("TX message was silently lost, likely never sent")
			txWaiters.ResumeOldest([]byte{MCI_STT_ERROR, MCI_STS_NO_ACK})

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

		if len(res) > 0 {
			switch res[0] {
			case MCI_STT_ERROR:
				err := errorMessage(res[1:])
				if retryableError(err) && retry < commandRetries {
					log.Printf("TX command failed, retrying (%d/%d): %v",
						retry+1, commandRetries, err)
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

	for {
		waitCh := make(chan []byte)
		i.configCommandChan <- request{append([]byte{byte(MCI_PT_CONFIG)}, command...), waitCh}

		select {
		case res := <-waitCh:
			if len(res) == 0 {
				return nil, errors.WithStack(ErrTerminal)
			}

			return res, nil
		case <-time.After(5 * time.Second):
			log.Println("Stick didn't respond after five seconds, retrying command")
		}
	}
}

func (i *Interface) sendExtendedCommand(command []byte) ([]byte, error) {
	for {
		waitCh := make(chan []byte)
		i.extendedCommandChan <- request{append([]byte{byte(MCI_PT_EXTENDED)}, command...), waitCh}

		select {
		case res := <-waitCh:
			if len(res) < 2 {
				return nil, errors.WithStack(ErrTerminal)
			}

			if res[0] == MCI_STT_ERROR {
				return nil, errors.WithStack(errorMessage(res[1:]))
			}
			return res, nil

		case <-time.After(5 * time.Second):
			log.Println("Stick didn't respond after five seconds, retrying command")
		}
	}
}
