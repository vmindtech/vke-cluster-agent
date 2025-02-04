package healthcheck

import (
	"time"
)

var healthCheck *config

const (
	mysqlConnTimeout = 20 * time.Second
)

type config struct {
	serverUp bool
}

func InitHealthCheck() {
	healthCheck = &config{
		serverUp: true,
	}
}

func Readiness() map[string]bool {
	return map[string]bool{
		"serverUp": true,
	}
}

func Liveness() bool {
	return healthCheck.serverUp
}

func ServerShutdown() {
	healthCheck.serverUp = false
}

func IsConnectionSuccessful(conn map[string]bool) bool {
	for _, status := range conn {
		if !status {
			return false
		}
	}

	return true
}
