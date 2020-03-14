package xc

import "context"

/* New switching actuator output channels:

   0 = status
   1 = binary input
   2 = energy
   3 = power
   4 = load error */

const (
	CSAU_0101_10   int = 0
	CSAU_0101_10I      = 1 // Binary input
	CSAU_0101_10IE     = 3 // Binary input, Energy function
	CSAU_0101_16       = 4
	CSAU_0101_16I      = 5  // Binary input
	CSAU_0101_16IE     = 7  // Binary input, Energy function
	CSAP_01XX_12E      = 14 // Energy function
)

var switchNames = map[int]string{
	CSAU_0101_10:   "CSAU 01/01-10",
	CSAU_0101_10I:  "CSAU 01/01-10I",
	CSAU_0101_10IE: "CSAU 01/01-10IE",
	CSAU_0101_16:   "CSAU 01/01-16",
	CSAU_0101_16I:  "CSAU 01/01-16I",
	CSAU_0101_16IE: "CSAU 01/01-16IE",
	CSAP_01XX_12E:  "CSAP 01/xx-12E",
}

func (d *Datapoint) Switch(ctx context.Context, on bool) ([]byte, error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	if on {
		return d.device.iface.sendTxCommand(ctx, []byte{d.number, TX_EVENT_SWITCH, TX_EVENTDATA_ON})
	} else {
		return d.device.iface.sendTxCommand(ctx, []byte{d.number, TX_EVENT_SWITCH, TX_EVENTDATA_OFF})
	}
}
