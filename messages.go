package twitch

import "time"

// RawMessage data you receive from TMI
type RawMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
}

// GetType implements the Message interface, and returns this message's type
func (msg *RawMessage) GetType() MessageType {
	return msg.Type
}

// WhisperMessage data you receive from WHISPER message type
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

// GetType implements the Message interface, and returns this message's type
func (msg *WhisperMessage) GetType() MessageType {
	return msg.Type
}

// PrivateMessage data you receive from PRIVMSG message type
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

// GetType implements the Message interface, and returns this message's type
func (msg *PrivateMessage) GetType() MessageType {
	return msg.Type
}

// ClearChatMessage data you receive from CLEARCHAT message type
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

// GetType implements the Message interface, and returns this message's type
func (msg *ClearChatMessage) GetType() MessageType {
	return msg.Type
}

// RoomStateMessage data you receive from ROOMSTATE message type
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

// GetType implements the Message interface, and returns this message's type
func (msg *RoomStateMessage) GetType() MessageType {
	return msg.Type
}

// UserNoticeMessage  data you receive from USERNOTICE message type
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

// GetType implements the Message interface, and returns this message's type
func (msg *UserNoticeMessage) GetType() MessageType {
	return msg.Type
}

// UserStateMessage data you receive from the USERSTATE message type
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

// GetType implements the Message interface, and returns this message's type
func (msg *UserStateMessage) GetType() MessageType {
	return msg.Type
}

// NoticeMessage data you receive from the NOTICE message type
type NoticeMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
	Channel string
	MsgID   string
}

// GetType implements the Message interface, and returns this message's type
func (msg *NoticeMessage) GetType() MessageType {
	return msg.Type
}

// UserJoinMessage desJoines the message that is sent whenever a user joins a channel we're connected to
// See https://dev.twitch.tv/docs/irc/membership/#join-twitch-membership
type UserJoinMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	// Channel name
	Channel string

	// User name
	User string
}

// GetType implements the Message interface, and returns this message's type
func (msg *UserJoinMessage) GetType() MessageType {
	return msg.Type
}

// UserPartMessage describes the message that is sent whenever a user leaves a channel we're connected to
// See https://dev.twitch.tv/docs/irc/membership/#part-twitch-membership
type UserPartMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	// Channel name
	Channel string

	// User name
	User string
}

// GetType implements the Message interface, and returns this message's type
func (msg *UserPartMessage) GetType() MessageType {
	return msg.Type
}

// ReconnectMessage describes the
type ReconnectMessage struct {
	Raw     string
	Type    MessageType
	RawType string
}

// GetType implements the Message interface, and returns this message's type
func (msg *ReconnectMessage) GetType() MessageType {
	return msg.Type
}

// NamesMessage describes the data posted in response to a /names command
// See https://www.alien.net.au/irc/irc2numerics.html#353
type NamesMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	// Channel name
	Channel string

	// List of user names
	Users []string
}

// GetType implements the Message interface, and returns this message's type
func (msg *NamesMessage) GetType() MessageType {
	return msg.Type
}

// PingMessage describes an IRC PING message
type PingMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	Message string
}

// GetType implements the Message interface, and returns this message's type
func (msg *PingMessage) GetType() MessageType {
	return msg.Type
}

// PongMessage describes an IRC PONG message
type PongMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	Message string
}

// GetType implements the Message interface, and returns this message's type
func (msg *PongMessage) GetType() MessageType {
	return msg.Type
}
