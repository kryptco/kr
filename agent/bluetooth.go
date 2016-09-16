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
	bp.lastRead = time.Now()
	bp.Unlock()
	bp.Read <- data
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SplitMsgForBluetooth(message []byte) (splitMessage [][]byte) {
	log.Println("message len", len(message))
	block := 128
	n := byte(len(message) / block)
	for offset := 0; offset < len(message); offset += block {
		endOffset := min(offset+block, len(message))
		blockMsg := append([]byte{n}, message[offset:endOffset]...)
		splitMessage = append(splitMessage, blockMsg)
		n = n - 1
	}
	log.Println("split messages: ", len(splitMessage))
	return
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
			log.Printf("Writing %d byte message\n", len(msg))
			if _, err := n.Write(msg); err != nil {
				log.Printf("bluetooth: can't indicate: %s", err)
				return
			}
		case <-bp.Close:
			log.Println("Closing peripheral")
			err := n.Close()
			if err != nil {
				log.Println("Error closing  peripheral:", err)
			}
			//gatt.StopAdvertising()
			gatt.Stop()
		}
	}
}

var krsshCharUUID = ble.MustParse("20F53E48-C08D-423A-B2C2-1C797889AF24")

type BluetoothManager struct {
	sync.Mutex
	peripheral *BluetoothPeripheral
	writeQueue [][]byte
}

func (bm *BluetoothManager) SetPeripheral(bp *BluetoothPeripheral) {
	bm.Lock()
	defer bm.Unlock()
	bm.peripheral = bp
	bm.advertise()
	if len(bm.writeQueue) > 0 {
		for _, msg := range bm.writeQueue {
			bm.peripheral.Write <- msg
		}
		bm.writeQueue = [][]byte{}
	}
}

func (bm *BluetoothManager) advertise() {
	defer func() {
		//if r := recover(); r != nil {
		//log.Println("recovered: ", r)
		//}
	}()
	gatt.StopAdvertising()
	err := gatt.SetServices([]*ble.Service{bm.peripheral.service})
	if err != nil {
		log.Println("error setting gatt services:", err)
	}
	gatt.AdvertiseNameAndServices("krssh", bm.peripheral.uuid)
	if err != nil {
		log.Println("error advertising gatt services:", err)
		return
	}
	log.Println("Bluetooth advertising")
}

func (bm *BluetoothManager) Write(msg []byte) {
	bm.Lock()
	defer bm.Unlock()
	if bm.peripheral == nil {
		bm.writeQueue = append(bm.writeQueue, msg)
	} else {
		bm.peripheral.Write <- msg
	}
}

type BluetoothPeripheral struct {
	sync.Mutex
	Read       chan []byte
	Write      chan []byte
	Close      chan bool
	uuid       ble.UUID
	service    *ble.Service
	m          map[string]chan []byte
	lastRead   time.Time
	lastWrite  time.Time
	writeQueue [][]byte
}

func NewBluetoothPeripheral(uuidStr string) (bp *BluetoothPeripheral, err error) {
	uuid, err := ble.Parse(uuidStr)
	if err != nil {
		return
	}
	bp = &BluetoothPeripheral{
		uuid:  uuid,
		Read:  make(chan []byte, 1024),
		Close: make(chan bool),
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
}

func (bp *BluetoothPeripheral) start() {
	for {
		select {
		case msg := <-bp.Write:
			msgs := SplitMsgForBluetooth(msg)
			bp.Lock()
			for _, msg := range msgs {
				if len(bp.m) == 0 {
					bp.writeQueue = append(bp.writeQueue, msg)
					log.Printf("wrote queued messages\n")
				} else {
					for _, ch := range bp.m {
						ch <- msg
					}
					log.Printf("wrote msg to %d devices\n", len(bp.m))
				}
			}
			bp.lastWrite = time.Now()
			bp.Unlock()
		}
	}
}
