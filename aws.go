package kr

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const SQS_BASE_QUEUE_URL = "https://sqs.us-east-1.amazonaws.com/911777333295/"

func queueNameToURL(name string) string {
	return SQS_BASE_QUEUE_URL + name
}

var AWS_ENV_VARS_TO_UNSET = []string{
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_DEFAULT_REGION",
	"AWS_DEFAULT_PROFILE",
	"AWS_ACCESS_KEY",
	"AWS_SECRET_KEY",
	"AWS_SDK_LOAD_CONFIG",
}

var AWS_ENV_VARS_TO_DEVNULL = []string{
	"AWS_CONFIG_FILE",
	"AWS_SHARED_CREDENTIALS_FILE",
}

func unsetAWSEnvVars() {
	//	aws-sdk-go by default looks at ENV vars for shared configuration files. Unset them to prevent this:
	//	https://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html
	for _, env := range AWS_ENV_VARS_TO_UNSET {
		os.Unsetenv(env)
	}
	for _, env := range AWS_ENV_VARS_TO_DEVNULL {
		os.Setenv(env, "/dev/null")
	}
}

var unsetAWSEnvVarsOnce sync.Once

func getAWSSession() (conf client.ConfigProvider, err error) {
	unsetAWSEnvVarsOnce.Do(unsetAWSEnvVars)

	//	Public restricted credentials for SQS/SNS
	creds := credentials.NewStaticCredentials("AKIAJMZJ3X6MHMXRF7QQ", "0hincCnlm2XvpdpSD+LBs6NSwfF0250pEnEyYJ49", "")
	_, err = creds.Get()
	if err != nil {
		return
	}

	cfg := aws.NewConfig().WithRegion("us-east-1").WithCredentials(creds)
	conf, err = session.NewSession(cfg)
	if err != nil {
		return
	}
	return
}

func getSQSService() (sqsService *sqs.SQS, err error) {
	session, err := getAWSSession()
	if err != nil {
		return
	}
	sqsService = sqs.New(session)
	return
}

func getSNSService() (snsService *sns.SNS, err error) {
	session, err := getAWSSession()
	if err != nil {
		return
	}
	snsService = sns.New(session)
	return
}

func PushAlertToSNSEndpoint(alertText, requestCiphertext, endpointARN, sqsQueueName string) (err error) {
	apnsPayload, _ := json.Marshal(
		map[string]interface{}{
			"aps": map[string]interface{}{
				"alert":             alertText,
				"sound":             "",
				"content-available": 1,
				"mutable-content":   1,
				"queue":             sqsQueueName,
				"c":                 requestCiphertext,
				"session_uuid":      sqsQueueName,
				"category":          "authorize_identifier",
			},
		})
	gcmPayload, _ := json.Marshal(
		map[string]interface{}{
			"data": map[string]interface{}{
				"message": requestCiphertext,
				"queue":   sqsQueueName,
			},
		})
	err = pushToSNS(endpointARN, apnsPayload, gcmPayload)
	return
}

func PushToSNSEndpoint(requestCiphertext, endpointARN, sqsQueueName string) (err error) {

	apnsPayload, _ := json.Marshal(
		map[string]interface{}{
			"aps": map[string]interface{}{
				"alert":             "",
				"sound":             "",
				"content-available": 1,
				"queue":             sqsQueueName,
				"c":                 requestCiphertext,
			},
		})
	gcmPayload, _ := json.Marshal(
		map[string]interface{}{
			"data": map[string]interface{}{
				"message": requestCiphertext,
				"queue":   sqsQueueName,
			},
		})
	err = pushToSNS(endpointARN, apnsPayload, gcmPayload)
	return
}

