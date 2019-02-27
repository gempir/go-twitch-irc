package twitch

import (
	"strconv"
	"strings"
	"time"
)

// MessageType different message types possible to receive via IRC
type MessageType int

const (
	// UNSET is the default message type, for whenever a new message type is added by twitch that we don't parse yet
	UNSET MessageType = -1
	// WHISPER private messages
	WHISPER MessageType = 0
	// PRIVMSG standard chat message
	PRIVMSG MessageType = 1
	// CLEARCHAT timeout messages
	CLEARCHAT MessageType = 2
	// ROOMSTATE changes like sub mode
	ROOMSTATE MessageType = 3
	// USERNOTICE messages like subs, resubs, raids, etc
	USERNOTICE MessageType = 4
	// USERSTATE messages
	USERSTATE MessageType = 5
	// NOTICE messages like sub mode, host on
	NOTICE MessageType = 6
)

type channelMessage struct {
	RawMessage
	Channel string
}

type roomMessage struct {
	channelMessage
	RoomID string
}

// Unsure of a better name, but this isn't entirely descriptive of the contents
type chatMessage struct {
	roomMessage
	ID   string
	Time time.Time
}

type userMessage struct {
	Action bool
	Emotes []*Emote
}

// Emote twitch emotes
type Emote struct {
	Name  string
	ID    string
	Count int
}

// message is purely for internal use
type message struct {
	RawMessage RawMessage
	Channel    string
	Username   string
}

func parseMessage(line string) *message {
	if !strings.HasPrefix(line, "@") {
		return &message{
			RawMessage: RawMessage{
				Type:    UNSET,
				Raw:     line,
				Message: line,
			},
		}
	}

	split := strings.SplitN(line, " :", 3)
	if len(split) < 3 {
		for i := 0; i < 3-len(split); i++ {
			split = append(split, "")
		}
	}

	rawType, channel, username := parseMiddle(split[1])

	rawMessage := RawMessage{
		Type:    parseMessageType(rawType),
		RawType: rawType,
		Raw:     line,
		Tags:    parseTags(split[0]),
		Message: split[2],
	}

	return &message{
		Channel:    channel,
		Username:   username,
		RawMessage: rawMessage,
	}
}

func parseMiddle(middle string) (string, string, string) {
	var rawType, channel, username string

	for i, v := range strings.SplitN(middle, " ", 3) {
		switch {
		case i == 1:
			rawType = v
		case strings.Contains(v, "!"):
			username = strings.SplitN(v, "!", 2)[0]
		case strings.Contains(v, "#"):
			channel = strings.TrimPrefix(v, "#")
		}
	}

	return rawType, channel, username
}

func parseMessageType(messageType string) MessageType {
	switch messageType {
	case "PRIVMSG":
		return PRIVMSG
	case "WHISPER":
		return WHISPER
	case "CLEARCHAT":
		return CLEARCHAT
	case "NOTICE":
		return NOTICE
	case "ROOMSTATE":
		return ROOMSTATE
	case "USERSTATE":
		return USERSTATE
	case "USERNOTICE":
		return USERNOTICE
	default:
		return UNSET
	}
}

func parseTags(tagsRaw string) map[string]string {
	tags := make(map[string]string)

	tagsRaw = strings.TrimPrefix(tagsRaw, "@")
	for _, v := range strings.Split(tagsRaw, ";") {
		tag := strings.SplitN(v, "=", 2)

		var value string
		if len(tag) > 1 {
			value = tag[1]
		}

		tags[tag[0]] = value
	}
	return tags
}

func (m *message) parsePRIVMSGMessage() (*User, *PRIVMSGMessage) {
	privateMessage := PRIVMSGMessage{
		chatMessage: *m.parseChatMessage(),
		userMessage: *m.parseUserMessage(),
	}

	if privateMessage.Action {
		privateMessage.Message = privateMessage.Message[8 : len(privateMessage.Message)-1]
	}

	rawBits, ok := m.RawMessage.Tags["bits"]
	if !ok {
		return m.parseUser(), &privateMessage
	}

	bits, _ := strconv.Atoi(rawBits)
	privateMessage.Bits = bits
	return m.parseUser(), &privateMessage
}

