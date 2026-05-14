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

// RKE2 Related Constants
const (
	RKE2RestartWaitDuration          = 600 * time.Second
	ServiceVerificationInterval      = 5 * time.Second
	ServiceVerificationTimeout       = 2 * time.Minute
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
