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

	client.OnNewMessage(func(channel string, user twitch.User, message twitch.Message) {
		fmt.Println(message.Text)
	})

	client.Join("gempir")

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
```
### Available Data

The 2 structs twitch.User and twitch.Message look like this:
```go
type User struct {
	UserID      string
	Username    string
	DisplayName string
	UserType    string
	Color       string
	Badges      map[string]int
}

type Message struct {
	Type      MessageType
	Time      time.Time
	Action    bool
	Emotes    []*Emote
	Tags      map[string]string
	Text      string
	Raw       string
	ChannelID string
}
```
Channel is just a string like "lirik", note the absent #.

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
```
### Callbacks

These callbacks are available to pass to the client:
```go
client.OnConnect(func() {})
client.OnNewWhisper(func(user twitch.User, message twitch.Message) {})
client.OnNewMessage(func(channel string, user twitch.User, message twitch.Message) {})
client.OnNewRoomstateMessage(func(channel string, user twitch.User, message twitch.Message) {})
client.OnNewClearchatMessage(func(channel string, user twitch.User, message twitch.Message) {})
client.OnNewUsernoticeMessage(func(channel string, user twitch.User, message twitch.Message) {})
client.OnNewNoticeMessage(func(channel string, user twitch.User, message twitch.Message) {})
client.OnNewUserstateMessage(func(channel string, user twitch.User, message twitch.Message) {})
client.OnUserJoin(func(channel, user string) {})
client.OnUserPart(func(channel, user string) {})
```
### Message Types

If you ever need more than basic PRIVMSG, this might be for you.
These are the 6 major message types currently supported:

	PRIVMSG
	WHISPER
	ROOMSTATE
	CLEARCHAT
	USERNOTICE
    NOTICE
