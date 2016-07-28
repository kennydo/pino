package pino

import (
	"fmt"

	irc "github.com/fluffle/goirc/client"
	"github.com/nlopes/slack"
	"gopkg.in/kyokomi/emoji.v1"
)

// Pino is the central orchestrator
type Pino struct {
	config                   *Config
	ircProxy                 *ircProxy
	slackProxy               *slackProxy
	slackChannelToIRCChannel map[SlackChannel]IRCChannel
	ircChannelToSlackChannel map[IRCChannel]SlackChannel
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

	slackProxy, err := newSlackProxy(&config.Slack)
	if err != nil {
		return pino, fmt.Errorf("Could not create Slack client: %v", err)
	}
	pino.slackProxy = slackProxy

	pino.slackChannelToIRCChannel = make(map[SlackChannel]IRCChannel)
	pino.ircChannelToSlackChannel = make(map[IRCChannel]SlackChannel)
	// Set up the Slack channel -> IRC channel name mappings, and vice versa
	for slackChannel, ircChannel := range pino.config.ChannelMapping {
		pino.slackChannelToIRCChannel[slackChannel] = ircChannel
		pino.ircChannelToSlackChannel[ircChannel] = slackChannel
	}

	return pino, nil
}

// Run connects to IRC and Slack and runs the main loop
func (pino *Pino) Run() error {
	if err := pino.ircProxy.connect(); err != nil {
		return fmt.Errorf("IRC connection error: %s", err.Error())
	}
	if err := pino.slackProxy.connect(); err != nil {
		return fmt.Errorf("Slack connection error: %s", err.Error())
	}

	// Channel to signal that the program should stop running
	quit := make(chan bool)

	go pino.handleIRCEvents(quit)
	go pino.handleSlackEvents(quit)

	<-quit

	return nil
}

// Consumes incoming IRC events in a loop
func (pino *Pino) handleIRCEvents(quit chan bool) {
	previousNickMemberships := pino.ircProxy.snapshotOfNicksInChannels()

	// For buffer playback, we care about whether the buffer playback mode changed in the previous line
	// in deciding whether to print the subsequent lines
	wasInBufferPlaybackMode := false
	isInBufferPlaybackMode := false

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

				message := fmt.Sprintf("Connected to IRC on %v!", pino.ircProxy.config.Server)
				pino.slackProxy.sendMessageToOwner(message)

			case irc.DISCONNECTED:
				fmt.Printf("Disconnected from IRC!")
				message := fmt.Sprintf("Disconnected from IRC on %v!", pino.ircProxy.config.Server)
				pino.slackProxy.sendMessageToOwner(message)

			case irc.ACTION:
				channel := IRCChannel(line.Target())
				action := line.Text()
				username := line.Nick

				fmt.Printf("ACTION: %v %s\n", username, action)
				message := fmt.Sprintf("> *%v %v*", username, action)

				if !isInBufferPlaybackMode {
					pino.slackProxy.sendMessageAsUser(pino.ircChannelToSlackChannel[channel], username, message)
				}

			case irc.JOIN:
				channel := IRCChannel(line.Text())
				username := line.Nick
				usermask := line.Src

				fmt.Printf("JOIN: %v(%v) has joined %v\n", line.Nick, line.Src, channel)
				message := fmt.Sprintf("> *%v* (%v) joined the channel", username, usermask)
				pino.slackProxy.sendMessageAsBot(pino.ircChannelToSlackChannel[channel], message)

				previousNickMemberships = pino.ircProxy.snapshotOfNicksInChannels()
			case irc.INVITE:
				// Actually doing anything with invites has not been implemented yet.
				channel := line.Args[1]
				fmt.Printf("INVITE: %v(%v) invited you to %v\n", line.Nick, line.Src, channel)

			case irc.KICK:
				channel := IRCChannel(line.Target())
				kicker := line.Nick
				kickee := line.Args[1]
				reason := line.Args[2]
				fmt.Printf("KICK: (%v) %v has kicked %v (%v)\n", channel, kicker, kickee, reason)

				message := fmt.Sprintf("> *%v* kicked *%v* from the channel (%v)", kicker, kickee, reason)
				pino.slackProxy.sendMessageAsBot(pino.ircChannelToSlackChannel[channel], message)

				previousNickMemberships = pino.ircProxy.snapshotOfNicksInChannels()
			case irc.MODE:
				username := line.Nick
				mode := line.Args[1]

				if len(line.Args) == 2 {
					// This was a User mode command
					destination := line.Args[0]
					fmt.Printf("MODE: %v has set mode %v on %v\n", username, mode, destination)
				} else {
					// This was a Channel mode command
					channel := IRCChannel(line.Args[0])
					destination := line.Args[2]
					fmt.Printf("MODE: (%v) %v sets %v %v\n", channel, username, mode, destination)

					message := fmt.Sprintf("> *%v* sets *%v* *%v*", username, mode, destination)
					pino.slackProxy.sendMessageAsBot(pino.ircChannelToSlackChannel[channel], message)
				}

			case irc.NICK:
				oldNick := line.Nick
				newNick := line.Text()
				fmt.Printf("NICK: %v is now known as %v\n", oldNick, newNick)

				message := fmt.Sprintf("> %v is now known as *%v*", oldNick, newNick)
				for ircChannel, nicks := range previousNickMemberships {
					if _, ok := nicks[oldNick]; !ok {
						// The user was not in this channel
						continue
					}

					slackChannel := pino.ircChannelToSlackChannel[ircChannel]
					pino.slackProxy.sendMessageAsBot(slackChannel, message)
				}

				previousNickMemberships = pino.ircProxy.snapshotOfNicksInChannels()
			case irc.PART:
				channel := IRCChannel(line.Target())
				reason := line.Text()
				username := line.Nick
				usermask := line.Src
				fmt.Printf("PART: (%v) %v(%v) has left (%s)\n", channel, username, usermask, reason)

				message := fmt.Sprintf("> *%v* (%v) left the channel", username, usermask)
				pino.slackProxy.sendMessageAsBot(pino.ircChannelToSlackChannel[channel], message)

				previousNickMemberships = pino.ircProxy.snapshotOfNicksInChannels()
			case irc.PRIVMSG:
				target := line.Target()
				username := line.Nick
				text := line.Text()

				fmt.Printf("PRIVMSG: (%v) <%v> %v\n", target, username, text)

				if isInBufferPlaybackMode {
					if isBufferPlaybackEndLine(line) {
						isInBufferPlaybackMode = false
					}
				} else {
					if isBufferPlaybackStartLine(line) {
						isInBufferPlaybackMode = true
					}
				}

				// Simply being out of buffer playback mode is not sufficient for deciding to output the line
				// because we consider ourselves out of the buffer playback mode on the line where playback ends.
				if !wasInBufferPlaybackMode && !isInBufferPlaybackMode {

					possibleChannel := IRCChannel(target)
					if slackChannel, ok := pino.ircChannelToSlackChannel[possibleChannel]; ok {

						if pino.ircProxy.shouldHighlightOwnerOnMessageByNick(text, username) {
							pino.slackProxy.sendMessageAsBot(
								slackChannel,
								fmt.Sprintf("@%v: you were pinged by %v", pino.slackProxy.config.Owner, username),
							)
						}

						pino.slackProxy.sendMessageAsUser(slackChannel, username, text)
					}
				}

				wasInBufferPlaybackMode = isInBufferPlaybackMode

			case irc.QUIT:
				username := line.Nick
				usermask := line.Src
				reason := line.Args[0]

				fmt.Printf("QUIT: %v(%v) has quit (%v)\n", username, usermask, reason)

				message := fmt.Sprintf("> *%v* (%v) left IRC (%v)", username, usermask, reason)
				for ircChannel, nicks := range previousNickMemberships {
					if _, ok := nicks[username]; !ok {
						// The user was not in this channel
						continue
					}

					slackChannel := pino.ircChannelToSlackChannel[ircChannel]
					pino.slackProxy.sendMessageAsBot(slackChannel, message)
				}

				previousNickMemberships = pino.ircProxy.snapshotOfNicksInChannels()
			case irc.TOPIC:
				channel := IRCChannel(line.Target())
				username := line.Nick
				topic := line.Text()
				fmt.Printf("TOPIC: (%v) %v has changed the topic to \"%v\"\n", channel, username, topic)

				message := fmt.Sprintf("> *%v* changed the topic to *%v*", username, topic)
				pino.slackProxy.sendMessageAsBot(pino.ircChannelToSlackChannel[channel], message)

			default:
				fmt.Printf("Received unrecognized line: %#v\n", line)
			}
		}

	}
}

