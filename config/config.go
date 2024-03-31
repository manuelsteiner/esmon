package config

import (
	"esmon/constants"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	Clusters []ClusterConfig `mapstructure:"clusters" validate:"unique=Alias,unique=Endpoint,dive"`
	Http     HttpConfig      `mapstructure:"http"`
	General  GeneralConfig   `mapstructure:"general"`
	Theme    ThemeConfig     `mapstructure:"theme"`
}

type ClusterConfig struct {
	Alias    string `mapstructure:"alias" validate:"required"`
	Endpoint string `mapstructure:"endpoint" validate:"required,http_url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type HttpConfig struct {
	Timeout  uint `mapstructure:"timeout"`
	Insecure bool `mapstructure:"insecure"`
}

type GeneralConfig struct {
	RefreshInterval uint `mapstructure:"refresh_interval"`
}

type ThemeConfig struct {
	LogoColor string `mapstructure:"logo_color"`

	SpinnerColor string `mapstructure:"spinner_color"`

	ForegroundColorLight       string `mapstructure:"foreground_color_light"`
	ForegroundColorDark        string `mapstructure:"foreground_color_dar"`
	ForegroundColorLightMuted  string `mapstructure:"foreground_color_light_muted"`
	ForegroundColorDarkMuted   string `mapstructure:"foreground_color_dark_muted"`
	ForegroundColorHighlighted string `mapstructure:"foreground_color_highlighted"`

	BackgroundColorStatusGreen  string `mapstructure:"background_color_status_green"`
	BackgroundColorStatusYellow string `mapstructure:"background_color_status_yellow"`
	BackgroundColorStatusRed    string `mapstructure:"background_color_status_red"`
	BackgroundColorStatusError  string `mapstructure:"background_color_status_error"`

	BorderColor      string `mapstructure:"border_color"`
	BorderColorMuted string `mapstructure:"border_color_muted"`
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
	v.SetDefault("http.insecure", constants.DefaultHttpInsecure)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config

	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	if len(config.Clusters) > 0 {
		sort.Slice(config.Clusters, func(i, j int) bool {
			return config.Clusters[i].Alias < config.Clusters[j].Alias
		})
	}

	return &config, nil

}

func Validate(config *Config) error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate.Struct(config)
}
