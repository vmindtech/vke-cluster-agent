package config

import (
	"golang.org/x/text/language"
)

const (
	productionEnv = "production"
)

type AgentConfig struct {
	AppName string
	Env     string
	Version string
}

type LanguageConfig struct {
	Default   language.Tag
	Languages []language.Tag
}

type VKEConfig struct {
	ClusterID                   string
	ProjectID                   string
	IdentityURL                 string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	VKEURL                      string
}

func (a AgentConfig) IsProductionEnv() bool {
	return a.Env == productionEnv
}
