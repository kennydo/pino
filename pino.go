package pino

import (
	"fmt"

	irc "github.com/fluffle/goirc/client"
)

// Pino is the central orchestrator
type Pino struct {
	config    *Config
	ircClient *irc.Conn
}

// NewPino creates a new Pino instance
func NewPino(config *Config) (*Pino, error) {
	pino := &Pino{
		config: config,
	}

	ircClient, err := newIRCClient(&config.IRC)
	if err != nil {
		return pino, fmt.Errorf("Could not create IRC client: %v", err)
	}

	ircClient.HandleFunc(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) { fmt.Printf("Connected"); conn.Join("#pino") })

	quit := make(chan bool)
	ircClient.HandleFunc(irc.DISCONNECTED,
		func(conn *irc.Conn, line *irc.Line) { fmt.Printf("Quitting: %#v", line); quit <- true })

	if err := ircClient.Connect(); err != nil {
		fmt.Printf("Connection error: %s\n", err.Error())
		return pino, nil
	}

	<-quit

	return pino, nil
}
