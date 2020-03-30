package xc

type BatteryState byte

func (b BatteryState) String() string {
	switch b {
	case 0:
		return "N/A"
	case 1:
		return "empty"
	case 2:
		return "very weak"
	case 3:
		return "weak"
	case 4:
		return "good"
	case 5:
		return "new"
	case 16:
		return "mains-powered"
	default:
		return "error"
	}
}
