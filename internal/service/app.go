package service

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/sirupsen/logrus"
	"github.com/vmindtech/vke-cluster-agent/config"
	"github.com/vmindtech/vke-cluster-agent/pkg/constants"
)

type IAppService interface {
	GetOpenstackSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL string) (*gophercloud.ProviderClient, error)
	CheckVKEClusterCertificateExpiration(isExpired chan bool)
}

type appService struct {
	logger             *logrus.Logger
	iOpenstackService  IOpenstackService
	iVKEClusterService IVKEService
}

func NewAppService(l *logrus.Logger, iOpenstackService IOpenstackService, iVKEClusterService IVKEService) IAppService {
	return &appService{
		logger:             l,
		iOpenstackService:  iOpenstackService,
		iVKEClusterService: iVKEClusterService,
	}
}

func (a *appService) GetOpenstackSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL string) (*gophercloud.ProviderClient, error) {
	return a.iOpenstackService.ValidateAndCreateSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL)
}

func (a *appService) CheckVKEClusterCertificateExpiration(isExpired chan bool) {
	clID := config.GlobalConfig.GetVKEConfig().ClusterID
	pjID := config.GlobalConfig.GetVKEConfig().ProjectID
	identityURL := config.GlobalConfig.GetVKEConfig().IdentityURL
	applicationCredentialID := config.GlobalConfig.GetVKEConfig().ApplicationCredentialID
	applicationCredentialSecret := config.GlobalConfig.GetVKEConfig().ApplicationCredentialSecret
	for {
		providerClient, err := a.GetOpenstackSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL)
		if err != nil {
			a.logger.WithFields(logrus.Fields{
				"application_credential_id": applicationCredentialID,
				"project_id":                pjID,
				"identity_url":              identityURL,
				"error":                     err,
			}).Error("Failed to get openstack session")
			return
		}

		token := providerClient.Token()

		getClusterResponse, err := a.iVKEClusterService.GetCluster(clID, token, identityURL)
		if err != nil {
			a.logger.WithFields(logrus.Fields{
				"cluster_id": clID,
				"error":      err,
			}).Error("Failed to get cluster")
			return
		}

		if IsExpired(getClusterResponse.ClusterCertificateExpireDate, constants.OneWeekMaintenanceWindow) {
			isExpired <- true
		}

		time.Sleep(constants.VKECheckCertificateExpirationInterval)
	}
}
