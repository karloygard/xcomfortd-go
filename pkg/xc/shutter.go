package xc

import (
	"context"
	"log"
)

type ShutterCommand byte

const (
	ShutterClose     ShutterCommand = MCI_TED_CLOSE
	ShutterOpen      ShutterCommand = MCI_TED_OPEN
	ShutterStop      ShutterCommand = MCI_TED_JSTOP
	ShutterStepClose ShutterCommand = MCI_TED_STEP_CLOSE
	ShutterStepOpen  ShutterCommand = MCI_TED_STEP_OPEN
)

type ShutterStatus string

const (
	ShutterStateStopped ShutterStatus = "stopped"
	ShutterStateOpening ShutterStatus = "opening"
	ShutterStateClosing ShutterStatus = "closing"
	ShutterStateClosed  ShutterStatus = "closed"
	ShutterStateOpen    ShutterStatus = "open"

	ShutterStateUnknown ShutterStatus = "unknown"
)

func (d *Datapoint) Shutter(ctx context.Context, cmd ShutterCommand) ([]byte, error) {
	d.queue.Lock()
	defer d.queue.Unlock()

	return d.device.iface.sendTxCommand(ctx, []byte{d.number, MCI_TE_JALO, byte(cmd)})
}

func (d *Datapoint) shutterStatus(h Handler, status byte) (string, error) {
	switch status {
	case RX_IS_STOP:
		h.StatusShutter(d, ShutterStateStopped)
		return "status shutter stopped", nil
	case RX_IS_OPEN:
		h.StatusShutter(d, ShutterStateOpening)
		return "status shutter opening", nil
	case RX_IS_CLOSE:
		h.StatusShutter(d, ShutterStateClosing)
		return "status shutter closing", nil
	default:
		log.Printf("unknown shutter status %d\n", status)
		return "unknown", errMsgNotHandled
	}
}

func (d *Device) extendedStatusShutter(h Handler, data []byte) {
	log.Printf("====Subtype: %d\n", d.subtype)

	status := data[1]

	shutterState := ShutterStateOpen

	switch status {
	case CJAU_OPEN:
	case CJAU_CLOSED:
		shutterState = ShutterStateClosed
	default:
		shutterState = ShutterStateStopped
	}

	for _, dp := range d.datapoints {
		if dp.channel == 0 {
			// Status channel is always 0
			h.StatusShutter(dp, shutterState)
			break
		}
	}

	log.Printf("Device %d, type %s sent extended status message: {shutterState: %s, closedPercentage: %d})\n", d.serialNumber, names[d.deviceType].name, shutterState, status)
}

const (
	CJAU_OPEN    = 0x0
	CJAU_CLOSED  = 0x64
	CJAU_STOPPED = 0xFF
)
