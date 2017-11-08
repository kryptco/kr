package kr

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const NOTIFY_LOG_FILE_NAME = "krd-notify.log"

type Notifier struct {
	*os.File
	*sync.Mutex
}

func OpenNotifier(id string) (n Notifier, err error) {
	filePath, err := NotifyDirFile(NOTIFY_LOG_FILE_NAME + "-" + id)
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

func OpenNotificationReader(id string) (r NotificationReader, err error) {
	filePath, err := NotifyDirFile(NOTIFY_LOG_FILE_NAME + "-" + id)
	if err != nil {
		return
	}
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_APPEND|os.O_RDONLY, 0666)
	if err != nil {
		return
	}
	//	some systems don't truncate correctly
	//	2 = io.Seekend, but not added until Go 1.7
	file.Seek(0, 2)
	r = NotificationReader{
		File:       file,
		lineReader: bufio.NewReader(file),
	}
	return
}

func (r NotificationReader) Read() (body []byte, err error) {
	return r.lineReader.ReadBytes('\n')
}

func StartNotifyCleanup() {
	go func() {
		for {
			notifyDir, err := NotifyDir()
			if err == nil {
				notifyDirFile, err := os.Open(notifyDir)
				if err == nil {
					names, err := notifyDirFile.Readdirnames(0)
					if err == nil {
						for _, name := range names {
							if strings.HasSuffix(name, "]") {
								logFilePath := filepath.Join(notifyDir, name)
								logFile, err := os.Open(logFilePath)
								if err == nil {
									info, err := logFile.Stat()
									if err == nil {
										if time.Now().Sub(info.ModTime()) >= time.Hour {
											_ = os.Remove(logFilePath)
										}
									}
								}
							}
						}
					}
				}
			}
			<-time.After(1 * time.Hour)
		}
	}()
}

func StartControlServerLogger(prefix string) (r NotificationReader, err error) {
	r, err = OpenNotificationReader(prefix)
	if err != nil {
		return
	}
	go func() {
		if prefix != "" {
			defer os.Remove(r.Name())
		}

		printedNotifications := map[string]bool{}
		for {
			notification, err := r.Read()
			switch err {
			case nil:
				notificationStr := string(notification)
				if _, ok := printedNotifications[notificationStr]; ok {
					continue
				}
				os.Stderr.WriteString(notificationStr)
				printedNotifications[notificationStr] = true
			case io.EOF:
				<-time.After(50 * time.Millisecond)
			default:
				return
			}
		}
	}()
	return
}