func pushToSNS(endpointARN string, apnsPayload []byte, gcmPayload []byte) (err error) {
	snsService, err := getSNSService()
	if err != nil {
		return
	}
	message := map[string]interface{}{
		"APNS":         string(apnsPayload),
		"APNS_SANDBOX": string(apnsPayload),
		"GCM":          string(gcmPayload),
	}
	messageJson, err := json.Marshal(message)
	if err != nil {
		return
	}
	publishInput := &sns.PublishInput{
		Message:          aws.String(string(messageJson)),
		MessageStructure: aws.String("json"),
		TargetArn:        aws.String(endpointARN),
	}
	_, err = snsService.Publish(publishInput)
	if err != nil {
		if strings.Contains(err.Error(), "EndpointDisabled") {
			enableErr := enableSNSEndpoint(endpointARN)
			if enableErr != nil {
				log.Error("EnableSNSEndpoint error:", enableErr)
				return
			}
			//	try again
			_, err = snsService.Publish(publishInput)
		}
		return
	}
	return
}

func enableSNSEndpoint(arn string) (err error) {
	snsService, err := getSNSService()
	if err != nil {
		return
	}
	input := &sns.SetEndpointAttributesInput{
		Attributes: map[string]*string{
			"Enabled": aws.String("true"),
		},
		EndpointArn: aws.String(arn),
	}
	_, err = snsService.SetEndpointAttributes(input)
	return
}

func ReceiveAndDeleteFromQueue(queueName string) (messages []string, err error) {
	sqsService, err := getSQSService()
	if err != nil {
		log.Error(err)
		return
	}

	receiveMessageInput := &sqs.ReceiveMessageInput{
		MaxNumberOfMessages: aws.Int64(10),
		QueueUrl:            aws.String(queueNameToURL(queueName)),
		WaitTimeSeconds:     aws.Int64(3),
	}

	receiveResponse, err := sqsService.ReceiveMessage(receiveMessageInput)
	if err != nil {
		if strings.Contains(err.Error(), "AWS.SimpleQueueService.NonExistentQueue") {
			log.Warning("SQS queue " + queueName + " does not exist, creating")
			_, err = CreateQueue(queueName)
			if err != nil {
				return
			}
		}
		_, err = sqsService.ReceiveMessage(receiveMessageInput)
		if err != nil {
			log.Error(err)
			return
		}
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
			QueueUrl: aws.String(queueNameToURL(queueName)),
			Entries:  deleteRequestEntries,
		}

		_, err = sqsService.DeleteMessageBatch(deleteMessageInput)
		if err != nil {
			log.Error(err)
			return
		}
	}

	return
}

func SendToQueue(queueName string, message string) (err error) {
	sqsService, err := getSQSService()
	if err != nil {
		log.Error(err)
		return
	}

	sendMessageInput := &sqs.SendMessageInput{
		MessageBody: aws.String(message),
		QueueUrl:    aws.String(queueNameToURL(queueName)),
	}

	_, err = sqsService.SendMessage(sendMessageInput)
	if err != nil {
		if strings.Contains(err.Error(), "AWS.SimpleQueueService.NonExistentQueue") {
			log.Warning("SQS queue " + queueName + " does not exist, creating")
			_, err = CreateQueue(queueName)
			if err != nil {
				return
			}
		}
		_, err = sqsService.SendMessage(sendMessageInput)
		if err != nil {
			log.Error(err)
			return
		}
	}
	return
}

// Return URL for queue named `queueName`
func CreateQueue(queue string) (queueURL string, err error) {
	sqsService, err := getSQSService()
	if err != nil {
		log.Error(err)
		return
	}
	createQueueInput := &sqs.CreateQueueInput{
		QueueName: aws.String(queue), // Required
		Attributes: map[string]*string{
			//	longer to store Unpair messages
			sqs.QueueAttributeNameMessageRetentionPeriod: aws.String("172800"),
			sqs.QueueAttributeNameVisibilityTimeout:      aws.String("1"),
		},
	}
	createQueueResponse, err := sqsService.CreateQueue(createQueueInput)
	if err != nil {
		return
	}

	queueURL = *createQueueResponse.QueueUrl
	return
}