func (m *message) parseCLEARCHATMessage() *CLEARCHATMessage {
	clearchatMessage := CLEARCHATMessage{
		chatMessage:  *m.parseChatMessage(),
		TargetUserID: m.RawMessage.Tags["target-user-id"],
	}

	clearchatMessage.TargetUsername = clearchatMessage.Message
	clearchatMessage.Message = ""

	rawBanDuration, ok := m.RawMessage.Tags["ban-duration"]
	if !ok {
		return &clearchatMessage
	}

	banDuration, _ := strconv.Atoi(rawBanDuration)
	clearchatMessage.BanDuration = banDuration
	return &clearchatMessage
}

func (m *message) parseUser() *User {
	user := User{
		ID:          m.RawMessage.Tags["user-id"],
		Name:        m.Username,
		DisplayName: m.RawMessage.Tags["display-name"],
		Color:       m.RawMessage.Tags["color"],
		Badges:      m.parseBadges(),
	}

	// USERSTATE doesn't contain a Username, but it does have a display-name tag.
	if user.Name == "" {
		user.Name = strings.ToLower(user.DisplayName)
	}

	return &user
}
func (m *message) parseBadges() map[string]int {
	badges := make(map[string]int)

	rawBadges, ok := m.RawMessage.Tags["badges"]
	if !ok {
		return badges
	}

	for _, v := range strings.Split(rawBadges, ",") {
		badge := strings.SplitN(v, "/", 2)
		if len(badge) < 2 {
			continue
		}

		badges[badge[0]], _ = strconv.Atoi(badge[1])
	}

	return badges
}

func (m *message) parseChatMessage() *chatMessage {
	chatMessage := chatMessage{
		roomMessage: *m.parseRoomMessage(),
		ID:          m.RawMessage.Tags["id"],
	}

	i, err := strconv.ParseInt(m.RawMessage.Tags["tmi-sent-ts"], 10, 64)
	if err != nil {
		return &chatMessage
	}

	chatMessage.Time = time.Unix(0, int64(i*1e6))
	return &chatMessage
}

func (m *message) parseRoomMessage() *roomMessage {
	return &roomMessage{
		channelMessage: *m.parseChannelMessage(),
		RoomID:         m.RawMessage.Tags["room-id"],
	}
}

func (m *message) parseChannelMessage() *channelMessage {
	return &channelMessage{
		RawMessage: m.RawMessage,
		Channel:    m.Channel,
	}
}

func (m *message) parseUserMessage() *userMessage {
	userMessage := userMessage{
		Emotes: m.parseEmotes(),
	}

	text := m.RawMessage.Message
	if strings.HasPrefix(text, "\u0001ACTION") && strings.HasSuffix(text, "\u0001") {
		userMessage.Action = true
	}

	return &userMessage
}

func (m *message) parseEmotes() []*Emote {
	var emotes []*Emote

	rawEmotes := m.RawMessage.Tags["emotes"]
	if rawEmotes == "" {
		return emotes
	}

	runes := []rune(m.RawMessage.Message)

	for _, v := range strings.Split(rawEmotes, "/") {
		split := strings.SplitN(v, ":", 2)
		pos := strings.SplitN(split[1], ",", 2)
		indexPair := strings.SplitN(pos[0], "-", 2)
		firstIndex, _ := strconv.Atoi(indexPair[0])
		lastIndex, _ := strconv.Atoi(indexPair[1])

		e := &Emote{
			Name:  string(runes[firstIndex:lastIndex]),
			ID:    split[0],
			Count: strings.Count(split[1], ",") + 1,
		}

		emotes = append(emotes, e)
	}

	return emotes
}

func parseJoinPart(text string) (string, string) {
	username := strings.Split(text, "!")
	channel := strings.Split(username[1], "#")
	return strings.Trim(channel[1], " "), strings.Trim(username[0], " :")
}

func parseNames(text string) (string, []string) {
	lines := strings.Split(text, ":")
	channelDirty := strings.Split(lines[1], "#")
	channel := strings.Trim(channelDirty[1], " ")
	users := strings.Split(lines[2], " ")

	return channel, users
}
