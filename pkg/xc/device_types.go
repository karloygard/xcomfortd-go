package xc

type DeviceType int

func (d DeviceType) String() string {
	return names[d].name
}

const (
	DT_CTAA_01     DeviceType = 1
	DT_CTAA_02                = 2
	DT_CTAA_04                = 3
	DT_CRCA_000x              = 5
	DT_CSAx_01                = 16
	DT_CDAx_01                = 17
	DT_CJAU_0101              = 18
	DT_CBEU_0201              = 19
	DT_CBEU_0202              = 20
	DT_CHSZ_1201              = 21
	DT_CHMU_00                = 22
	DT_CTEU_02                = 23
	DT_CAEE_02                = 24
	DT_CAAE_01                = 25
	DT_CRMA_00                = 26
	DT_CJAU_0102              = 27
	DT_CKOZ_00                = 28
	DT_CBMA_02                = 29
	DT_CHSZ_02                = 48
	DT_CHSZ_1203              = 49
	DT_CHSZ_1204              = 50
	DT_CRCA_00                = 51
	DT_CROU_00                = 52
	DT_CIZE_02                = 53
	DT_CEMx_01                = 54
	DT_CHAZ_01                = 55
	DT_CHSZ_01                = 56
	DT_CKOZ_0208              = 57
	DT_CKOZ_0009              = 62
	DT_CHVZ_01                = 65
	DT_CRMA_00_FW             = 67
	ROSETTA_SENSOR            = 68
	DT_CHAZ_0112              = 71
	DT_CSAU_0101              = 74
	DT_CROU_0101              = 75
	DT_CDAx_01NG              = 77
	DT_CRCA_00xx              = 78
	DT_CHAX_010x              = 81
)

// We don't pay attention to channel modes yet, this is a simplification.

type channelType int

const (
	UNKNOWN channelType = iota
	STATUS_BOOL
	STATUS_PERCENT
	STATUS_SHUTTER
	PUSHBUTTON
	SWITCH
	ONOFF
	TEMPERATURE_SWITCH
	VALUE_SWITCH
	HUMIDITY_SWITCH
	MOTION
	ENERGY
	POWER
	CURRENT
	VOLTAGE
	PULSES
)

type deviceInfo struct {
	name     string
	channels []channelType
}

