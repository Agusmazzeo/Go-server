package aws_handler

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type SecretManager struct {
	svc *secretsmanager.SecretsManager
}

func NewSecretManager(svc *secretsmanager.SecretsManager) *SecretManager {
	return &SecretManager{svc: svc}
}

func (s *SecretManager) GetSecretValue(secretId string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretId),
	}

	result, err := s.svc.GetSecretValue(input)
	if err != nil {
		return "", err
	}

	return *result.SecretString, nil
}

func (s *SecretManager) CreateSecret(name, value string) error {
	input := &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(value),
	}

	_, err := s.svc.CreateSecret(input)

	return err
}
