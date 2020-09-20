package xc

import (
	"errors"
	"fmt"
)

var (
	ErrTerminal          = errors.New("Terminal error")
	ErrUnknown           = errors.New("Message unknown")
	ErrDpOutOfRange      = errors.New("Datapoint out of range")
	ErrBusyMRF           = errors.New("RF busy, TX msg lost")
	ErrBusyMRFRX         = errors.New("RF busy, RX in progress")
	ErrTxMsgLost         = errors.New("TX lost, repeat it, buffer full")
	ErrNoAck             = errors.New("Timeout, no ACK received")
	ErrUnrecognisedError = errors.New("Unknown error")

	ErrUnknownDPLFormat  = errors.New("Unknown DPL format")
	ErrUnexpectedReponse = errors.New("Unexpected response")
)

var generalErrorString = map[byte]string{
	ERR_T_SWITCH:          "Invalid SWITCH data",
	ERR_T_PERCENT:         "Invalid PERCENT value",
	ERR_T_DIM:             "Invalid DIM data",
	ERR_T_JALO:            "Invalid JALO data",
	ERR_T_JALO_STEP:       "Invalid JALO_STEP data",
	ERR_T_PUSHBUTTON:      "Invalid PUSHBUTTON data",
	ERR_T_EVENT:           "Invalid EVENT (IN or OUT)",
	ERR_T_TIMEACCOUNT:     "ERR_T_TIMEACCOUNT",
	ERR_T_SEND_OK_MRF:     "ERR_T_SEND_OK_MRF",
	ERR_T_RELEASE:         "Invalid RELEASE mode",
	ERR_T_BACK_TO_FACTORY: "Invalid BACK_TO_FACTORY mode",
	ERR_T_COUNTER_RX:      "Invalid COUNTER_RX mode",
	ERR_T_COUNTER_TX:      "Invalid COUNTER_TX mode",
	ERR_T_TYPE:            "Invalid CONFIG packet TYPE (OUT) ",
	ERR_T_PACKET_TYPE:     "Invalid packet TYPE (OUT)",
	ERR_T_RFREVISION:      "Invalid RF-firmware revision",
	ERR_T_SEND_CLASS:      "Invalid SEND_CLASS mode",
	ERR_T_SEND_RFSEQNO:    "Invalid SEND_RFSEQNO mode",
	ERR_T_BUFFER_FULL:     "Buffer full,wait for OK",
	ERR_T_CRC:             "CRC-Error",
	ERR_T_BM_NO_TARGET:    "Basic Mode: no target available (no actuator in learnmode)",
	ERR_T_DP_NOT_ASSIGNED: "DP is not assigned to actuator",
	ERR_T_VALUE:           "unexpected value",
}

type ErrGeneral struct {
	status byte
}

func (e ErrGeneral) Error() string {
	errorMessage, found := generalErrorString[e.status]
	if !found {
		errorMessage = "unknown error"
	}
	return fmt.Sprintf("General error: %s", errorMessage)
}

func errorMessage(data []byte) error {
	switch data[0] {
	case MCI_STS_GENERAL:
		return ErrGeneral{data[2]}
	case MCI_STS_UNKNOWN:
		return ErrUnknown
	case MCI_STS_DP_OOR:
		return ErrDpOutOfRange
	case MCI_STS_BUSY_MRF:
		return ErrBusyMRF
	case MCI_STS_BUSY_MRF_RX:
		return ErrBusyMRFRX
	case MCI_STS_TX_MSG_LOST:
		return ErrTxMsgLost
	case MCI_STS_NO_ACK:
		return ErrNoAck
	default:
		return ErrUnrecognisedError
	}
}

func retryableError(err error) bool {
	return errors.Is(err, ErrDpOutOfRange) ||
		errors.Is(err, ErrBusyMRF) ||
		errors.Is(err, ErrBusyMRFRX) ||
		errors.Is(err, ErrTxMsgLost) ||
		errors.Is(err, ErrNoAck)
}
