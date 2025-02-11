package service

import (
	"crypto/tls"
	"net/http"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/sirupsen/logrus"
)

type IOpenstackService interface {
	ValidateAndCreateSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL string) (*gophercloud.ProviderClient, error)
}

type openstackService struct {
	logger *logrus.Logger
}

func NewOpenstackService(l *logrus.Logger) IOpenstackService {
	return &openstackService{
		logger: l,
	}
}

func (o *openstackService) ValidateAndCreateSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL string) (*gophercloud.ProviderClient, error) {
	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint:            identityURL,
		ApplicationCredentialID:     applicationCredentialID,
		ApplicationCredentialSecret: applicationCredentialSecret,
		TenantID:                    pjID,
		AllowReauth:                 true,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		o.logger.WithFields(logrus.Fields{
			"identityURL":             identityURL,
			"projectID":               pjID,
			"applicationCredentialID": applicationCredentialID,
			"error":                   err.Error(),
		}).Error("OpenStack authentication failed")
		return nil, err
	}

	provider.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return provider, nil
}
