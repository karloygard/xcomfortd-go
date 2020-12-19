package xc

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/text/encoding/charmap"
)

func (i *Interface) ReadFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	extension := filepath.Ext(filename)
	switch strings.ToLower(extension) {
	case ".txt":
		if i.devices, i.datapoints, err = i.txtReader(f); err != nil {
			return err
		}
	case ".dpl":
		if i.devices, i.datapoints, err = i.dplReader(f); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown file type %s", extension)
	}

	return nil
}

func (i *Interface) txtReader(file io.Reader) (devices map[int]*Device, datapoints map[byte]*Datapoint, err error) {
	r := csv.NewReader(file)
	r.Comma = '\t'
	r.FieldsPerRecord = 9

	datapoints = make(map[byte]*Datapoint)
	devices = make(map[int]*Device)

	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, errors.WithStack(err)
		}

		serialNo, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		datapoint, err := strconv.Atoi(record[0])
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		deviceType, err := strconv.Atoi(record[3])
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		channel, err := strconv.Atoi(record[4])
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		mode, err := strconv.Atoi(record[5])
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		device, exists := devices[serialNo]
		if !exists {
			device = &Device{
				serialNumber: serialNo,
				deviceType:   DeviceType(deviceType),
				iface:        i,
			}
			devices[serialNo] = device
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
		datapoints[byte(datapoint)] = dp

		if i.verbose {
			log.Printf("Datapoint %d: device %s, serial %d, channel %d, '%s'",
				dp.number, dp.device.deviceType, dp.device.serialNumber, dp.channel, dp.name)
		}
	}

	return devices, datapoints, nil
}

func (i *Interface) dplReader(in io.ReadSeeker) (devices map[int]*Device, datapoints map[byte]*Datapoint, err error) {
	dec := charmap.Windows1252.NewDecoder()

	datapoints = make(map[byte]*Datapoint)
	devices = make(map[int]*Device)

	basicHeader := make([]byte, 16)

	if _, err := io.ReadFull(in, basicHeader); err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if basicHeader[0] != DPL_TYPE_EXT2 {
		return nil, nil, ErrUnknownDPLFormat
	}

	numberBasicEntries := int(basicHeader[8]&0xf)<<8 + int(basicHeader[9])

	basicEntries := make([]byte, 16*numberBasicEntries)
	if _, err := io.ReadFull(in, basicEntries); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	extendedHeader := make([]byte, int(basicHeader[11]))
	if _, err := in.Seek(int64(binary.LittleEndian.Uint32(basicHeader[12:16])), io.SeekStart); err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if _, err := io.ReadFull(in, extendedHeader); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	textList := make([]byte, binary.LittleEndian.Uint16(extendedHeader[114:116]))
	if _, err := in.Seek(int64(binary.LittleEndian.Uint32(extendedHeader[116:120])), io.SeekStart); err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if _, err := io.ReadFull(in, textList); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	locationName := make(map[uint16]string)
	for len(textList) > 0 {
		id := binary.LittleEndian.Uint16(textList[:2])
		length := int(textList[2])
		name := textList[3:length]

		locationName[id] = string(name)

		textList = textList[length:]
	}

	if _, err := in.Seek(int64(binary.LittleEndian.Uint32(basicHeader[12:16])+uint32(basicHeader[11])), io.SeekStart); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	for j := 0; j < numberBasicEntries; j++ {
		extendedEntry := make([]byte, basicEntries[11])
		if _, err := io.ReadFull(in, extendedEntry); err != nil {
			return nil, nil, errors.WithStack(err)
		}

		serialNo := int(binary.LittleEndian.Uint32(basicEntries[2:6]))
		deviceType := binary.LittleEndian.Uint16(basicEntries[6:8])

		utf8name, err := dec.Bytes(bytes.Trim(extendedEntry[:53], "\x00"))
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		device, exists := devices[serialNo]
		if !exists {
			device = &Device{
				serialNumber: serialNo,
				deviceType:   DeviceType(deviceType),
				iface:        i,
			}
			devices[serialNo] = device
		}

		dp := &Datapoint{
			device:  device,
			name:    strings.Join(strings.Fields(strings.TrimSpace(string(utf8name))), " "),
			number:  byte(binary.LittleEndian.Uint16(basicEntries[:2])),
			channel: int(basicEntries[8]),
			mode:    int(basicEntries[9]),
			sensor:  basicEntries[10] != 0,
		}

		device.datapoints = append(device.datapoints, dp)
		datapoints[byte(dp.number)] = dp

		if i.verbose {
			log.Printf("Datapoint %d: device %s, serial %d, channel %d, mode %d, '%s'",
				dp.number, dp.device.deviceType, dp.device.serialNumber, dp.channel, dp.mode, dp.name)

			//log.Printf("SW version [%d, %d]", extendedEntry[53], extendedEntry[54])
			if extendedEntry[55] != 0 {
				log.Printf("Level: %d.%d.%d, location [%s, %s, %s]",
					extendedEntry[55], extendedEntry[58], extendedEntry[61],
					locationName[binary.LittleEndian.Uint16(extendedEntry[56:58])],
					locationName[binary.LittleEndian.Uint16(extendedEntry[59:61])],
					locationName[binary.LittleEndian.Uint16(extendedEntry[62:64])])
			}
		}

		basicEntries = basicEntries[16:]
	}

	return devices, datapoints, nil
}
