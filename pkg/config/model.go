package config

import (
	"golang.org/x/text/language"
)

const (
	productionEnv = "production"
)

type Configuration struct {
	App      WebConfig
	Language LanguageConfig
}

type WebConfig struct {
	AppName string
	Port    string
	Env     string
	Version string
	Debug   bool
}

type LanguageConfig struct {
	Default   language.Tag
	Languages []language.Tag
}

func (w WebConfig) IsProductionEnv() bool {
	return w.Env == productionEnv
}
