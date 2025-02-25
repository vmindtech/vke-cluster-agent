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
	GetWebConfig() AgentConfig
	GetLanguageConfig() LanguageConfig
	GetVKEConfig() VKEConfig
}

type configureManager struct {
	Web      AgentConfig
	Language LanguageConfig
	VKE      VKEConfig
}

func NewConfigureManager() IConfigureManager {
	viper.AutomaticEnv()

	if os.Getenv("golang_env") == "development" {
		configFile := "config-" + os.Getenv("golang_env") + ".json"
		viper.SetConfigFile(configFile)
		viper.SetConfigType("json")

		err := viper.ReadInConfig()
		if err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				panic(err)
			}
		}
	}

	GlobalConfig = &configureManager{
		Web:      loadWebConfig(),
		Language: loadLanguageConfig(),
		VKE:      loadVKEConfig(),
	}

	return GlobalConfig
}

func loadWebConfig() AgentConfig {
	return AgentConfig{
		AppName: viper.GetString("APP_NAME"),
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

func (c *configureManager) GetWebConfig() AgentConfig {
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
		VKEURL:                      viper.GetString("VKE_URL"),
		ApplicationCredentialID:     viper.GetString("VKE_APPLICATION_CREDENTIAL_ID"),
		ApplicationCredentialSecret: viper.GetString("VKE_APPLICATION_CREDENTIAL_SECRET"),
	}
}
