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
	read chan []byte
	m    map[string]chan []byte
}

func (k *krsshChar) written(req ble.Request, rsp ble.ResponseWriter) {
	k.Lock()
	data := req.Data()
	log.Println("got data:", data)
	k.Unlock()
	k.read <- data
}
func (k *krsshChar) Write(msg []byte) {
	k.Lock()
	for _, ch := range k.m {
		ch <- msg
	}
	k.Unlock()
}

func (k *krsshChar) notify(req ble.Request, n ble.Notifier) {
	ch := make(chan []byte)
	k.Lock()
	k.m[req.Conn().RemoteAddr().String()] = ch
	k.Unlock()
	log.Printf("bluetooth: Notification subscribed on Conn %s", req.Conn().RemoteAddr().String())
	defer func() {
		k.Lock()
		delete(k.m, req.Conn().RemoteAddr().String())
		k.Unlock()
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

func makeKRSSHChar() (c *ble.Characteristic, k *krsshChar) {
	k = &krsshChar{m: make(map[string]chan []byte)}
	c = ble.NewCharacteristic(krsshCharUUID)
	c.HandleWrite(ble.WriteHandlerFunc(k.written))
	c.HandleNotify(ble.NotifyHandlerFunc(k.notify))
	c.HandleIndicate(ble.NotifyHandlerFunc(k.notify))
	return
}

type BluetoothPeripheral struct {
	sync.Mutex
	Read      chan []byte
	Write     chan []byte
	uuid      ble.UUID
	krsshChar *krsshChar
	service   *ble.Service
}

func NewBluetoothPeripheral(uuidStr string) (bp *BluetoothPeripheral, err error) {
	uuid, err := ble.Parse(uuidStr)
	if err != nil {
		return
	}
	service := ble.NewService(uuid)
	char, krsshChar := makeKRSSHChar()
	service.AddCharacteristic(char)

	bp = &BluetoothPeripheral{
		uuid:      uuid,
		Read:      make(chan []byte, 1024),
		Write:     make(chan []byte, 1024),
		krsshChar: krsshChar,
		service:   service,
	}

	krsshChar.read = bp.Read
	return
}

func (bp *BluetoothPeripheral) bluetoothMain() {
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
			bp.krsshChar.Write(msg)
		}
	}
}