// Consumes incoming Slack events in a loop
func (pino *Pino) handleSlackEvents(quit chan bool) {
	for {
		select {
		case msg := <-pino.slackProxy.rtm.IncomingEvents:
			switch event := msg.Data.(type) {
			case *slack.MessageEvent:
				pino.handleSlackMessageEvent(event, quit)
			case *slack.ConnectingEvent:
			case *slack.ConnectedEvent:
			case *slack.HelloEvent:
				fmt.Printf("Connected to Slack!\n")
			case *slack.UserTypingEvent:
			case *slack.LatencyReport:
			case *slack.PresenceChangeEvent:
			case *slack.ReconnectUrlEvent:
			case *slack.AckMessage:
			default:
				fmt.Printf("Received unrecognized Slack msg: %#v\n", msg.Data)
			}
		}
	}
}

func (pino *Pino) handleSlackMessageEvent(event *slack.MessageEvent, quit chan bool) {
	// For development, we'll still want to print out all received messages
	//fmt.Printf("Message: %#v\n", event)

	slackChannel := pino.slackProxy.getChannelName(event.Channel)
	destinationIRCChannel := pino.slackChannelToIRCChannel[slackChannel]

	if event.BotID != "" {
		// Sending any messages from a bot to IRC might cause a vicious cycle
		return
	}

	// We only support a small subset of message subtypes:
	// - "" (no subtype means it's a normal message)
	// - "me_message" (a /me action)
	if event.SubType != "me_message" && event.SubType != "" {
		fmt.Printf("Ignoring message with unsupported subtype: %#v\n", event)
		return
	}

	text := pino.slackProxy.renderFormattedMessageForDisplay(event.Text)

	text = decodeSlackHTMLEntities(text)

	// Convert stuff like ":pizza:" to the actual pizza emoji
	text = emoji.Sprint(text)

	if event.SubType == "me_message" {
		pino.ircProxy.sendAction(destinationIRCChannel, text)
		return
	}

	// In the normal case, it's a normal message
	pino.ircProxy.sendMessage(destinationIRCChannel, text)
}
