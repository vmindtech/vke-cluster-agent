package resource

import "time"

type NodeGroup struct {
	ClusterUUID      string `json:"cluster_uuid"`
	NodeGroupUUID    string `json:"node_group_uuid"`
	NodeGroupName    string `json:"node_group_name"`
	NodeGroupMinSize int    `json:"node_group_min_size"`
	NodeGroupMaxSize int    `json:"node_group_max_size"`
	NodeDiskSize     int    `json:"node_disk_size"`
	NodeFlavorUUID   string `json:"node_flavor_uuid"`
	NodeGroupsType   string `json:"node_groups_type"`
	CurrentNodes     int    `json:"current_nodes"`
	NodeGroupsStatus string `json:"node_groups_status"`
}

type VKEClusterResponse struct {
	ClusterUUID                  string      `json:"cluster_uuid"`
	ClusterName                  string      `json:"cluster_name"`
	ClusterVersion               string      `json:"cluster_version"`
	ClusterStatus                string      `json:"cluster_status"`
	ClusterProjectUUID           string      `json:"cluster_project_uuid"`
	ClusterLoadbalancerUUID      string      `json:"cluster_loadbalancer_uuid"`
	ClusterMasterServerGroup     NodeGroup   `json:"cluster_master_server_group_uuid"`
	ClusterWorkerServerGroups    []NodeGroup `json:"cluster_worker_server_groups_uuid"`
	ClusterSubnets               []string    `json:"cluster_subnets"`
	ClusterEndpoint              string      `json:"cluster_endpoint"`
	ClusterAPIAccess             string      `json:"cluster_api_access"`
	ClusterCertificateExpireDate time.Time   `json:"cluster_certificate_expire_date"`
}
