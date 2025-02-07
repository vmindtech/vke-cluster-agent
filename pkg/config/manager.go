package config

import (
	"github.com/spf13/viper"
	"golang.org/x/text/language"
)

const (
	VaultEnvPath         = "%s/data/%s"
	EnvironmentTypeLocal = "local"
)

var GlobalConfig IConfigureManager

type IConfigureManager interface {
	GetWebConfig() WebConfig
	GetLanguageConfig() LanguageConfig
}

type configureManager struct {
	App      WebConfig
	Language LanguageConfig
}

func NewConfigureManager() IConfigureManager {
	viper.SetConfigFile("config.json")
	viper.SetConfigType("json")

	_ = viper.ReadInConfig()

	GlobalConfig = &configureManager{
		App:      loadWebConfig(),
		Language: loadLanguageConfig(),
	}

	return GlobalConfig
}

func (c *configureManager) GetWebConfig() WebConfig {
	return c.App
}

func (c *configureManager) GetLanguageConfig() LanguageConfig {
	return c.Language
}

func loadWebConfig() WebConfig {
	return WebConfig{
		AppName: viper.GetString("APP_NAME"),
		Port:    viper.GetString("PORT"),
		Env:     viper.GetString("ENV"),
		Version: viper.GetString("VERSION"),
	}
}

func loadLanguageConfig() LanguageConfig {
	return LanguageConfig{
		Default: language.English,
		Languages: []language.Tag{
			language.English,
		},
	}
}
