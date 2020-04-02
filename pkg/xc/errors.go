package xc

import "errors"

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
