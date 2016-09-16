package darwin

// State ...
type State int

// State ...
const (
	StateUnknown      State = 0
	StateResetting    State = 1
	StateUnsupported  State = 2
	StateUnauthorized State = 3
	StatePoweredOff   State = 4
	StatePoweredOn    State = 5
)

func (s State) String() string {
	str := []string{
		"Unknown",
		"Resetting",
		"Unsupported",
		"Unauthorized",
		"PoweredOff",
		"PoweredOn",
	}
	return str[int(s)]
}
