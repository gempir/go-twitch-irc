package twitch

import (
	"strconv"
	"strings"
	"time"
)

// MessageType different message types possible to receive via IRC
type MessageType int

const (
	// UNSET is for message types we currently don't support
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

// Emote twitch emotes
type Emote struct {
	Name  string
	ID    string
	Count int
}

func parseMessage(line string) (*User, interface{}) {
	ircMessage, err := parseIRCMessage(line)
	if err != nil {
		return nil, parseRawMessage(ircMessage)
	}

	switch ircMessage.Command {
	case "WHISPER":
		return parseUser(ircMessage), parseWhisperMessage(ircMessage)
	case "PRIVMSG":
		return parseUser(ircMessage), parsePrivateMessage(ircMessage)
	case "CLEARCHAT":
		return nil, parseClearChatMessage(ircMessage)
	case "ROOMSTATE":
		return nil, parseRoomStateMessage(ircMessage)
	case "USERNOTICE":
		return parseUser(ircMessage), parseUserNoticeMessage(ircMessage)
	case "USERSTATE":
		return parseUser(ircMessage), parseUserStateMessage(ircMessage)
	case "NOTICE":
		return nil, parseNoticeMessage(ircMessage)
	default:
		return nil, parseRawMessage(ircMessage)
	}
}

func parseMessageType(messageType string) MessageType {
	switch messageType {
	case "WHISPER":
		return WHISPER
	case "PRIVMSG":
		return PRIVMSG
	case "CLEARCHAT":
		return CLEARCHAT
	case "ROOMSTATE":
		return ROOMSTATE
	case "USERNOTICE":
		return USERNOTICE
	case "USERSTATE":
		return USERSTATE
	case "NOTICE":
		return NOTICE
	default:
		return UNSET
	}
}

func parseUser(message *ircMessage) *User {
	user := User{
		ID:          message.Tags["user-id"],
		Name:        message.Source.Username,
		DisplayName: message.Tags["display-name"],
		Color:       message.Tags["color"],
		Badges:      make(map[string]int),
	}

	if rawBadges := message.Tags["badges"]; rawBadges != "" {
		user.Badges = parseBadges(rawBadges)
	}

	// USERSTATE doesn't contain a Username, but it does have a display-name tag
	if user.Name == "" && user.DisplayName != "" {
		user.Name = strings.ToLower(user.DisplayName)
		user.Name = strings.Replace(user.Name, " ", "", 1)
	}

	return &user
}

func parseBadges(rawBadges string) map[string]int {
	badges := make(map[string]int)

	for _, badge := range strings.Split(rawBadges, ",") {
		pair := strings.SplitN(badge, "/", 2)
		badges[pair[0]], _ = strconv.Atoi(pair[1])
	}

	return badges
}

func parseRawMessage(message *ircMessage) *RawMessage {
	rawMessage := RawMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
		Tags:    message.Tags,
	}

	for i, v := range message.Params {
		if !strings.Contains(v, "#") {
			rawMessage.Message = strings.Join(message.Params[i:], " ")
			break
		}
	}

	return &rawMessage
}

func parseWhisperMessage(message *ircMessage) *WhisperMessage {
	whisperMessage := WhisperMessage{
		Raw:       message.Raw,
		Type:      parseMessageType(message.Command),
		RawType:   message.Command,
		Tags:      message.Tags,
		MessageID: message.Tags["message-id"],
		ThreadID:  message.Tags["thread-id"],
	}

	if len(message.Params) == 2 {
		whisperMessage.Message = message.Params[1]
	}

	whisperMessage.Target = message.Params[0]
	whisperMessage.Emotes = parseEmotes(message.Tags["emotes"], whisperMessage.Message)

	if strings.Contains(whisperMessage.Message, "/me") {
		whisperMessage.Message = strings.TrimPrefix(whisperMessage.Message, "/me")
		whisperMessage.Action = true
	}

	return &whisperMessage
}

func parsePrivateMessage(message *ircMessage) *PrivateMessage {
	privateMessage := PrivateMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
		Tags:    message.Tags,
		RoomID:  message.Tags["room-id"],
		ID:      message.Tags["id"],
		Time:    parseTime(message.Tags["tmi-sent-ts"]),
	}

	if len(message.Params) == 2 {
		privateMessage.Message = message.Params[1]
	}

	privateMessage.Channel = strings.TrimPrefix(message.Params[0], "#")
	privateMessage.Emotes = parseEmotes(message.Tags["emotes"], privateMessage.Message)

	rawBits, ok := message.Tags["bits"]
	if ok {
		bits, _ := strconv.Atoi(rawBits)
		privateMessage.Bits = bits
	}

	text := privateMessage.Message
	if strings.HasPrefix(text, "\u0001ACTION") && strings.HasSuffix(text, "\u0001") {
		privateMessage.Message = text[8 : len(text)-1]
		privateMessage.Action = true
	}

	return &privateMessage
}

