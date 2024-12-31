package main

import (
	"context"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	dpFilenameEnvVar = "DATAPOINT_FILENAME"
	mqttServerEnvVar = "MQTT_SERVER"

	appVersion = "0.66"
)

func main() {
	app := cli.NewApp()

	app.Version = appVersion
	app.Usage = "an xComfort daemon"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "file",
			Aliases: []string{"f"},
			EnvVars: []string{dpFilenameEnvVar},
			Usage:   "Datapoint file exported from MRF software",
		},
		&cli.StringFlag{
			Name:    "client-id",
			Aliases: []string{"i"},
			Value:   "xcomfort",
			Usage:   "MQTT client id",
		},
		&cli.StringFlag{
			Name:    "server",
			Aliases: []string{"s"},
			EnvVars: []string{mqttServerEnvVar},
			Usage:   "MQTT server (format tcp://username:password@host:port)",
		},
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "More logging",
		},
		&cli.BoolFlag{
			Name:    "eprom",
			Aliases: []string{"e"},
			Usage:   "Read datapoints from eprom",
		},
		&cli.BoolFlag{
			Name:    "hadiscovery",
			Aliases: []string{"hd"},
			Usage:   "Enable Home Assistant MQTT Discovery",
		},
		&cli.StringFlag{
			Name:    "hadiscoveryprefix",
			Aliases: []string{"hp"},
			Value:   "homeassistant",
			Usage:   "Home Assistant discovery prefix",
		},
		&cli.BoolFlag{
			Name:    "hadiscoveryremove",
			Aliases: []string{"hr"},
			Usage:   "Home Assistant discovery autoremove",
		},
		&cli.BoolFlag{
			Name:  "hidapi",
			Usage: "Use hidapi for usb communication",
		},
		&cli.StringSliceFlag{
			Name:  "host",
			Usage: "Host names/IP addresses of ECI",
		},
	}
	app.Action = openDevices

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("%+v", err)
	}
}

func openDevices(c *cli.Context) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-sigs:
		case <-ctx.Done():
		}
	}()

	log.SetOutput(logRedacter{log.Writer()})

	if c.Bool("verbose") {
		log.Printf("Starting %s, version %s", c.App.Name, c.App.Version)
	}

	var devices []io.ReadWriteCloser

	if c.Bool("hidapi") {
		devices, err = openHidDevices()
	} else {
		var done func() error
		devices, done, err = openUsbDevices(ctx)
		defer done()
	}
	defer func() {
		for i := range devices {
			devices[i].Close()
		}
	}()
	if err != nil {
		return err
	}

	d, err := openEciDevices(ctx, c.StringSlice("host"))
	devices = append(devices, d...)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		log.Println("No devices found")
		return nil
	}

	var wg sync.WaitGroup
	for i := range devices {
		dev := devices[i]
		wg.Add(1)
		go func(id int) {
			if err := run(ctx, dev, c, id); err != nil {
				log.Println(err)
				cancel()
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	return nil
}

func run(ctx context.Context, conn io.ReadWriteCloser,
	cliContext *cli.Context, id int) error {

	relay := &MqttRelay{}

	relay.Init(relay, cliContext.Bool("verbose"))

	if cliContext.String("file") != "" {
		if err := relay.ReadFile(cliContext.String("file")); err != nil {
			return err
		}
	}

	url, err := url.Parse(cliContext.String("server"))
	if err != nil {
		return errors.WithStack(err)
	}

	if cliContext.Bool("hadiscovery") {
		relay.SetupHADiscovery(cliContext.String("hadiscoveryprefix"),
			cliContext.Bool("hadiscoveryremove"))
	}

	if err := relay.Connect(ctx, cliContext.String("client-id"), url, id); err != nil {
		return err
	}
	defer relay.Close()

	go func() {
		// Some sanity checking
		hwrev, rfrev, fwrev, err := relay.Revision()
		if err != nil {
			log.Fatalf("%+v", err)
		}
		log.Printf("CI HW/RF/FW revision: %d, %.1f, %d",
			hwrev, float32(rfrev)/10, fwrev)
		if rfrev < 90 {
			log.Println("This software may not work well with RF Revision < 9.0")
		}

		rf, fw, err := relay.Release()
		if err != nil {
			log.Fatalf("%+v", err)
		}
		log.Printf("CI RF/Firmware release: %.2f, %.2f", rf, fw)

		serial, err := relay.Serial()
		if err != nil {
			log.Fatalf("%+v", err)
		}
		log.Printf("CI serial number: %d", serial)

		if err := relay.SetOKMRF(); err != nil {
			log.Fatalf("%+v", err)
		}
		if err := relay.SetRfSeqNo(); err != nil {
			log.Fatalf("%+v", err)
		}

		if cliContext.Bool("eprom") {
			if err := relay.RequestDPL(ctx); err != nil {
				log.Fatalf("%+v", err)
			}
		}

		if err := relay.HADiscoveryAdd(); err != nil {
			log.Fatalf("%+v", err)
		}
	}()

	defer relay.HADiscoveryRemove()

	return relay.Run(ctx, conn)
}
