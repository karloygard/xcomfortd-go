package xc

type SignalStrength byte

func (s SignalStrength) String() string {
	switch {
	case s <= 67:
		return "good"
	case s <= 75:
		return "normal"
	case s <= 90:
		return "weak"
	case s <= 120:
		return "very weak"
	default:
		return "error"
	}
}
