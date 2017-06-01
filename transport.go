package kr

import (
	"encoding/base64"
	"strings"
)

type Transport interface {
	Setup(ps *PairingSecret) (err error)
	PushAlert(ps *PairingSecret, alertText string, message []byte) (err error)
	SendMessage(ps *PairingSecret, message []byte) (err error)
	Read(notifier *Notifier, ps *PairingSecret) (ciphertexts [][]byte, err error)
}

type AWSTransport struct{}

func (t AWSTransport) Setup(ps *PairingSecret) (err error) {
	_, err = CreateQueue(ps.SQSSendQueueName())
	if err != nil {
		return
	}
	_, err = CreateQueue(ps.SQSRecvQueueName())
	if err != nil {
		return
	}
	return
}

func (t AWSTransport) PushAlert(ps *PairingSecret, alertText string, message []byte) (err error) {
	ctxt, err := ps.EncryptMessage(message)
	if err != nil {
		return
	}

	ctxtString := base64.StdEncoding.EncodeToString(ctxt)
	go func() {
		arn := ps.GetSNSEndpointARN()
		if arn != nil {
			if pushErr := PushAlertToSNSEndpoint(alertText, ctxtString, *arn, ps.SQSSendQueueName()); pushErr != nil {
				log.Error("Push error:", pushErr)
			}
		}
	}()
	err = SendToQueue(ps.SQSSendQueueName(), ctxtString)
	if err != nil {
		return
	}
	return
}
func (t AWSTransport) SendMessage(ps *PairingSecret, message []byte) (err error) {
	ctxt, err := ps.EncryptMessage(message)
	if err != nil {
		return
	}
	ctxtString := base64.StdEncoding.EncodeToString(ctxt)

	go func() {
		ps.Lock()
		arn := ps.snsEndpointARN
		ps.Unlock()
		if arn != nil {
			if pushErr := PushToSNSEndpoint(ctxtString, *arn, ps.SQSSendQueueName()); pushErr != nil {
				log.Error("Push error:", pushErr)
			}
		}
	}()

	err = SendToQueue(ps.SQSSendQueueName(), ctxtString)
	if err != nil {
		return
	}
	return
}

func notifyIfSignatureExpiredErr(err error, notifier *Notifier) {
	if err == nil || notifier == nil {
		return
	}
	if strings.Contains(err.Error(), "Signature expired") {
		notifier.Notify([]byte(Red("Kryptonite ▶ Your system time is out of sync! Kryptonite will not work until you have synchronized your system time. Please run ") + Yellow(NTP_UPDATE_CMD) + Red(" and try again.") + "\r\n"))
	}
}

func (t AWSTransport) Read(notifier *Notifier, ps *PairingSecret) (ciphertexts [][]byte, err error) {
	ctxtStrings, err := ReceiveAndDeleteFromQueue(ps.SQSRecvQueueName())
	notifyIfSignatureExpiredErr(err, notifier)
	if err != nil {
		return
	}

	for _, ctxtString := range ctxtStrings {
		ctxt, err := base64.StdEncoding.DecodeString(ctxtString)
		if err != nil {
			log.Error("base64 ciphertext decoding error")
			continue
		}
		ciphertexts = append(ciphertexts, ctxt)
	}
	return
}
