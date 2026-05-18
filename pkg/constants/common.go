package constants

import "time"

// Language
const (
	TurkishLanguage = "tr"
	EnglishLanguage = "en"
)

// Maintenance Window
const (
	OneHourMaintenanceWindow = 1 * time.Hour
	OneWeekMaintenanceWindow = 7 * 24 * time.Hour
	TwoWeekMaintenanceWindow = 14 * 24 * time.Hour
)

// VKE Check Certificate Expiration Interval
const (
	VKECheckCertificateExpirationInterval = 1 * time.Hour
)

// Host filesystem path (DaemonSet volumeMount). Used as fallback for systemctl.
const HostRootPath = "/host"

// HostInitPID is the host init/systemd PID visible when the pod runs with hostPID.
const HostInitPID = "1"

// HostSystemctlPath is the systemctl path on typical RKE2 nodes.
const HostSystemctlPath = "/usr/bin/systemctl"

// RKE2 Related Constants
const (
	RKE2RestartWaitDuration          = 10 * time.Minute
	ServiceVerificationInterval      = 5 * time.Second
	ServiceVerificationTimeout       = 10 * time.Minute
	KubeconfigVerificationTimeout    = 2 * time.Minute
	ClusterUpdateVerificationTimeout = 10 * time.Minute
)

// Node Label Selectors
const (
	MasterNodeLabelSelector = "node-role.kubernetes.io/control-plane=true"
	WorkerNodeLabelSelector = "!node-role.kubernetes.io/master,!node-role.kubernetes.io/control-plane"
)

// New constants
const (
	CertificateCheckInterval = 1 * time.Hour

	RenewalProcessTimeout = 30 * time.Minute
)
