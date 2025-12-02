package xc

import (
	"sync"

	"golang.org/x/sync/semaphore"
)

// Interface
type Interface struct {
	datapoints map[byte]*Datapoint
	devices    map[int]*Device

	// tx command queue
	txCommandChan chan request
	txSemaphore   *semaphore.Weighted

	// config command queue
	configCommandChan chan request
	configMutex       sync.Mutex

	// extended command queue
	extendedCommandChan chan request
	extendedMutex       sync.Mutex

	setupChan chan datapoints

	verbose bool
	handler Handler
}

type Event string

const (
	EventOn           Event = "on"
	EventOff          Event = "off"
	EventSwitchOn     Event = "switchOn"
	EventSwitchOff    Event = "switchOff"
	EventUpPressed    Event = "upPressed"
	EventUpReleased   Event = "upReleased"
	EventDownPressed  Event = "downPressed"
	EventDownReleased Event = "downReleased"
	EventForced       Event = "forced"
	EventSingleOn     Event = "singleOn"
	EventValue        Event = "value"
	EventTooCold      Event = "tooCold"
	EventTooWarm      Event = "tooWarm"
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
	// Datapoint updated shutter state
	StatusShutter(datapoint *Datapoint, status ShutterStatus)
	// Datapoint sent event
	Event(datapoint *Datapoint, event Event)
	// RC data wheel position
	Wheel(datapoint *Datapoint, value interface{})
	// HRV valve position
	Valve(datapoint *Datapoint, position int)
	// HRV mode
	Mode(datapoint *Datapoint, heating string)
	// Datapoint sent event with value
	ValueEvent(datapoint *Datapoint, event Event, value interface{})
	// Datapoint sent value
	Value(datapoint *Datapoint, value interface{})
	// Battery state updated
	Battery(device *Device, percentage int)
	// Power updated
	Power(device *Device, value interface{})
	// Internal temperature updated
	InternalTemperature(device *Device, centigrade int)
	// Rssi updated
	Rssi(device *Device, rssi int)
	// Datapoint list changed
	DPLChanged()
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

type datapoints struct {
	devices    map[int]*Device
	datapoints map[byte]*Datapoint
	done       chan bool
}

// Init loads datapoints from the specified file and takes a handler which
// will get callbacks when events are received.
func (i *Interface) Init(handler Handler, verbose bool) {
	i.datapoints = make(map[byte]*Datapoint)
	i.devices = make(map[int]*Device)

	i.handler = handler
	i.verbose = verbose

	// Only allow four tx commands in parallel
	i.txSemaphore = semaphore.NewWeighted(4)
	i.txCommandChan = make(chan request)

	i.configCommandChan = make(chan request)
	i.extendedCommandChan = make(chan request)

	i.setupChan = make(chan datapoints)
}
