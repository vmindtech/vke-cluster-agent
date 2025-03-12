package request

import "time"

type UpdateClusterRequest struct {
	ClusterName                  string    `json:"cluster_name"`
	ClusterVersion               string    `json:"cluster_version"`
	ClusterStatus                string    `json:"cluster_status"`
	ClusterAPIAccess             string    `json:"cluster_api_access"`
	ClusterCertificateExpireDate time.Time `json:"cluster_certificate_expire_date"`
}
