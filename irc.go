package pino

import (
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"

	irc "github.com/fluffle/goirc/client"
)

type ircProxy struct {
	config         *IRCConfig
	client         *irc.Conn
	incomingEvents chan *irc.Line
	highlightRules []*ircHighlightRule
}

type ircHighlightRule struct {
	nickRegexp      *regexp.Regexp
	messageRegexp   *regexp.Regexp
	shouldHighlight bool
}

func newIRCProxy(config *IRCConfig) (*ircProxy, error) {
	proxy := new(ircProxy)
	proxy.config = config
	proxy.incomingEvents = make(chan *irc.Line)

	nick := config.Nickname
	if nick == "" {
		return nil, fmt.Errorf("Nickname must be defined in IRC config")
	}
	name := config.Name
	if name == "" {
		name = "Pino"
	}
	ident := name

	server := config.Server
	if server == "" {
		return nil, fmt.Errorf("Server must be defined in IRC config")
	}

	proxy.highlightRules = make([]*ircHighlightRule, len(config.HighlightRules))
	for i, highlightConfig := range config.HighlightRules {
		var nickRegexp *regexp.Regexp
		var messageRegexp *regexp.Regexp

		if highlightConfig.NickPattern != "" {
			nickRegexp = regexp.MustCompile(highlightConfig.NickPattern)
		}

		if highlightConfig.MessagePattern != "" {
			messageRegexp = regexp.MustCompile(highlightConfig.MessagePattern)
		}

		proxy.highlightRules[i] = &ircHighlightRule{
			nickRegexp:      nickRegexp,
			messageRegexp:   messageRegexp,
			shouldHighlight: highlightConfig.ShouldHighlight,
		}
	}

	clientConfig := irc.NewConfig(nick, ident, name)
	clientConfig.Version = "Version"
	clientConfig.QuitMessage = "Bye!"
	clientConfig.Server = server
	clientConfig.Pass = config.Password
	clientConfig.SSL = config.IsSSL
	clientConfig.SSLConfig = &tls.Config{InsecureSkipVerify: true}

	proxy.client = irc.Client(clientConfig)
	proxy.client.EnableStateTracking()

	proxy.registerEventHandlers()

	return proxy, nil
}

func (proxy *ircProxy) registerEventHandlers() {
	eventTypes := []string{
		irc.CONNECTED,
		irc.DISCONNECTED,
		irc.ACTION,
		irc.JOIN,
		irc.INVITE,
		irc.KICK,
		irc.MODE,
		irc.NICK,
		irc.PART,
		irc.PRIVMSG,
		irc.QUIT,
		irc.TOPIC,
	}

	sendLineToChannel := func(conn *irc.Conn, line *irc.Line) {
		proxy.incomingEvents <- line
	}

	for _, eventType := range eventTypes {
		proxy.client.HandleFunc(eventType, sendLineToChannel)
	}
}

func (proxy *ircProxy) connect() error {
	return proxy.client.Connect()
}

// Connect to the configured channel
func (proxy *ircProxy) join(channel IRCChannel) {
	key := proxy.config.Channels[channel]
	proxy.client.Join(string(channel), string(key))
}

// Get the list of names in a channel.
// Note that this is sometimes wrong right when we join a channel (before we've received the list of names).
func (proxy *ircProxy) names(channel IRCChannel) []string {
	statefulChannel := proxy.client.StateTracker().GetChannel(string(channel))
	channelNicks := statefulChannel.Nicks

	names := make([]string, len(channelNicks))
	i := 0
	for name := range channelNicks {
		names[i] = name
		i++
	}

	return names
}

func (proxy *ircProxy) snapshotOfNicksInChannels() map[IRCChannel]map[string]bool {
	mapping := make(map[IRCChannel]map[string]bool)

	st := proxy.client.StateTracker()
	for channelName := range proxy.config.Channels {
		channel := st.GetChannel(string(channelName))
		if channel == nil {
			continue
		}

		mapping[channelName] = make(map[string]bool)

		for nickname := range channel.Nicks {
			mapping[channelName][nickname] = true
		}
	}

	return mapping
}

func (proxy *ircProxy) sendMessage(channel IRCChannel, text string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		proxy.client.Privmsg(string(channel), line)
	}
}

func (proxy *ircProxy) sendAction(channel IRCChannel, action string) {
	lines := strings.Split(action, "\n")
	for _, line := range lines {
		proxy.client.Action(string(channel), line)
	}
}

func isBufferPlaybackStartLine(line *irc.Line) bool {
	if line.Cmd != irc.PRIVMSG {
		return false
	}

	if line.Nick != "***" {
		return false
	}

	return line.Text() == "Buffer Playback..."
}

func isBufferPlaybackEndLine(line *irc.Line) bool {
	if line.Cmd != irc.PRIVMSG {
		return false
	}

	if line.Nick != "***" {
		return false
	}

	return line.Text() == "Playback Complete."
}

func (proxy *ircProxy) shouldHighlightOwnerOnMessageByNick(message, nick string) bool {
	if len(proxy.highlightRules) == 0 {
		return false
	}

	for _, rule := range proxy.highlightRules {
		var nickMatches, messageMatches bool

		if (rule.nickRegexp != nil) && (rule.nickRegexp.FindString(nick) != "") {
			nickMatches = true
		}

		if (rule.messageRegexp != nil) && (rule.messageRegexp.FindString(message) != "") {
			messageMatches = true
		}

		if ((rule.nickRegexp == nil) || nickMatches) && ((rule.messageRegexp == nil) || messageMatches) {
			return rule.shouldHighlight
		}
	}

	return false
}
