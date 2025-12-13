package xc

import (
	"context"
	"encoding/binary"
	"log"
)

func (d *Datapoint) AsyncDesiredTemperature(value float32) {
	d.asyncDesiredTemperature = value
}

func (d *Datapoint) AsyncCurrentTemperature(value float32) {
	d.asyncCurrentTemperature = value
}

func (d *Datapoint) asyncSendTemperatures(ctx context.Context,
	currentTemperature float32) {

	d.queue.Lock()
	defer d.queue.Unlock()

	setpoint := make([]byte, 2)
	current := make([]byte, 2)

	if d.asyncCurrentTemperature != -1 {
		// Use user provided temperature if set
		currentTemperature = d.asyncCurrentTemperature
	}

	binary.BigEndian.PutUint16(setpoint, uint16(d.asyncDesiredTemperature*10))
	binary.BigEndian.PutUint16(current, uint16(currentTemperature*10))

	if _, err := d.device.iface.sendTxCommand(ctx, []byte{
		d.number,
		MCI_TE_HRV_IN,
		setpoint[0], setpoint[1],
		current[0], current[1],
	}); err != nil {
		log.Printf("WARNING: command for datapoint %d failed: %v", d.number, err)
	}
}
