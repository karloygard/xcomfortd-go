package xc

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"sync"

	"golang.org/x/sync/semaphore"
)

type Interface struct {
	datapoints map[byte]*Datapoint
	devices    map[int]*Device

	txCommandQueue chan request
	txSemaphore    *semaphore.Weighted

	configCommandQueue chan request
	configMutex        sync.Mutex

	handler Handler
}

type request struct {
	command  []byte
	consumer chan []byte
}

type Handler interface {
	StatusValue(datapoint, value int)
	StatusBool(datapoint int, on bool)
}

func (i *Interface) Device(serialNumber int) *Device {
	return i.devices[serialNumber]
}

func (i *Interface) Datapoint(number int) *Datapoint {
	return i.datapoints[byte(number)]
}

func (i *Interface) Init(filename string, handler Handler) error {
	i.datapoints = make(map[byte]*Datapoint)
	i.devices = make(map[int]*Device)

	i.txCommandQueue = make(chan request)
	i.txSemaphore = semaphore.NewWeighted(4)

	i.configCommandQueue = make(chan request)

	i.handler = handler

	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = 9

	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		serialNo, err := strconv.Atoi(record[2])
		if err != nil {
			return err
		}
		datapoint, err := strconv.Atoi(record[0])
		if err != nil {
			return err
		}
		deviceType, err := strconv.Atoi(record[3])
		if err != nil {
			return err
		}
		channel, err := strconv.Atoi(record[4])
		if err != nil {
			return err
		}

		device, exists := i.devices[serialNo]
		if !exists {
			device = &Device{
				serialNumber: serialNo,
				deviceType:   DeviceType(deviceType),
				iface:        i,
			}
			i.devices[serialNo] = device
		}

		dp := &Datapoint{
			device:  device,
			name:    record[1],
			number:  byte(datapoint),
			channel: channel,
		}
		device.Datapoints = append(device.Datapoints, dp)
		i.datapoints[byte(datapoint)] = dp
	}

	return nil
}
