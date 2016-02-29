package pino

import (
	"crypto/tls"
	"fmt"

	irc "github.com/fluffle/goirc/client"
)

func newIRCClient(config *IRCConfig) (*irc.Conn, error) {
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
	clientConfig.SSL = config.IsSSL
	clientConfig.SSLConfig = &tls.Config{InsecureSkipVerify: true}

	client := irc.Client(clientConfig)

	return client, nil
}
