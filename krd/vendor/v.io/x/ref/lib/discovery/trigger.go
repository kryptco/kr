// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import "reflect"

// A Trigger is a simple multi-channel receiver. It triggers callbacks
// when it receives signals from the corresponding channels.
// Each callback will run only once and run in a separate goroutine.
type Trigger struct {
	addCh chan *addRequest
}

type addRequest struct {
	callback func()
	sc       reflect.SelectCase
}

// Add enqueues callback to be invoked on the next event on ch.
func (tr *Trigger) Add(callback func(), ch <-chan struct{}) {
	sc := reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	tr.addCh <- &addRequest{callback, sc}
}

// Stop stops the trigger.
func (tr *Trigger) Stop() {
	close(tr.addCh)
}

func (tr *Trigger) loop() {
	callbacks := make([]func(), 1, 4)
	cases := make([]reflect.SelectCase, 1, 4)
	cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(tr.addCh)}

	for {
		chosen, recv, ok := reflect.Select(cases)
		switch chosen {
		case 0:
			if !ok {
				// The trigger has been stopped.
				return
			}
			// Add a new callback.
			req := recv.Interface().(*addRequest)
			callbacks = append(callbacks, req.callback)
			cases = append(cases, req.sc)
		default:
			go callbacks[chosen]()
			n := len(callbacks) - 1
			callbacks[chosen], callbacks[n] = callbacks[n], nil
			callbacks = callbacks[:n]
			cases[chosen], cases[n] = cases[n], reflect.SelectCase{}
			cases = cases[:n]
		}
	}
}

func NewTrigger() *Trigger {
	tr := &Trigger{addCh: make(chan *addRequest)}
	go tr.loop()
	return tr
}
