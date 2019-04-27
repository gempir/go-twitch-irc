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
	User User

	Raw       string
	Type      MessageType
	RawType   string
	Tags      map[string]string
	Message   string
	Target    string
	MessageID string
	ThreadID  string
	Emotes    []*Emote
	Action    bool
}

type PrivateMessage struct {
	User User

	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
	Channel string
	RoomID  string
	ID      string
	Time    time.Time
	Emotes  []*Emote
	Bits    int
	Action  bool
}

type ClearChatMessage struct {
	Raw            string
	Type           MessageType
	RawType        string
	Tags           map[string]string
	Message        string
	Channel        string
	RoomID         string
	Time           time.Time
	BanDuration    int
	TargetUserID   string
	TargetUsername string
}

type RoomStateMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
	Channel string
	RoomID  string
	State   map[string]int
}

type UserNoticeMessage struct {
	User User

	Raw       string
	Type      MessageType
	RawType   string
	Tags      map[string]string
	Message   string
	Channel   string
	RoomID    string
	ID        string
	Time      time.Time
	Emotes    []*Emote
	MsgID     string
	MsgParams map[string]string
	SystemMsg string
}

type UserStateMessage struct {
	User User

	Raw       string
	Type      MessageType
	RawType   string
	Tags      map[string]string
	Message   string
	Channel   string
	EmoteSets []string
}

type NoticeMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
	Channel string
	MsgID   string
}

type UserJoinMessage struct {
	// Channel name
	Channel string

	// User name
	User string
}

type UserPartMessage struct {
	// Channel name
	Channel string

	// User name
	User string
}
```

For unsupported message types, we return RawMessage:
```go
type RawMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
}
```

### Available Methods

ParseMessage parses a raw Twitch IRC message into a User and a message object. User can be nil.

	func ParseMessage(line string) (*User, interface{})

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
client.OnPrivateMessage(func(message PrivateMessage) {})
client.OnWhisperMessage(func(message WhisperMessage) {})
client.OnClearChatMessage(func(message ClearChatMessage) {})
client.OnRoomStateMessage(func(message RoomStateMessage) {})
client.OnUsernoticeMessage(func(message UserNoticeMessage) {})
client.OnUserstateMessage(func(message UserStateMessage) {})
client.OnNoticeMessage(func(message NoticeMessage) {})
client.OnUserJoinMessage(func(message UserJoinMessage) {})
client.OnUserPartMessage(func(message UserPartMessage) {})
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
    JOIN
    PART
