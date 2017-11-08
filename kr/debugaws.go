package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kryptco/kr"
	"github.com/urfave/cli"
)

func timeFunc(name string, f func()) {
	start := time.Now()
	f()
	end := time.Now()
	os.Stderr.WriteString(fmt.Sprintf("%s took %dms\r\n", name, end.Sub(start)/time.Millisecond))
}

func debugAWSCommand(c *cli.Context) (_ error) {
	go func() {
		kr.Analytics{}.PostEventUsingPersistedTrackingID("kr", "debugaws", nil, nil)
	}()
	queueName, err := kr.Rand256Base62()
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}

	timeFunc("Create SQS Queue", func() {
		_, err = kr.CreateQueue(queueName)
		if err != nil {
			PrintFatal(os.Stderr, err.Error())
		}
	})

	message, err := kr.RandNBase64(2048)
	if err != nil {
		PrintFatal(os.Stderr, err.Error())
	}
	timeFunc("Send SQS Message", func() {
		err = kr.SendToQueue(queueName, message)
		if err != nil {
			PrintFatal(os.Stderr, err.Error())
		}
	})

	timeFunc("Receive SQS Message", func() {
		go func() {
			<-time.After(10 * time.Second)
			PrintFatal(os.Stderr, "Took longer than 10s to receive SQS message")
		}()
		for {
			messages, err := kr.ReceiveAndDeleteFromQueue(queueName)
			if err != nil {
				PrintFatal(os.Stderr, err.Error())
			}
			for _, receivedMessage := range messages {
				if receivedMessage == message {
					return
				}
			}
		}
	})

	PrintErr(os.Stderr, "AWS connectivity check succeeded.")

	return
}
