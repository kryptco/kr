package kr

import (
	"bufio"
	"os"
	"sync"
	"time"
)

const NOTIFY_LOG_FILE_NAME = "krd-notify.log"

type Notifier struct {
	*os.File
	*sync.Mutex
}

func OpenNotifier() (n Notifier, err error) {
	filePath, err := KrDirFile(NOTIFY_LOG_FILE_NAME)
	if err != nil {
		return
	}
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	n = Notifier{file, &sync.Mutex{}}
	return
}

func (n Notifier) Notify(body []byte) (err error) {
	n.Lock()
	defer n.Unlock()

	_, err = n.Write(body)
	if err != nil {
		return
	}
	err = n.Sync()
	//	FIXME: workaround to ensure success logged to stderr before signature returned to SSH
	<-time.After(50 * time.Millisecond)
	return
}

type NotificationReader struct {
	*os.File
	lineReader *bufio.Reader
}

func OpenNotificationReader() (r NotificationReader, err error) {
	filePath, err := KrDirFile(NOTIFY_LOG_FILE_NAME)
	if err != nil {
		return
	}
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_APPEND|os.O_RDONLY, 0666)
	if err != nil {
		return
	}
	r = NotificationReader{
		File:       file,
		lineReader: bufio.NewReader(file),
	}
	return
}

func (r NotificationReader) Read() (body []byte, err error) {
	return r.lineReader.ReadBytes('\n')
}
