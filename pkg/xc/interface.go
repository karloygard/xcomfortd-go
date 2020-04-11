package xc

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Interface
type Interface struct {
	datapoints map[byte]*Datapoint
	devices    map[int]*Device

	// tx command queue
	txCommandQueue chan request
	txSemaphore    *semaphore.Weighted

	// config command queue
	configCommandQueue chan request
	configMutex        sync.Mutex

	handler Handler
}

type Event string

const (
	EventOn           Event = "on"
	EventOff                = "off"
	EventSwitchOn           = "switchOn"
	EventSwitchOff          = "switchOff"
	EventUpPressed          = "upPressed"
	EventUpReleased         = "upReleased"
	EventDownPressed        = "downPressed"
	EventDownReleased       = "downReleased"
	EventForced             = "forced"
	EventSingleOn           = "singleOn"
	EventValue              = "value"
	EventTooCold            = "tooCold"
	EventTooWarm            = "tooWarm"
)

func (e Event) String() string {
	return string(e)
}

// Handler interface for receiving callbacks
type Handler interface {
	// Datapoint updated value
	StatusValue(datapoint *Datapoint, value int)
	// Datapoint updated state
	StatusBool(datapoint *Datapoint, on bool)
	// Datapoint sent event
	Event(datapoint *Datapoint, event Event)
	// Datapoint sent event with value
	ValueEvent(datapoint *Datapoint, event Event, value interface{})
	// Battery state updated
	Battery(device *Device, percentage int)
	// Internal temperature updated
	InternalTemperature(device *Device, centigrade int)
	// Rssi updated
	Rssi(device *Device, rssi int)
}

// Device returns the device with the specified serialNumber
func (i *Interface) Device(serialNumber int) *Device {
	return i.devices[serialNumber]
}

// Datapoint returns the requested datapoint
func (i *Interface) Datapoint(number int) *Datapoint {
	return i.datapoints[byte(number)]
}

// ForEachDatapoint takes a function as input and will apply that function to each
// datapoint that is registered.
func (i *Interface) ForEachDatapoint(dpfunc func(*Datapoint) error) error {
	for _, v := range i.datapoints {
		if err := dpfunc(v); err != nil {
			return err
		}
	}
	return nil
}

// ForEachDevice takes a function as input and will apply that function to each
// device that is registered.
func (i *Interface) ForEachDevice(devfunc func(*Device) error) error {
	for _, v := range i.devices {
		if err := devfunc(v); err != nil {
			return err
		}
	}
	return nil
}

type request struct {
	command    []byte
	responseCh chan []byte
}

// Init loads datapoints from the specified file and takes a handler which
// will get callbacks when events are received.
func (i *Interface) Init(filename string, handler Handler, verbose bool) error {
	i.datapoints = make(map[byte]*Datapoint)
	i.devices = make(map[int]*Device)
	i.handler = handler

	// Only allow four tx commands in parallel
	i.txSemaphore = semaphore.NewWeighted(4)
	i.txCommandQueue = make(chan request)

	i.configCommandQueue = make(chan request)

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
		mode, err := strconv.Atoi(record[5])
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
			name:    strings.Join(strings.Fields(strings.TrimSpace(record[1])), " "),
			number:  byte(datapoint),
			channel: channel,
			mode:    mode,
			sensor:  record[6] == "1",
		}
		device.datapoints = append(device.datapoints, dp)
		i.datapoints[byte(datapoint)] = dp

		if verbose {
			log.Printf("Dp %d: device %s, serial %d, channel %d, '%s'",
				dp.number, dp.device.deviceType, dp.device.serialNumber, dp.channel, dp.name)
		}
	}

	return nil
}