func parseClearChatMessage(message *ircMessage) *ClearChatMessage {
	clearChatMessage := ClearChatMessage{
		Raw:          message.Raw,
		Type:         parseMessageType(message.Command),
		RawType:      message.Command,
		Tags:         message.Tags,
		RoomID:       message.Tags["room-id"],
		Time:         parseTime(message.Tags["tmi-sent-ts"]),
		TargetUserID: message.Tags["target-user-id"],
	}

	clearChatMessage.Channel = strings.TrimPrefix(message.Params[0], "#")

	rawBanDuration, ok := message.Tags["ban-duration"]
	if ok {
		duration, _ := strconv.Atoi(rawBanDuration)
		clearChatMessage.BanDuration = duration
	}

	if len(message.Params) == 2 {
		clearChatMessage.TargetUsername = message.Params[1]
	}

	return &clearChatMessage
}

func parseRoomStateMessage(message *ircMessage) *RoomStateMessage {
	roomStateMessage := RoomStateMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
		Tags:    message.Tags,
		RoomID:  message.Tags["room-id"],
		State:   make(map[string]int),
	}

	roomStateMessage.Channel = strings.TrimPrefix(message.Params[0], "#")

	stateTags := []string{"emote-only", "followers-only", "r9k", "rituals", "slow", "subs-only"}
	for _, tag := range stateTags {
		rawValue, ok := message.Tags[tag]
		if !ok {
			continue
		}

		value, _ := strconv.Atoi(rawValue)
		roomStateMessage.State[tag] = value
	}

	return &roomStateMessage
}

func parseUserNoticeMessage(message *ircMessage) *UserNoticeMessage {
	userNoticeMessage := UserNoticeMessage{
		Raw:       message.Raw,
		Type:      parseMessageType(message.Command),
		RawType:   message.Command,
		Tags:      message.Tags,
		RoomID:    message.Tags["room-id"],
		ID:        message.Tags["id"],
		Time:      parseTime(message.Tags["tmi-sent-ts"]),
		MsgID:     message.Tags["msg-id"],
		MsgParams: make(map[string]string),
		SystemMsg: message.Tags["system-msg"],
	}

	if len(message.Params) == 2 {
		userNoticeMessage.Message = message.Params[1]
	}

	userNoticeMessage.Channel = strings.TrimPrefix(message.Params[0], "#")
	userNoticeMessage.Emotes = parseEmotes(message.Tags["emotes"], userNoticeMessage.Message)

	for tag, value := range message.Tags {
		if strings.Contains(tag, "msg-param") {
			userNoticeMessage.MsgParams[tag] = value
		}
	}

	return &userNoticeMessage
}

func parseUserStateMessage(message *ircMessage) *UserStateMessage {
	userStateMessage := UserStateMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
		Tags:    message.Tags,
	}

	userStateMessage.Channel = strings.TrimPrefix(message.Params[0], "#")

	rawEmoteSets, ok := message.Tags["emote-sets"]
	if ok {
		userStateMessage.EmoteSets = strings.Split(rawEmoteSets, ",")
	}

	return &userStateMessage
}

func parseNoticeMessage(message *ircMessage) *NoticeMessage {
	noticeMessage := NoticeMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
		Tags:    message.Tags,
		MsgID:   message.Tags["msg-id"],
	}

	if len(message.Params) == 2 {
		noticeMessage.Message = message.Params[1]
	}

	noticeMessage.Channel = strings.TrimPrefix(message.Params[0], "#")

	return &noticeMessage
}

func parseTime(rawTime string) time.Time {
	if rawTime == "" {
		return time.Time{}
	}

	time64, _ := strconv.ParseInt(rawTime, 10, 64)
	return time.Unix(0, int64(time64*1e6))
}

func parseEmotes(rawEmotes, message string) []*Emote {
	var emotes []*Emote

	if rawEmotes == "" {
		return emotes
	}

	runes := []rune(message)

	for _, v := range strings.Split(rawEmotes, "/") {
		split := strings.SplitN(v, ":", 2)
		pairs := strings.SplitN(split[1], ",", 2)
		pair := strings.SplitN(pairs[0], "-", 2)

		firstIndex, _ := strconv.Atoi(pair[0])
		lastIndex, _ := strconv.Atoi(pair[1])

		emote := &Emote{
			Name:  string(runes[firstIndex : lastIndex+1]),
			ID:    split[0],
			Count: strings.Count(split[1], ",") + 1,
		}

		emotes = append(emotes, emote)
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
