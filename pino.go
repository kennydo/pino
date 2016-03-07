package pino

import (
	"fmt"

	irc "github.com/fluffle/goirc/client"
)

// Pino is the central orchestrator
type Pino struct {
	config   *Config
	ircProxy *ircProxy
}

// NewPino creates a new Pino instance
func NewPino(config *Config) (*Pino, error) {
	pino := &Pino{
		config: config,
	}

	ircProxy, err := newIRCProxy(&config.IRC)
	if err != nil {
		return pino, fmt.Errorf("Could not create IRC client: %v", err)
	}

	ircProxy.registerEventHandler(irc.CONNECTED, func(conn *irc.Conn, line *irc.Line) {
		fmt.Printf("Connected\n")
	})

	quit := make(chan bool)
	ircProxy.registerEventHandler(irc.DISCONNECTED, func(conn *irc.Conn, line *irc.Line) {
		fmt.Printf("Quitting: %#v\n", line)
		quit <- true
	})

	// Listening to the normal events
	ircProxy.registerEventHandler(irc.ACTION, func(conn *irc.Conn, line *irc.Line) {
		action := line.Text()
		fmt.Printf("ACTION: %v %s\n", line.Nick, action)
	})

	ircProxy.registerEventHandler(irc.JOIN, func(conn *irc.Conn, line *irc.Line) {
		channel := line.Text()
		fmt.Printf("JOIN: %v(%v) has joined %v\n", line.Nick, line.Src, channel)
	})

	ircProxy.registerEventHandler("NAMES", func(conn *irc.Conn, line *irc.Line) {
		fmt.Printf("NAMES: %#v\n", line)
	})

	ircProxy.registerEventHandler(irc.INVITE, func(conn *irc.Conn, line *irc.Line) {
		// Actually doing anything with invites has not been implemented yet.
		channel := line.Args[1]
		fmt.Printf("INVITE: %v(%v) invited you to %v\n", line.Nick, line.Src, channel)
	})

	ircProxy.registerEventHandler(irc.KICK, func(conn *irc.Conn, line *irc.Line) {
		kickee := line.Args[1]
		reason := line.Args[2:]
		fmt.Printf("KICK: (%v) %v has kicked %v (%v)\n", line.Target(), line.Nick, kickee, reason)
	})

	ircProxy.registerEventHandler(irc.MODE, func(conn *irc.Conn, line *irc.Line) {
		mode := line.Args[1]
		destination := line.Args[2]
		fmt.Printf("MODE: (%v) %v has set mode: %v %v\n", line.Target(), line.Nick, mode, destination)
	})

	ircProxy.registerEventHandler(irc.NICK, func(conn *irc.Conn, line *irc.Line) {
		newNick := line.Text()
		fmt.Printf("NICK: %v is now known as %v\n", line.Nick, newNick)
	})

	ircProxy.registerEventHandler(irc.PART, func(conn *irc.Conn, line *irc.Line) {
		reason := line.Text()
		fmt.Printf("PART: (%v) %v(%v) has left (%s)\n", line.Target(), line.Nick, line.Src, reason)
	})
	ircProxy.registerEventHandler(irc.PRIVMSG, func(conn *irc.Conn, line *irc.Line) {
		fmt.Printf("PRIVMSG: (%v) <%v> %v\n", line.Target(), line.Nick, line.Text())
	})
	ircProxy.registerEventHandler(irc.QUIT, func(conn *irc.Conn, line *irc.Line) {
		fmt.Printf("QUIT: %v(%v) has quit\n", line.Nick, line.Src)
	})

	ircProxy.registerEventHandler(irc.TOPIC, func(conn *irc.Conn, line *irc.Line) {
		newTopic := line.Text()
		fmt.Printf("TOPIC: (%v) %v has changed the topic to \"%v\"\n", line.Target(), line.Nick, newTopic)
	})
	// End listening to all events

	if err := ircProxy.connect(); err != nil {
		fmt.Printf("Connection error: %s\n", err.Error())
		return pino, err
	}

	for _, ircChannel := range config.ChannelMapping {
		fmt.Printf("Joining IRC channel: %v\n", ircChannel)
		ircProxy.join(ircChannel)
	}

	<-quit

	return pino, nil
}
