package service

import (
	"crypto/tls"
	"net/http"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"k8s.io/klog/v2"
)

type IOpenstackService interface {
	ValidateAndCreateSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL string) (*gophercloud.ProviderClient, error)
}

type openstackService struct {
}

func NewOpenstackService() IOpenstackService {
	return &openstackService{}
}

func (o *openstackService) ValidateAndCreateSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL string) (*gophercloud.ProviderClient, error) {
	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint:            identityURL,
		ApplicationCredentialID:     applicationCredentialID,
		ApplicationCredentialSecret: applicationCredentialSecret,
		AllowReauth:                 true,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		klog.Errorf("OpenStack authentication failed - identityURL: %s, projectID: %s, applicationCredentialID: %s, error: %v",
			identityURL, pjID, applicationCredentialID, err)
		return nil, err
	}

	provider.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return provider, nil
}
