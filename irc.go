package pino

import (
	"crypto/tls"
	"fmt"

	irc "github.com/fluffle/goirc/client"
)

type ircProxy struct {
	config         *IRCConfig
	client         *irc.Conn
	incomingEvents chan *irc.Line
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
