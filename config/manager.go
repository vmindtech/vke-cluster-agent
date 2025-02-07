package config

import (
	"os"

	"github.com/spf13/viper"
	"golang.org/x/text/language"
)

const (
	EnvironmentTypeLocal = "local"
)

var GlobalConfig IConfigureManager

type IConfigureManager interface {
	GetWebConfig() WebConfig
	GetLanguageConfig() LanguageConfig
	GetVKEConfig() VKEConfig
}

type configureManager struct {
	Web      WebConfig
	Language LanguageConfig
	VKE      VKEConfig
}

func NewConfigureManager() IConfigureManager {
	viper.SetConfigFile("config-" + os.Getenv("golang_env") + ".json")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config-" + os.Getenv("golang_env") + ".json")

	_ = viper.ReadInConfig()

	GlobalConfig = &configureManager{
		Web:      loadWebConfig(),
		Language: loadLanguageConfig(),
		VKE:      loadVKEConfig(),
	}

	return GlobalConfig
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

func (c *configureManager) GetWebConfig() WebConfig {
	return c.Web
}

func (c *configureManager) GetLanguageConfig() LanguageConfig {
	return c.Language
}

func (c *configureManager) GetVKEConfig() VKEConfig {
	return c.VKE
}

func loadVKEConfig() VKEConfig {
	return VKEConfig{
		ClusterID:                   viper.GetString("VKE_CLUSTER_ID"),
		ProjectID:                   viper.GetString("VKE_PROJECT_ID"),
		IdentityURL:                 viper.GetString("VKE_IDENTITY_URL"),
		ApplicationCredentialID:     viper.GetString("VKE_APPLICATION_CREDENTIAL_ID"),
		ApplicationCredentialSecret: viper.GetString("VKE_APPLICATION_CREDENTIAL_SECRET"),
	}
}
