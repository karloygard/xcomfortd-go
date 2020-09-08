package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

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

	app.Version = "0.2 (alpha)"
	app.Usage = "an xComfort daemon"
	app.Commands = []cli.Command{
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
					Usage: "USB device number, if more than one is available (default 0)",
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
					Name:  "hadiscovery, hd",
					Usage: "Enable Home Assistant MQTT Discovery",
				},
				cli.StringFlag{
					Name:  "hadiscoveryprefix, hp",
					Value: "homeassistant",
					Usage: "Home Assistant discovery prefix",
				},
			},
			Action: usb,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

func usb(cliContext *cli.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	relay := MqttRelay{}

	go func() {
		defer cancel()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-sigs:
		case <-ctx.Done():
		}
	}()

	if err := relay.Init(cliContext.String("file"), &relay, cliContext.Bool("verbose")); err != nil {
		return err
	}

	url, err := url.Parse(cliContext.String("server"))
	if err != nil {
		return errors.WithStack(err)
	}

	if err := relay.Connect(ctx, cliContext.String("client-id"), url); err != nil {
		return err
	}
	defer relay.Close()

	go func() {
		// Some sanity checking
		hwrev, rfrev, fwrev, err := relay.Revision()
		if err != nil {
			panic(err)
		}
		log.Printf("CI HW/RF/FW revision: %d, %.1f, %d", hwrev, float32(rfrev)/10, fwrev)
		if rfrev < 90 {
			log.Println("This software may not work well with RF Revision < 9.0")
		}

		rf, fw, err := relay.Release()
		if err != nil {
			panic(err)
		}
		log.Printf("CI RF/Firmware release: %.2f, %.2f", rf, fw)

		serial, err := relay.Serial()
		if err != nil {
			panic(err)
		}
		log.Printf("CI serial number: %d", serial)
	}()

	if cliContext.Bool("hadiscovery") {
		discoveryPrefix := cliContext.String("hadiscoveryprefix")

		relay.HADiscoveryAdd(discoveryPrefix)
		defer relay.HADiscoveryRemove(discoveryPrefix)
	}

	return Usb(ctx, cliContext.Int("device-number"), &relay.Interface)
}
