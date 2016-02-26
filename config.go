package pino

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Config holds the configuration that Pino expects
type Config struct {
	IRC struct {
		Nickname string `yaml:"Nickname"`
		Server   string `yaml:"Server"`
		IsSSL    bool   `yaml:"IsSSL"`
	} `yaml:"IRC"`
	Slack struct {
		Owner string `yaml:"Owner"`
		Token string `yaml:"Token"`
	} `yaml:"Slack"`
	ChannelMapping map[string]string `yaml:"ChannelMapping"`
}

// LoadConfig returns the Config parsed from the given config file path
func LoadConfig(path string) Config {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	config := Config{}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}
