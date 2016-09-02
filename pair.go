package krssh

const SQS_BASE_QUEUE_URL = "https://sqs.us-east-1.amazonaws.com/911777333295/"

type PairingSecret struct {
	SQSBaseQueueName   string `json:"q"`
	SymmetricSecretKey []byte `json:"k"`
}

func (ps PairingSecret) SQSSendQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSBaseQueueName
}
func (ps PairingSecret) SQSRecvQueueURL() string {
	return SQS_BASE_QUEUE_URL + ps.SQSBaseQueueName + "-recv"
}

func GeneratePairingSecret() (ps PairingSecret, err error) {
	symmetricSecretKey, err := GenSymmetricSecretKey()
	if err != nil {
		return
	}
	ps.SymmetricSecretKey = symmetricSecretKey.Bytes

	ps.SQSBaseQueueName, err = Rand128Base62()
	if err != nil {
		return
	}

	_, err = CreateSendAndReceiveQueues(ps.SQSBaseQueueName)
	if err != nil {
		return
	}

	return
}
