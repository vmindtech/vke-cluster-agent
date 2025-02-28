package service

import (
	"crypto/tls"
	"net/http"
	"time"

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
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		TLSHandshakeTimeout: 30 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	}

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint:            identityURL,
		ApplicationCredentialID:     applicationCredentialID,
		ApplicationCredentialSecret: applicationCredentialSecret,
		AllowReauth:                 true,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 60,
	}

	providerClient, err := openstack.NewClient(identityURL)
	if err != nil {
		klog.Errorf("Failed to create OpenStack client - identityURL: %s, error: %v", identityURL, err)
		return nil, err
	}
	providerClient.HTTPClient = *client

	err = openstack.Authenticate(providerClient, authOpts)
	if err != nil {
		klog.Errorf("OpenStack authentication failed - identityURL: %s, projectID: %s, applicationCredentialID: %s, error: %v",
			identityURL, pjID, applicationCredentialID, err)
		return nil, err
	}

	return providerClient, nil
}
