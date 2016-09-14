package main

import (
	//"encoding/base64"
	"log"
	"sync"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib/gatt"
)

func (bp *BluetoothPeripheral) written(req ble.Request, rsp ble.ResponseWriter) {
	bp.Lock()
	data := req.Data()
	log.Println("got data:", data)
	bp.Unlock()
	bp.Read <- data
}

func (bp *BluetoothPeripheral) notify(req ble.Request, n ble.Notifier) {
	ch := make(chan []byte)
	bp.Lock()
	bp.m[req.Conn().RemoteAddr().String()] = ch
	log.Printf("writing queued messages\n")
	for _, msg := range bp.writeQueue {
		n.Write(msg)
	}
	bp.writeQueue = [][]byte{}
	log.Printf("wrote queued messages\n")
	bp.Unlock()
	log.Printf("bluetooth: Notification subscribed on Conn %s", req.Conn().RemoteAddr().String())
	defer func() {
		bp.Lock()
		delete(bp.m, req.Conn().RemoteAddr().String())
		bp.Unlock()
	}()
	for {
		select {
		case <-n.Context().Done():
			log.Printf("bluetooth: Notification unsubscribed on Conn %s", req.Conn().RemoteAddr().String())
			return
		case msg := <-ch:
			if _, err := n.Write(msg); err != nil {
				log.Printf("bluetooth: can't indicate: %s", err)
				return
			}
		}
	}
}

var krsshCharUUID = ble.MustParse("20F53E48-C08D-423A-B2C2-1C797889AF24")

type BluetoothPeripheral struct {
	sync.Mutex
	Read       chan []byte
	Write      chan []byte
	writeQueue [][]byte
	uuid       ble.UUID
	service    *ble.Service
	m          map[string]chan []byte
}

func NewBluetoothPeripheral(uuidStr string) (bp *BluetoothPeripheral, err error) {
	uuid, err := ble.Parse(uuidStr)
	if err != nil {
		return
	}
	bp = &BluetoothPeripheral{
		uuid:  uuid,
		Read:  make(chan []byte, 1024),
		m:     map[string]chan []byte{},
		Write: make(chan []byte, 1024),
	}

	service := ble.NewService(uuid)
	char := ble.NewCharacteristic(krsshCharUUID)
	char.HandleWrite(ble.WriteHandlerFunc(bp.written))
	char.HandleNotify(ble.NotifyHandlerFunc(bp.notify))
	char.HandleIndicate(ble.NotifyHandlerFunc(bp.notify))
	service.AddCharacteristic(char)

	bp.service = service

	return
}

func (bp *BluetoothPeripheral) bluetoothMain() {
	go bp.start()
	panicked := false
	for !panicked {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered startBluetooth", r)
					panicked = true
				}
			}()
			gatt.Reset()
			if err := gatt.AddService(bp.service); err != nil {
				log.Printf("can't add service: %s", err)
				gatt.RemoveAllServices()
				<-time.After(10 * time.Second)
				return
			}
			if err := gatt.AdvertiseNameAndServices("Gopher", bp.service.UUID); err != nil {
				log.Printf("can't advertise: %s", err)
				gatt.RemoveAllServices()
				<-time.After(10 * time.Second)
				return
			}
			log.Printf("Bluetooth advertising")
			select {}
		}()
	}
}

func (bp *BluetoothPeripheral) start() {
	for {
		select {
		case msg := <-bp.Write:
			bp.Lock()
			if len(bp.m) == 0 {
				bp.writeQueue = append(bp.writeQueue, msg)
				log.Printf("wrote queued messages\n")
			} else {
				for _, ch := range bp.m {
					ch <- msg
				}
				log.Printf("wrote msg to %d devices\n", len(bp.m))
			}
			bp.Unlock()
		}
	}
}
