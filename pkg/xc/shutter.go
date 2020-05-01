package xc

import (
	"context"
	"log"
)

type ShutterCommand byte

const (
	ShutterClose     ShutterCommand = TX_EVENTDATA_CLOSE
	ShutterOpen                     = TX_EVENTDATA_OPEN
	ShutterStop                     = TX_EVENTDATA_JSTOP
	ShutterStepClose                = TX_EVENTDATA_STEP_CLOSE
	ShutterStepOpen                 = TX_EVENTDATA_STEP_OPEN
)

type ShutterStatus string

const (
	ShutterStopped ShutterStatus = "stopped"
	ShutterOpening               = "opening"
	ShutterClosing               = "closing"
)

func (d *Datapoint) Shutter(ctx context.Context, cmd ShutterCommand) ([]byte, error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	return d.device.iface.sendTxCommand(ctx, []byte{d.number, TX_EVENT_JALO, byte(cmd)})
}

func (d *Datapoint) shutterStatus(h Handler, status byte) (string, error) {
	switch status {
	case RX_IS_STOP:
		h.StatusShutter(d, ShutterStopped)
		return "status shutter stopped", nil
	case RX_IS_OPEN:
		h.StatusShutter(d, ShutterOpening)
		return "status shutter opening", nil
	case RX_IS_CLOSE:
		h.StatusShutter(d, ShutterClosing)
		return "status shutter closing", nil
	default:
		log.Printf("unknown shutter status %d\n", status)
		return "unknown", errMsgNotHandled
	}
}
