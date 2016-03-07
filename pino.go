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
	pino.ircProxy = ircProxy

	return pino, nil
}

// Run connects to IRC and Slack and runs the main loop
func (pino *Pino) Run() error {
	if err := pino.ircProxy.connect(); err != nil {
		return fmt.Errorf("Connection error: %s\n", err.Error())
	}

	// Channel to signal that the program should stop running
	quit := make(chan bool)

	go pino.handleIRCEvents(quit)

	<-quit

	return nil
}

// Consumes incoming IRC events in a loop
func (pino *Pino) handleIRCEvents(quit chan bool) {
	for {
		select {
		case line := <-pino.ircProxy.incomingEvents:
			switch line.Cmd {
			case irc.CONNECTED:
				fmt.Printf("Connected to IRC!\n")
				ircChannels := pino.config.getUsedIRCChannels()
				for _, ircChannel := range ircChannels {
					fmt.Printf("Joining IRC channel: %v\n", ircChannel)
					pino.ircProxy.join(ircChannel)
				}

			case irc.DISCONNECTED:
				fmt.Printf("Disconnected from IRC!")

			case irc.ACTION:
				action := line.Text()
				fmt.Printf("ACTION: %v %s\n", line.Nick, action)

			case irc.JOIN:
				channel := line.Text()
				fmt.Printf("JOIN: %v(%v) has joined %v\n", line.Nick, line.Src, channel)

			case irc.INVITE:
				// Actually doing anything with invites has not been implemented yet.
				channel := line.Args[1]
				fmt.Printf("INVITE: %v(%v) invited you to %v\n", line.Nick, line.Src, channel)

			case irc.KICK:
				kickee := line.Args[1]
				reason := line.Args[2:]
				fmt.Printf("KICK: (%v) %v has kicked %v (%v)\n", line.Target(), line.Nick, kickee, reason)

			case irc.MODE:
				if len(line.Args) == 2 {
					// This was a User mode command
					destination := line.Args[0]
					mode := line.Args[1]
					fmt.Printf("MODE: %v has set mode %v on %v\n", line.Nick, mode, destination)
				} else {
					// This was a Channel mode command
					channel := line.Args[0]
					mode := line.Args[1]
					destination := line.Args[2]
					fmt.Printf("MODE: (%v) %v has set mode %v on %v\n", channel, line.Nick, mode, destination)
				}

			case irc.NICK:
				newNick := line.Text()
				fmt.Printf("NICK: %v is now known as %v\n", line.Nick, newNick)

			case irc.PART:
				reason := line.Text()
				fmt.Printf("PART: (%v) %v(%v) has left (%s)\n", line.Target(), line.Nick, line.Src, reason)

			case irc.PRIVMSG:
				fmt.Printf("PRIVMSG: (%v) <%v> %v\n", line.Target(), line.Nick, line.Text())

			case irc.QUIT:
				fmt.Printf("QUIT: %v(%v) has quit\n", line.Nick, line.Src)

			case irc.TOPIC:
				newTopic := line.Text()
				fmt.Printf("TOPIC: (%v) %v has changed the topic to \"%v\"\n", line.Target(), line.Nick, newTopic)

			default:
				fmt.Printf("Received unrecognized line: %#v\n", line)
			}
		default:
			// No message was received
		}
	}
}
