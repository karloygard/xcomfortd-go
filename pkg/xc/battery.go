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

func (b BatteryState) percentage() int {
	switch b {
	case 1:
		return 20
	case 2:
		return 40
	case 3:
		return 60
	case 4:
		return 80
	case 5:
		return 100
	default:
		return 0
	}
}
