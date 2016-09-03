package krssh

import (
	"errors"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var ErrNoMessages = errors.New("No messages in SQS Queue")

func TestSQS() (err error) {
	creds := credentials.NewStaticCredentials("AKIAJMZJ3X6MHMXRF7QQ", "0hincCnlm2XvpdpSD+LBs6NSwfF0250pEnEyYJ49", "")
	_, err = creds.Get()
	if err != nil {
		log.Fatal(err)
	}

	cfg := aws.NewConfig().WithRegion("us-east-1").WithCredentials(creds)
	session, err := session.NewSession(cfg)
	if err != nil {
		log.Fatal(err)
	}

	sqsService := sqs.New(session)

	randName, err := Rand256Base62()
	if err != nil {
		log.Fatal(err)
	}

	createQueueInput := &sqs.CreateQueueInput{
		QueueName: aws.String(randName), // Required
		Attributes: map[string]*string{
			sqs.QueueAttributeNameMessageRetentionPeriod: aws.String("600"),
			sqs.QueueAttributeNameVisibilityTimeout:      aws.String("0"),
		},
	}

	response, err := sqsService.CreateQueue(createQueueInput)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("created queue:", response)

	sendMessageInput := &sqs.SendMessageInput{
		MessageBody: aws.String("test"),
		QueueUrl:    response.QueueUrl,
	}

	sendResponse, err := sqsService.SendMessage(sendMessageInput)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("sent message:", sendResponse)

	receiveMessageInput := &sqs.ReceiveMessageInput{
		MaxNumberOfMessages: aws.Int64(1),
		QueueUrl:            response.QueueUrl,
		WaitTimeSeconds:     aws.Int64(10),
	}

	receiveResponse, err := sqsService.ReceiveMessage(receiveMessageInput)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("received message:", receiveResponse)

	return
}

func getSQSService() (sqsService *sqs.SQS, err error) {
	creds := credentials.NewStaticCredentials("AKIAJMZJ3X6MHMXRF7QQ", "0hincCnlm2XvpdpSD+LBs6NSwfF0250pEnEyYJ49", "")
	_, err = creds.Get()
	if err != nil {
		return
	}

	cfg := aws.NewConfig().WithRegion("us-east-1").WithCredentials(creds)
	session, err := session.NewSession(cfg)
	if err != nil {
		return
	}

	sqsService = sqs.New(session)
	return
}

func ReceiveAndDeleteFromQueue(queueUrl string) (messages []string, err error) {
	sqsService, err := getSQSService()
	if err != nil {
		log.Println(err)
		return
	}

	receiveMessageInput := &sqs.ReceiveMessageInput{
		MaxNumberOfMessages: aws.Int64(10),
		QueueUrl:            aws.String(queueUrl),
		WaitTimeSeconds:     aws.Int64(10),
	}

	receiveResponse, err := sqsService.ReceiveMessage(receiveMessageInput)
	if err != nil {
		log.Println(err)
		return
	}

	deleteRequestEntries := []*sqs.DeleteMessageBatchRequestEntry{}
	for i, message := range receiveResponse.Messages {
		messages = append(messages, *message.Body)
		deleteRequestEntries = append(deleteRequestEntries, &sqs.DeleteMessageBatchRequestEntry{
			Id:            aws.String(strconv.Itoa(i)),
			ReceiptHandle: message.ReceiptHandle,
		})
	}
	if len(messages) > 0 {
		deleteMessageInput := &sqs.DeleteMessageBatchInput{
			QueueUrl: aws.String(queueUrl),
			Entries:  deleteRequestEntries,
		}

		_, err = sqsService.DeleteMessageBatch(deleteMessageInput)
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		err = ErrNoMessages
	}

	return
}

func SendToQueue(queueUrl string, message string) (err error) {
	sqsService, err := getSQSService()
	if err != nil {
		log.Println(err)
		return
	}

	sendMessageInput := &sqs.SendMessageInput{
		MessageBody: aws.String(message),
		QueueUrl:    aws.String(queueUrl),
	}

	_, err = sqsService.SendMessage(sendMessageInput)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// Create queues named `queueBaseName` and `queueBaseName-recv`
// Return URL for queue named `queueBaseName`
func CreateSendAndReceiveQueues(queueBaseName string) (baseQueueURL string, err error) {
	sqsService, err := getSQSService()
	if err != nil {
		log.Println(err)
		return
	}
	createSendQueueInput := &sqs.CreateQueueInput{
		QueueName: aws.String(queueBaseName), // Required
		Attributes: map[string]*string{
			sqs.QueueAttributeNameMessageRetentionPeriod: aws.String("600"),
			sqs.QueueAttributeNameVisibilityTimeout:      aws.String("0"),
		},
	}
	createSendQueueResponse, err := sqsService.CreateQueue(createSendQueueInput)
	if err != nil {
		return
	}

	createRecvQueueInput := &sqs.CreateQueueInput{
		QueueName: aws.String(queueBaseName + "-recv"), // Required
		Attributes: map[string]*string{
			sqs.QueueAttributeNameMessageRetentionPeriod: aws.String("600"),
			sqs.QueueAttributeNameVisibilityTimeout:      aws.String("0"),
		},
	}
	_, err = sqsService.CreateQueue(createRecvQueueInput)
	if err != nil {
		return
	}

	baseQueueURL = *createSendQueueResponse.QueueUrl
	return
}