var names = map[DeviceType]deviceInfo{
	DT_CTAA_01:     {"Single pushbutton (CTAA-01/xx)", []channelType{PUSHBUTTON}},
	DT_CTAA_02:     {"Double pushbutton (CTAA-02/xx)", []channelType{PUSHBUTTON, PUSHBUTTON}},
	DT_CTAA_04:     {"Quad pushbutton (CTAA-04/xx)", []channelType{PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON}},
	DT_CRCA_000x:   {"Room Controller (with Switch) (CRCA-00/01..04)", []channelType{TEMPERATURE_SWITCH}},
	DT_CSAx_01:     {"Switching Actuator (CSAx-01/xx)", []channelType{STATUS_BOOL}},
	DT_CDAx_01:     {"Dimming Actuator (CDAx-01/xx)", []channelType{STATUS_PERCENT}},
	DT_CJAU_0101:   {"Shutter Actuator (CJAU-01/01)", []channelType{STATUS_SHUTTER}},
	DT_CBEU_0201:   {"Binary Input, 230V (CBEU-02/01)", []channelType{SWITCH, SWITCH}},
	DT_CBEU_0202:   {"Binary Input, Battery (CBEU-02/02)", []channelType{SWITCH, SWITCH}},
	DT_CHSZ_1201:   {"Remote Control 12 channel (old design) (CHSZ-12/01)", []channelType{PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON}},
	DT_CHMU_00:     {"Home-Manager (CHMU-00/xx)", nil},
	DT_CTEU_02:     {"Temperature Input (CTEU-02/xx)", []channelType{TEMPERATURE_SWITCH, TEMPERATURE_SWITCH}},
	DT_CAEE_02:     {"Analog Input (CAEE-02/xx)", []channelType{VALUE_SWITCH, VALUE_SWITCH}},
	DT_CAAE_01:     {"Analog Actuator (CAAE-01/xx)", []channelType{STATUS_PERCENT}},
	DT_CRMA_00:     {"Room-Manager (CRMA-00/xx)", nil},
	DT_CJAU_0102:   {"Shutter Actuator with Security (CJAU-01/02)", []channelType{STATUS_SHUTTER}},
	DT_CKOZ_00:     {"Communication Interface (CKOZ-00/03)", nil},
	DT_CBMA_02:     {"Motion Detector (CBMA-02/xx)", []channelType{MOTION, MOTION}},
	DT_CHSZ_02:     {"Remote Control 2 channel small (CHSZ-02/02)", []channelType{PUSHBUTTON, PUSHBUTTON}},
	DT_CHSZ_1203:   {"Remote Control 12 channel (CHSZ-12/03)", []channelType{PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON}},
	DT_CHSZ_1204:   {"Remote Control 12 channel with display (CHSZ-12/04)", []channelType{PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON, PUSHBUTTON}},
	DT_CRCA_00:     {"Room Controller with Switch/Humidity (CRCA-00/05)", []channelType{TEMPERATURE_SWITCH, VALUE_SWITCH}},
	DT_CROU_00:     {"Router (no communication possible, just ignore it) (CROU-00/01)", nil},
	DT_CIZE_02:     {"Impulse Input (CIZE-02/01)", []channelType{PULSES, PULSES}},
	DT_CEMx_01:     {"EMS (CEMx-01/01)", []channelType{ENERGY, POWER, CURRENT, VOLTAGE}},
	DT_CHAZ_01:     {"E-Radiator Actuator (CHAZ-01/xx)", nil},
	DT_CHSZ_01:     {"Remote Control Alarm Pushbutton (CHSZ-01/05)", []channelType{PUSHBUTTON}},
	DT_CKOZ_0208:   {"BOSCOS (Bed/Chair Occupancy Sensor) (CKOZ-02/08)", []channelType{SWITCH}},
	DT_CKOZ_0009:   {"MEP (CKOZ-00/09)", nil},
	DT_CHVZ_01:     {"HRV (CHVZ-01/03)", nil},
	DT_CRMA_00_FW:  {"Room-Manager (new firmware) (CRMA-00/xx)", nil},
	ROSETTA_SENSOR: {"Rosetta sensor", []channelType{PUSHBUTTON, PUSHBUTTON}},
	DT_CHAZ_0112:   {"Multi Channel Heating Actuator (CHAZ-01/12)", []channelType{ONOFF}},
	DT_CSAU_0101:   {"Switching Actuator New Generation (CSAU-01/01-1xxx)", []channelType{STATUS_BOOL, SWITCH, ENERGY, POWER, ONOFF}},
	DT_CROU_0101:   {"Router New Generation (CROU-01/01-Sx)", []channelType{UNKNOWN, ONOFF, ONOFF, ONOFF, ONOFF}},
	DT_CDAx_01NG:   {"Dimming Actuator New Generation (CDAx-01/xx)", []channelType{STATUS_PERCENT, SWITCH, SWITCH, ENERGY, POWER, ONOFF}},
	DT_CRCA_00xx:   {"Room Controller Touch (CRCA-00/xx)", []channelType{TEMPERATURE_SWITCH, HUMIDITY_SWITCH, UNKNOWN, UNKNOWN, PUSHBUTTON, PUSHBUTTON, TEMPERATURE_SWITCH, SWITCH}},
	DT_CHAX_010x:   {"Heating actuator", []channelType{STATUS_PERCENT, UNKNOWN, ENERGY, ONOFF}},
	//69: "Rosetta Router",
}
