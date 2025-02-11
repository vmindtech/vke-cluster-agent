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
