package pino

import (
	"fmt"

	slack "github.com/nlopes/slack"
)

type slackProxy struct {
	config *SlackConfig
	client *slack.Client
	rtm    *slack.RTM
}

func newSlackProxy(config *SlackConfig) (*slackProxy, error) {
	proxy := new(slackProxy)
	proxy.config = config

	token := config.Token
	if token == "" {
		return nil, fmt.Errorf("Token must be defined in Slack config")
	}

	proxy.client = slack.New(token)
	proxy.rtm = proxy.client.NewRTM()

	return proxy, nil
}
