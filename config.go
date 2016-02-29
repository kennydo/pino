package pino

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config holds the configuration that Pino expects
type Config struct {
	IRC            IRCConfig         `yaml:"IRC"`
	Slack          SlackConfig       `yaml:"Slack"`
	ChannelMapping map[string]string `yaml:"ChannelMapping"`
}

// IRCConfig define the IRC-specific config
type IRCConfig struct {
	Nickname string `yaml:"Nickname"`
	Name     string `yaml:"Name"`
	Server   string `yaml:"Server"`
	IsSSL    bool   `yaml:"IsSSL"`
}

// SlackConfig defines the Slack-specific config
type SlackConfig struct {
	Owner string `yaml:"Owner"`
	Token string `yaml:"Token"`
}

// LoadConfig returns the Config parsed from the given config file path
func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("Unable to read config file from %v: %v", path, err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("Unable to parse YAML from config file %v: %v", path, err)
	}

	return config, nil
}
