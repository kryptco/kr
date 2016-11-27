package kr

import (
	"testing"
	"time"
)

func TrueBefore(t *testing.T, predicate func() bool, deadline time.Time) {
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		<-time.After(time.Millisecond)
	}
	t.Fatal("predicate unsatisfied by deadline")
}
