package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/karloygard/xcomfortd-go/pkg/bus"
	"github.com/karloygard/xcomfortd-go/pkg/busproxy"
	"github.com/karloygard/xcomfortd-go/pkg/front/mqtt"
	"github.com/karloygard/xcomfortd-go/pkg/xc"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	dpFilenameEnvVar = "DATAPOINT_FILENAME"
	clientIdEnvVar   = "MQTT_CLIENT_ID"
	mqttServerEnvVar = "MQTT_SERVER"
)

func main() {
	app := cli.NewApp()

	app.Version = "0.13"
	app.Usage = "an xComfort daemon"
	app.Commands = []cli.Command{
		{
			Name:    "hid",
			Aliases: []string{"h"},
			Usage:   "connect via HID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Value: os.Getenv(dpFilenameEnvVar),
					Usage: "Datapoint file exported from MRF software",
				},
				cli.IntFlag{
					Name:  "device-number, d",
					Usage: "USB device number, if more than one is available",
				},
				cli.StringFlag{
					Name:  "client-id, i",
					Value: os.Getenv(clientIdEnvVar),
					Usage: "MQTT client id",
				},
				cli.StringFlag{
					Name:  "server, s",
					Value: os.Getenv(mqttServerEnvVar),
					Usage: "MQTT server (format tcp://username:password@host:port)",
				},
				cli.BoolFlag{
					Name:  "verbose, v",
					Usage: "More logging",
				},
				cli.BoolFlag{
					Name:  "eprom, e",
					Usage: "Read datapoints from eprom",
				},
				cli.BoolFlag{
					Name:  "hadiscovery, hd",
					Usage: "Enable Home Assistant MQTT Discovery",
				},
				cli.StringFlag{
					Name:  "hadiscoveryprefix, hp",
					Value: "homeassistant",
					Usage: "Home Assistant discovery prefix",
				},
			},
			Action: hidCommand,
		},
		{
			Name:    "usb",
			Aliases: []string{"u"},
			Usage:   "connect via USB",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Value: os.Getenv(dpFilenameEnvVar),
					Usage: "Datapoint file exported from MRF software",
				},
				cli.IntFlag{
					Name:  "device-number, d",
					Usage: "USB device number, if more than one is available",
				},
				cli.StringFlag{
					Name:  "client-id, i",
					Value: os.Getenv(clientIdEnvVar),
					Usage: "MQTT client id",
				},
				cli.StringFlag{
					Name:  "server, s",
					Value: os.Getenv(mqttServerEnvVar),
					Usage: "MQTT server (format tcp://username:password@host:port)",
				},
				cli.BoolFlag{
					Name:  "verbose, v",
					Usage: "More logging",
				},
				cli.BoolFlag{
					Name:  "eprom, e",
					Usage: "Read datapoints from eprom",
				},
				cli.BoolFlag{
					Name:  "hadiscovery, hd",
					Usage: "Enable Home Assistant MQTT Discovery",
				},
				cli.StringFlag{
					Name:  "hadiscoveryprefix, hp",
					Value: "homeassistant",
					Usage: "Home Assistant discovery prefix",
				},
			},
			Action: usbCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

func usbCommand(cliContext *cli.Context) error {
	return start(Usb, cliContext)
}

func hidCommand(cliContext *cli.Context) error {
	return start(Hid, cliContext)
}

func start(comm func(ctx context.Context, number int, x *xc.Interface) error, cliContext *cli.Context) error {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	
	// Setup exit handler
	go func() {
		defer cancel()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-sigs:
		case <-ctx.Done():
		}
	}()

	// Create bridge
	xci, evbus := bus.CreateMessageBus(ctx, cliContext.Bool("verbose"))
	busproxy.CreateHaSync(evbus)

	// Read file
	if cliContext.String("file") != "" {
		if err := xci.ReadFile(cliContext.String("file")); err != nil {
			return err
		}
	}

	// Some sanity checking
	go func() {
		hwrev, rfrev, fwrev, err := xci.Revision()
		if err != nil {
			log.Fatalf("%+v", err)
		}
		log.Printf("CI HW/RF/FW revision: %d, %.1f, %d", hwrev, float32(rfrev)/10, fwrev)
		if rfrev < 90 {
			log.Println("This software may not work well with RF Revision < 9.0")
		}

		rf, fw, err := xci.Release()
		if err != nil {
			log.Fatalf("%+v", err)
		}
		log.Printf("CI RF/Firmware release: %.2f, %.2f", rf, fw)

		serial, err := xci.Serial()
		if err != nil {
			log.Fatalf("%+v", err)
		}
		log.Printf("CI serial number: %d", serial)

		if cliContext.Bool("eprom") {
			if err := xci.RequestDPL(ctx); err != nil {
				log.Fatalf("%+v", err)
			}
		}
	}()

	url, err := url.Parse(cliContext.String("server"))
	if err != nil {
		return errors.WithStack(err)
	}

	broker, err := mqtt.CreateMqttRelay(ctx, cliContext.String("client-id"), url, evbus)
	if err != nil {
		return errors.WithStack(err)
	}
	defer broker.Close()

	
	if cliContext.Bool("hadiscovery") {
		if err := broker.SetupHADiscovery(xci, cliContext.String("hadiscoveryprefix")); err != nil {
			log.Fatalf("%+v", err)
		}
	}

	if cliContext.Bool("hadiscovery") {
		defer broker.HADiscoveryRemove(xci)
	}

	return comm(ctx, cliContext.Int("device-number"), xci)
}
