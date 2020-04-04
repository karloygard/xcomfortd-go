package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli"
)

const (
	dpFilenameEnvVar = "DATAPOINT_FILENAME"
	clientIdEnvVar   = "MQTT_CLIENT_ID"
	mqttServerEnvVar = "MQTT_SERVER"
)

func main() {
	app := cli.NewApp()

	app.Version = "0.0.1 (alpha)"
	app.Usage = "an xComfort daemon"
	app.Commands = []cli.Command{
		cli.Command{
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
			Action: func(cliContext *cli.Context) error {
				ctx := context.Background()
				relay := MqttRelay{}

				if err := relay.Init(cliContext.String("file"), &relay, cliContext.Bool("verbose")); err != nil {
					return err
				}

				url, err := url.Parse(cliContext.String("server"))
				if err != nil {
					return err
				}

				if err := relay.Connect(ctx, cliContext.String("client-id"), url); err != nil {
					return err
				}

				go func() {
					serial, err := relay.Serial()
					if err != nil {
						panic(err)
					}
					log.Printf("CI serial number: %d", serial)

					rf, fw, err := relay.Release()
					if err != nil {
						panic(err)
					}
					log.Printf("CI RF/Firmware release: %.2f, %.2f", rf, fw)
				}()

				if cliContext.Bool("hadiscovery") {
					discoveryPrefix := cliContext.String("hadiscoveryprefix")

					relay.HADiscoveryAdd(discoveryPrefix)

					// On exit clean up
					sigs := make(chan os.Signal, 1)
					signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
					go func() {
						<-sigs
						relay.HADiscoveryRemove(discoveryPrefix)
						time.Sleep(1 * time.Second) // Wait a second to deliver the messages
						os.Exit(0)
					}()
				}

				return Usb(ctx, cliContext.Int("device-number"), &relay.Interface)
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
