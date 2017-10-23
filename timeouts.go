package kr

import (
	"time"
)

type TimeoutPhases struct {
	Alert time.Duration
	Fail  time.Duration
}

type Timeouts struct {
	Me       TimeoutPhases
	Pair     TimeoutPhases
	Sign     TimeoutPhases
	ACKDelay time.Duration
}

func DefaultTimeouts() Timeouts {
	return Timeouts{
		Me: TimeoutPhases{
			Alert: 4 * time.Second,
			Fail:  5 * time.Second,
		},
		Pair: TimeoutPhases{
			Alert: 4 * time.Second,
			Fail:  90 * time.Second,
		},
		Sign: TimeoutPhases{
			Alert: 2 * time.Second,
			Fail:  30 * time.Second,
		},
		ACKDelay: 60 * time.Second,
	}
}
