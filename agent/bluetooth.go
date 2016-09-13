package main

import (
	//"encoding/base64"
	"log"
	"sync"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib/gatt"
)

type krsshChar struct {
	sync.Mutex
	m map[string]chan []byte
}

func (k *krsshChar) written(req ble.Request, rsp ble.ResponseWriter) {
	k.Lock()
	log.Println("got data:", string(req.Data()))
	k.Unlock()
	go func() {
		<-time.After(90 * time.Second)
		k.Lock()
		k.m[req.Conn().RemoteAddr().String()] <- []byte("hi")
		k.Unlock()
	}()
}

func (k *krsshChar) notify(req ble.Request, n ble.Notifier) {
	ch := make(chan []byte)
	k.Lock()
	k.m[req.Conn().RemoteAddr().String()] = ch
	k.Unlock()
	log.Printf("krssh: Notification subscribed")
	defer func() {
		k.Lock()
		delete(k.m, req.Conn().RemoteAddr().String())
		k.Unlock()
	}()
	for {
		select {
		case <-n.Context().Done():
			log.Printf("bluetooth: Notification unsubscribed")
			return
		//case <-time.After(time.Second * 20):
		//log.Printf("bluetooth: timeout")
		//return
		case msg := <-ch:
			if _, err := n.Write(msg); err != nil {
				log.Printf("bluetooth: can't indicate: %s", err)
				return
			}
		}
	}
}

func makeKRSSHChar() *ble.Characteristic {
	krsshCharUUID := ble.MustParse("20F53E48-C08D-423A-B2C2-1C797889AF24")
	k := &krsshChar{m: make(map[string]chan []byte)}
	c := ble.NewCharacteristic(krsshCharUUID)
	c.HandleWrite(ble.WriteHandlerFunc(k.written))
	c.HandleNotify(ble.NotifyHandlerFunc(k.notify))
	c.HandleIndicate(ble.NotifyHandlerFunc(k.notify))
	return c
}

func startBluetooth() {
	testSvc := ble.NewService(ble.MustParse("7094E2FD-2642-4716-BB20-4D012DD36030"))
	testSvc.AddCharacteristic(makeKRSSHChar())
	if err := gatt.AddService(testSvc); err != nil {
		log.Fatalf("can't add service: %s", err)
	}

	if err := gatt.AdvertiseNameAndServices("Gopher", testSvc.UUID); err != nil {
		log.Fatalf("can't advertise: %s", err)
	}

	select {}
}
