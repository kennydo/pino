package pino

import (
	"crypto/md5"
	"fmt"
	"strings"

	slack "github.com/nlopes/slack"
)

type slackProxy struct {
	config           *SlackConfig
	client           *slack.Client
	rtm              *slack.RTM
	channelNameToID  map[SlackChannel]string
	channelIDToName  map[string]SlackChannel
	ownerID          string
	ownerIMChannelID string
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

	proxy.channelNameToID = make(map[SlackChannel]string)
	proxy.channelIDToName = make(map[string]SlackChannel)

	return proxy, nil
}

func (proxy *slackProxy) connect() error {
	go proxy.rtm.ManageConnection()

	// generate the mapping of channel name to ID, and vice versa
	channels, err := proxy.rtm.GetChannels(true)
	if err != nil {
		return fmt.Errorf("Could not get Slack channels: %v", err)
	}
	for _, channel := range channels {
		// The channel names returned by the API don't have the pound
		channelName := SlackChannel(fmt.Sprintf("#%v", channel.Name))

		// We don't care about unregistered channel
		if _, ok := proxy.config.Channels[channelName]; ok {
			proxy.channelNameToID[channelName] = channel.ID
			proxy.channelIDToName[channel.ID] = channelName
		}
	}
	fmt.Printf("Generated the following Slack channel name to ID mapping: %v\n", proxy.channelNameToID)

	users, err := proxy.rtm.GetUsers()
	if err != nil {
		return fmt.Errorf("Could not get Slack users: %v", err)
	}

	foundOwner := false
	for _, user := range users {
		if user.Name == proxy.config.Owner {
			// We found the user struct representing the owner!
			foundOwner = true

			proxy.ownerID = user.ID
			break
		}
	}
	if !foundOwner {
		return fmt.Errorf("Could not find a Slack user that matched the configured owner: %v", proxy.config.Owner)
	}

	_, _, imChannelID, err := proxy.rtm.OpenIMChannel(proxy.ownerID)
	if err != nil {
		return fmt.Errorf("Could not open a Slack IM channel with the owner: %v (%v)", proxy.config.Owner, proxy.ownerID)
	}
	proxy.ownerIMChannelID = imChannelID

	return nil
}

func generateUserIconURL(username string) string {
	return fmt.Sprintf("http://www.gravatar.com/avatar/%x?d=identicon", md5.Sum([]byte(username)))
}

func (proxy *slackProxy) sendMessageAsUser(channelName SlackChannel, username string, text string) {
	channelID := proxy.channelNameToID[channelName]
	params := slack.NewPostMessageParameters()
	params.Username = username
	params.AsUser = false
	params.IconURL = generateUserIconURL(username)

	_, _, err := proxy.rtm.PostMessage(channelID, text, params)
	if err != nil {
		fmt.Printf("Error while sending message: %v\n", err)
	}
}

func (proxy *slackProxy) sendMessageAsBot(channelName SlackChannel, text string) {
	channelID := proxy.channelNameToID[channelName]
	params := slack.NewPostMessageParameters()
	params.Username = "IRC"
	params.AsUser = false
	params.LinkNames = 1

	_, _, err := proxy.rtm.PostMessage(channelID, text, params)
	if err != nil {
		fmt.Printf("Error while sending message: %v\n", err)
	}
}

func (proxy *slackProxy) sendMessageToOwner(text string) {
	proxy.rtm.SendMessage(proxy.rtm.NewOutgoingMessage(text, proxy.ownerIMChannelID))
}

func (proxy *slackProxy) getChannelName(channelID string) SlackChannel {
	return proxy.channelIDToName[channelID]
}

// Slack decodes '&', '<', and '>' per https://api.slack.com/docs/formatting#how_to_escape_characters
// so we need to decode them.
func decodeSlackHTMLEntities(input string) string {
	output := input

	output = strings.Replace(output, "&amp;", "&", -1)
	output = strings.Replace(output, "&lt;", "<", -1)
	output = strings.Replace(output, "&gt;", ">", -1)

	return output
}
