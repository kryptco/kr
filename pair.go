package main

type PairingSecret struct {
	SQSQueueName string `json:"sqs_queue_name"`
	SymmetricKey []byte `json:"symmetric_key"`
}
