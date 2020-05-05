package xc

import "errors"

var (
	ErrTerminal          = errors.New("Terminal error")
	ErrGeneral           = errors.New("General error")
	ErrUnknown           = errors.New("Message unknown")
	ErrDpOutOfRange      = errors.New("Datapoint out of range")
	ErrBusyMRF           = errors.New("RF busy, TX msg lost")
	ErrBusyMRFRX         = errors.New("RF busy, RX in progress")
	ErrTxMsgLost         = errors.New("TX lost, repeat it, buffer full")
	ErrNoAck             = errors.New("Timeout, no ACK received")
	ErrUnrecognisedError = errors.New("Unknown error")
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

func retryableError(err error) bool {
	return err == ErrDpOutOfRange ||
		err == ErrBusyMRF ||
		err == ErrBusyMRFRX ||
		err == ErrTxMsgLost ||
		err == ErrNoAck
}
