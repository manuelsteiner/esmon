package config

import (
	"esmon/constants"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	Clusters []ClusterConfig `mapstructure:"clusters" validate:"unique=Alias,unique=Endpoint,dive"`
	Http     HttpConfig      `mapstructure:"http"`
	General  GeneralConfig   `mapstructure:"general"`
}

type ClusterConfig struct {
    Alias    string `mapstructure:"alias" validate:"required"`
	Endpoint string `mapstructure:"endpoint" validate:"required,http_url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type HttpConfig struct {
	Timeout uint `mapstructure:"timeout"`
}

type GeneralConfig struct {
	RefreshInterval uint `mapstructure:"refresh_interval"`
}

func Load(configFile string) (*Config, error) {
	userConfigDir, userConfigDirError := os.UserConfigDir()
	userHomeDir, userHomeDirError := os.UserHomeDir()

	v := viper.New()

	if configFile != "" {
		v.SetConfigName(filepath.Base(configFile))
		v.AddConfigPath(filepath.Dir(configFile))
	} else {
		v.SetConfigName(constants.ProgramName)
		v.SetConfigType("toml")

		v.AddConfigPath(".")
		if userConfigDirError == nil {
			v.AddConfigPath(filepath.Join(userConfigDir, constants.ProgramName))
		}
		if userHomeDirError == nil {
			v.AddConfigPath(filepath.Join(userHomeDir, ".config", constants.ProgramName))
		}
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix(strings.ToUpper(constants.ProgramName))
	v.AutomaticEnv()

	v.SetDefault("general.refresh_interval", constants.DefaultRefreshIntervalSeconds)
	v.SetDefault("http.timeout", constants.DefaultHttpTimeout)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config

	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil

}

func Validate(config *Config) error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate.Struct(config)
}
