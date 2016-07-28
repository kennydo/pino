package pino

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config holds the configuration that Pino expects
type Config struct {
	IRC            IRCConfig                   `yaml:"IRC"`
	Slack          SlackConfig                 `yaml:"Slack"`
	ChannelMapping map[SlackChannel]IRCChannel `yaml:"ChannelMapping"`
}

// IRCChannel is the name of an IRC channel, like "#CAA"
type IRCChannel string

// IRCChannelKey is an optional password for an IRC channel
type IRCChannelKey string

// SlackChannel is the name of a Slack channel, like "#CAA-on-Slack"
type SlackChannel string

// IRCConfig define the IRC-specific config
type IRCConfig struct {
	Nickname       string                       `yaml:"Nickname"`
	Name           string                       `yaml:"Name"`
	Server         string                       `yaml:"Server"`
	Password       string                       `yaml:"Password"`
	IsSSL          bool                         `yaml:"IsSSL"`
	Channels       map[IRCChannel]IRCChannelKey `yaml:"Channels"`
	HighlightRules []IRCHighlightRuleConfig     `yaml:"HighlightRules"`
}

// IRCHighlightRuleConfig defines when to directly ping the owner on Slack.
// You can define a nick pattern, a message pattern, or both.
// If a pattern is not defined, then it is assumed to match all values for that.
// The first rule that matches is executed. Default is to not highlight.
type IRCHighlightRuleConfig struct {
	NickPattern     string `yaml:"NickPattern"`
	MessagePattern  string `yaml:"MessagePattern"`
	ShouldHighlight bool   `yaml:"ShouldHighlight"`
}

// SlackConfig defines the Slack-specific config
type SlackConfig struct {
	Owner    string                  `yaml:"Owner"`
	Token    string                  `yaml:"Token"`
	Channels map[SlackChannel]string `yaml:"Channels"`
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

	// Verify that the channel mapping is consistent with the configured IRC/Slack Channels
	for slackChannel, ircChannel := range config.ChannelMapping {
		if _, ok := config.IRC.Channels[ircChannel]; !ok {
			return config, fmt.Errorf("IRC channel '%v' was specified in the channel mapping, but wasn't configured under IRC", ircChannel)
		}

		if _, ok := config.Slack.Channels[slackChannel]; !ok {
			return config, fmt.Errorf("Slack channel '%v' was specified in the channel mapping, but wasn't configured under Slack", slackChannel)
		}
	}

	return config, nil
}

func (config *Config) getUsedIRCChannels() []IRCChannel {
	channels := make([]IRCChannel, len(config.ChannelMapping))
	i := 0

	for _, ircChannel := range config.ChannelMapping {
		channels[i] = ircChannel
		i++
	}

	return channels
}
