package aws_handler

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type AWSHandler struct {
	SecretManager *SecretManager
	// Add other AWS service clients as needed
}

func NewAWSHandler(region string) (*AWSHandler, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	if err != nil {
		return nil, err
	}

	svc := secretsmanager.New(sess)
	secretManager := NewSecretManager(svc)

	return &AWSHandler{
		SecretManager: secretManager,
		// Initialize other AWS service clients here
	}, nil
}
