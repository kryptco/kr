package gatt

import (
	"fmt"

	"github.com/currantlabs/ble"
)

// Filter is the interface that wraps the basic Filt method.
//
// Filter returns true if the advertisement matches specified condition.
type Filter interface {
	Filter(a ble.Advertisement) bool
}

// FilterFunc is an adapter to allow the use of ordinary functions as Filters.
type FilterFunc func(a ble.Advertisement) bool

// Filter returns true if the adversisement matches specified condition.
func (f FilterFunc) Filter(a ble.Advertisement) bool {
	return f(a)
}

// Discover searches for and connects to a Peripheral which matches specified condition.
func Discover(f Filter) (ble.Client, error) {
	ch := make(chan ble.Advertisement)
	fn := func(a ble.Advertisement) {
		if !f.Filter(a) {
			return
		}
		StopScanning()
		ch <- a
	}
	if err := SetAdvHandler(ble.AdvHandlerFunc(fn)); err != nil {
		return nil, fmt.Errorf("can't set adv handler: %s", err)
	}
	if err := Scan(false); err != nil {
		return nil, fmt.Errorf("can't scan: %s", err)
	}

	a := <-ch

	cln, err := Dial(a.Address())
	if err != nil {
		return nil, fmt.Errorf("can't dial: %s", err)
	}
	return cln, nil
}
