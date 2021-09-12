# go-twitch-irc [![Coverage Status](https://coveralls.io/repos/github/gempir/go-twitch-irc/badge.svg?branch=master)](https://coveralls.io/github/gempir/go-twitch-irc?branch=master)

This is an irc client for connecting to twitch. It handles the annoying stuff like irc tag parsing.
I highly recommend reading the documentation below, but this readme gives a basic overview of the functionality.

Documentation: https://pkg.go.dev/github.com/gempir/go-twitch-irc/v2?tab=doc

## Getting Started
```go
package main

import (
	"fmt"

	"github.com/gempir/go-twitch-irc/v2"
)

func main() {
	// or client := twitch.NewAnonymousClient() for an anonymous user (no write capabilities)
	client := twitch.NewClient("yourtwitchusername", "oauth:123123123")

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
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

type ClearMessage struct {
	Raw         string
	Type        MessageType
	RawType     string
	Tags        map[string]string
	Message     string
	Channel     string
	Login       string
	TargetMsgID string
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

type GlobalUserStateMessage struct {
	User User

	Raw       string
	Type      MessageType
	RawType   string
	Tags      map[string]string
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

```go
func ParseMessage(line string) (*User, interface{})
```

### Client Methods

These are the available methods of the client so you can get your bot going:

```go
func (c *Client) Say(channel, text string)
func (c *Client) Whisper(username, text string)
func (c *Client) Join(channel string)
func (c *Client) Depart(channel string)
func (c *Client) Userlist(channel string) ([]string, error)
func (c *Client) Connect() error
func (c *Client) Disconnect() error
```

### Options

On your client you can configure multiple options:
```go
client.IrcAddress = "127.0.0.1:3030" // for custom irc server
client.TLS = false // enabled by default, will connect to non TLS server of twitch when off or the given client.IrcAddress
client.SetupCmd = "LOGIN custom_command_here" // Send a custom command on successful IRC connection, before authentication.
client.Capabilities = []string{twitch.TagsCapability, twitch.CommandsCapability} // Customize which capabilities are sent
client.SetRateLimiter(twitch.CreateVerifiedRateLimiter()) // If you have a verified bot or other needs use this to set a custom rate limiter
```

Option modifications must be done before calling Connect on the client.

#### Capabilities

By default, the client sends along all 3 Twitch capabilities ([Tags](https://dev.twitch.tv/docs/irc/tags), [Commands](https://dev.twitch.tv/docs/irc/commands), [Membership](https://dev.twitch.tv/docs/irc/membership)).  

### Callbacks

These callbacks are available to pass to the client:
```go
client.OnConnect(func() {})
client.OnPrivateMessage(func(message PrivateMessage) {})
client.OnWhisperMessage(func(message WhisperMessage) {})
client.OnClearChatMessage(func(message ClearChatMessage) {})
client.OnClearMessage(func(message ClearMessage) {})
client.OnRoomStateMessage(func(message RoomStateMessage) {})
client.OnUserNoticeMessage(func(message UserNoticeMessage) {})
client.OnUserStateMessage(func(message UserStateMessage) {})
client.OnGlobalUserStateMessage(func(message GlobalUserStateMessage) {})
client.OnNoticeMessage(func(message NoticeMessage) {})
client.OnUserJoinMessage(func(message UserJoinMessage) {})
client.OnUserPartMessage(func(message UserPartMessage) {})
```

### Message Types

If you ever need more than basic PRIVMSG, this might be for you.
These are the message types currently supported:

    WHISPER
    PRIVMSG
    CLEARCHAT
    CLEARMSG
    ROOMSTATE
    USERNOTICE
    USERSTATE
    GLOBALUSERSTATE
    NOTICE
    JOIN
    PART
