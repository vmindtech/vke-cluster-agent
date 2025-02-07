package config

import (
	"golang.org/x/text/language"
)

const (
	productionEnv = "production"
)

type WebConfig struct {
	AppName string
	Port    string
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
}

func (w WebConfig) IsProductionEnv() bool {
	return w.Env == productionEnv
}
