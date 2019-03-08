# go-twitch-irc [![Build Status](https://travis-ci.org/gempir/go-twitch-irc.svg?branch=master)](https://travis-ci.org/gempir/go-twitch-irc) [![Coverage Status](https://coveralls.io/repos/github/gempir/go-twitch-irc/badge.svg?branch=master)](https://coveralls.io/github/gempir/go-twitch-irc?branch=master)

This is an irc client for connecting to twitch. It handles the annoying stuff like irc tag parsing.
I highly recommend reading the godoc below, but this readme gives a basic overview of the functionality.

godoc: https://godoc.org/github.com/gempir/go-twitch-irc

## Getting Started
```go
package main

import (
	"github.com/gempir/go-twitch-irc"
	"fmt"
)

func main() {
	client := twitch.NewClient("justinfan123123", "oauth:123123123")

	client.OnNewMessage(func(user twitch.User, message twitch.PrivateMessage) {
		fmt.Println(message.Message)
	})

	client.Join("gempir")

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
```
### Available Data

The twitch.User and MessageType structs reflect the data Twitch provides, minus any fields that have been marked as deprecated:
```go
type User struct {
	ID          string
	Name        string
	DisplayName string
	Color       string
	Badges      map[string]int
}

type WhisperMessage struct {
	rawMessage
	userMessage
}

type PrivateMessage struct {
	tmiMessage
	userMessage
	Bits int
}

type ClearChatMessage struct {
	tmiMessage
	MsgID          string // Possible Values: "Clear", "Ban", "Timeout"
	BanDuration    int
	TargetUserID   string
	TargetUsername string
}

type RoomStateMessage struct {
	roomMessage
	Language string
	State    map[string]int
}

type UserNoticeMessage struct {
	tmiMessage
	userMessage
	MsgID     string
	MsgParams map[string]string
	SystemMsg string
}

type UserStateMessage struct {
	channelMessage
	EmoteSets []string
}

type NoticeMessage struct {
	channelMessage
	MsgID string
}
```

The embedded structs look like this:
```go
type rawMessage struct {
	Type    MessageType
	RawType string
	Raw     string
	Tags    map[string]string
	Message string
}

type channelMessage struct {
	rawMessage
	Channel string
}

type roomMessage struct {
	channelMessage
	RoomID string
}

type tmiMessage struct {
	roomMessage
	ID   string
	Time time.Time
}

type userMessage struct {
	Action bool
	Emotes []*Emote
}
```

### Client Methods

These are the available methods of the client so you can get your bot going:

	func (c *Client) Say(channel, text string)
	func (c *Client) Whisper(username, text string)
	func (c *Client) Join(channel string)
	func (c *Client) Depart(channel string)
	func (c *Client) Userlist(channel string) ([]string, error)
	func (c *Client) Connect() error
	func (c *Client) Disconnect() error

### Options

On your client you can configure multiple options:
```go
client.IrcAddress = "127.0.0.1:3030" // for custom irc server
client.TLS = false // enabled by default, will connect to non TLS server of twitch when off or the given client.IrcAddress
client.SetupCmd = "LOGIN custom_command_here" // Send a custom command on successful IRC connection, before authentication.
```
### Callbacks

These callbacks are available to pass to the client:
```go
client.OnConnect(func() {})
client.OnNewWhisper(func(user User, message WhisperMessage) {})
client.OnNewMessage(func(user User, message PrivateMessage) {})
client.OnNewClearchatMessage(func(message ClearChatMessage) {})
client.OnNewRoomstateMessage(func(message RoomStateMessage) {})
client.OnNewUsernoticeMessage(func(user User, message UserNoticeMessage) {})
client.OnNewUserstateMessage(func(user User, message UserStateMessage) {})
client.OnNewNoticeMessage(func(message NoticeMessage) {})
client.OnUserJoin(func(channel, user string) {})
client.OnUserPart(func(channel, user string) {})
```

### Message Types

If you ever need more than basic PRIVMSG, this might be for you.
These are the 7 major message types currently supported:

	WHISPER
	PRIVMSG
	CLEARCHAT
	ROOMSTATE
	USERNOTICE
	USERSTATE
    NOTICE
