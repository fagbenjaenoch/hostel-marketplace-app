package secrets

import (
	"context"

	"github.com/fagbenjaenoch/dorms-ng/internal/config"
	infisical "github.com/infisical/go-sdk"
)

var client infisical.InfisicalClientInterface

func SetupSecretsManager(config *config.Config) error {
	c := infisical.NewInfisicalClient(context.Background(), infisical.Config{})

	_, err := c.Auth().UniversalAuthLogin(config.Infisical.ClientID, config.Infisical.ClientSecret)
	if err != nil {
		return err
	}

	client = c

	return nil
}

func GetSecretClient() infisical.InfisicalClientInterface {
	if client == nil {
		return nil
	}
	return client
}
