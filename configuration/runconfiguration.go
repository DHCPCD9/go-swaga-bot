package configuration

import (
	_ "embed"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/sirupsen/logrus"
)

//go:embed default-config.yaml
var DEFAULT_CONFIG string

type GlobalConfiguration struct {
	Discord Discord `yaml:"discord"`
	Gemini  Gemini  `yaml:"gemini"`
}
type Discord struct {
	Token            string `yaml:"token"`
	IndexAllChannels bool   `yaml:"index-all-channels"`
}
type Gemini struct {
	Token string `yaml:"token"`
	Model string `yaml:"model"`
}

var Config *GlobalConfiguration

func ReadConfig() (*GlobalConfiguration, error) {
	//Trying to read config.yml, if not, creates a new one with default text in and trying to load it again
	logrus.Debug("Reading configuration file")
	bytes, err := os.ReadFile("config.yml")
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Debug("Configuration file not found, creating a new one with default values")
			logrus.Debug("Content: ", DEFAULT_CONFIG)
			err = os.WriteFile("config.yml", []byte(DEFAULT_CONFIG), 0644)
			if err != nil {
				return nil, err
			}
			logrus.Debug("Configuration file created, reading it again")
			bytes, err = os.ReadFile("config.yml")
			if err != nil {
				return nil, err
			}
		}

	}

	logrus.Debug("Parsing configuration file")
	var config GlobalConfiguration
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}

	logrus.Debug("Configuration file parsed successfully")
	Config = &config
	return Config, nil
}
