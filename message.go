package twitch

import (
	"strconv"
	"strings"
	"time"
)

// ADDING A NEW MESSAGE TYPE:
// 1. Add the message type at the bottom of the MessageType "const enum", with a unique index
// 2. Add a function at the bottom of file in the "parseXXXMessage" format, that parses your message type and returns a Message
// 3. Register the message into the map in the init function, specifying the message type value and the parser you just made

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
	// JOIN whenever a user joins a channel
	JOIN MessageType = 7
	// PART whenever a user parts from a channel
	PART MessageType = 8
	// RECONNECT is sent from Twitch when they request the client to reconnect (i.e. for an irc server restart)
	// https://dev.twitch.tv/docs/irc/commands/#reconnect-twitch-commands
	RECONNECT MessageType = 9
	// NAMES (or 353 https://www.alien.net.au/irc/irc2numerics.html#353) is the response sent from the server when
	// the client requests a list of names for a channel
	NAMES MessageType = 10
	// PING is a message that can be sent from the IRC server. go-twitch-irc responds to PINGs automatically
	PING MessageType = 11
	// PONG is a message that should be sent from the IRC server as a response to us sending a PING message.
	PONG MessageType = 12
	// CLEARMSG whenever a single message is deleted
	CLEARMSG MessageType = 13
	// GLOBALUSERSTATE On successful login, provides data about the current logged-in user through IRC tags
	GLOBALUSERSTATE MessageType = 14
)

type messageTypeDescription struct {
	Type   MessageType
	Parser func(*ircMessage) Message
}

var messageTypeMap map[string]messageTypeDescription

func init() {
	messageTypeMap = map[string]messageTypeDescription{
		"WHISPER":         {WHISPER, parseWhisperMessage},
		"PRIVMSG":         {PRIVMSG, parsePrivateMessage},
		"CLEARCHAT":       {CLEARCHAT, parseClearChatMessage},
		"ROOMSTATE":       {ROOMSTATE, parseRoomStateMessage},
		"USERNOTICE":      {USERNOTICE, parseUserNoticeMessage},
		"USERSTATE":       {USERSTATE, parseUserStateMessage},
		"NOTICE":          {NOTICE, parseNoticeMessage},
		"JOIN":            {JOIN, parseUserJoinMessage},
		"PART":            {PART, parseUserPartMessage},
		"RECONNECT":       {RECONNECT, parseReconnectMessage},
		"353":             {NAMES, parseNamesMessage},
		"PING":            {PING, parsePingMessage},
		"PONG":            {PONG, parsePongMessage},
		"CLEARMSG":        {CLEARMSG, parseClearMessage},
		"GLOBALUSERSTATE": {GLOBALUSERSTATE, parseGlobalUserStateMessage},
	}
}

// EmotePosition is a single position of an emote to be used for text replacement.
type EmotePosition struct {
	Start int
	End   int
}

// Emote twitch emotes
type Emote struct {
	Name      string
	ID        string
	Count     int
	Positions []EmotePosition
}

// ParseMessage parse a raw Twitch IRC message
func ParseMessage(line string) Message {
	// Uncomment this and recoverMessage if debugging a message that crashes the parser
	// defer recoverMessage(line)

	ircMessage, err := parseIRCMessage(line)
	if err != nil {
		return parseRawMessage(ircMessage)
	}

	if mt, ok := messageTypeMap[ircMessage.Command]; ok {
		return mt.Parser(ircMessage)
	}

	return parseRawMessage(ircMessage)
}

// func recoverMessage(line string) {
// 	if err := recover(); err != nil {
// 		log.Println(line)
// 		log.Println(err)
// 		log.Println(string(debug.Stack()))
// 	}
// }

// parseMessageType parses a message type from an irc COMMAND string
func parseMessageType(messageType string) MessageType {
	if mt, ok := messageTypeMap[messageType]; ok {
		return mt.Type
	}

	return UNSET
}

func parseUser(message *ircMessage) User {
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

	return user
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

func parseWhisperMessage(message *ircMessage) Message {
	whisperMessage := WhisperMessage{
		User: parseUser(message),

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

	if strings.Contains(whisperMessage.Message, "/me") {
		whisperMessage.Message = strings.TrimPrefix(whisperMessage.Message, "/me")
		whisperMessage.Action = true
	}

	whisperMessage.Emotes = parseEmotes(message.Tags["emotes"], whisperMessage.Message)

	return &whisperMessage
}

func parsePrivateMessage(message *ircMessage) Message {
	privateMessage := PrivateMessage{
		User: parseUser(message),

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

	rawBits, ok := message.Tags["bits"]
	if ok {
		bits, _ := strconv.Atoi(rawBits)
		privateMessage.Bits = bits
	}

	text := privateMessage.Message
	if strings.HasPrefix(text, "\u0001ACTION") && strings.HasSuffix(text, "\u0001") {
		privateMessage.Action = true
		if len(text) == 8 {
			privateMessage.Message = ""
		} else {
			privateMessage.Message = text[8 : len(text)-1]
		}
	}

	privateMessage.Emotes = parseEmotes(message.Tags["emotes"], privateMessage.Message)

	firstMessage, ok := message.Tags["first-msg"]

	if ok {
		privateMessage.FirstMessage = firstMessage == "1"
	}

	return &privateMessage
}

func parseClearChatMessage(message *ircMessage) Message {
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

func parseClearMessage(message *ircMessage) Message {
	clearMessage := ClearMessage{
		Raw:         message.Raw,
		Type:        parseMessageType(message.Command),
		RawType:     message.Command,
		Tags:        message.Tags,
		Login:       message.Tags["login"],
		TargetMsgID: message.Tags["target-msg-id"],
	}

	if len(message.Params) == 2 {
		clearMessage.Message = message.Params[1]
	}

	clearMessage.Channel = strings.TrimPrefix(message.Params[0], "#")

	return &clearMessage
}

func parseRoomStateMessage(message *ircMessage) Message {
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

func parseGlobalUserStateMessage(message *ircMessage) Message {
	globalUserStateMessage := GlobalUserStateMessage{
		Raw:       message.Raw,
		Type:      parseMessageType(message.Command),
		RawType:   message.Command,
		Tags:      message.Tags,
		User:      parseUser(message),
		EmoteSets: parseEmoteSets(message),
	}

	return &globalUserStateMessage
}

func parseUserNoticeMessage(message *ircMessage) Message {
	userNoticeMessage := UserNoticeMessage{
		User: parseUser(message),

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

func parseUserStateMessage(message *ircMessage) Message {
	userStateMessage := UserStateMessage{
		User: parseUser(message),

		Raw:       message.Raw,
		Type:      parseMessageType(message.Command),
		RawType:   message.Command,
		Tags:      message.Tags,
		Channel:   strings.TrimPrefix(message.Params[0], "#"),
		EmoteSets: parseEmoteSets(message),
	}

	return &userStateMessage
}

func parseNoticeMessage(message *ircMessage) Message {
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

func parseUserJoinMessage(message *ircMessage) Message {
	parsedMessage := UserJoinMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,

		User: message.Source.Username,
	}

	if len(message.Params) == 1 {
		parsedMessage.Channel = strings.TrimPrefix(message.Params[0], "#")
	}

	return &parsedMessage
}

func parseUserPartMessage(message *ircMessage) Message {
	parsedMessage := UserPartMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,

		User: message.Source.Username,
	}

	if len(message.Params) == 1 {
		parsedMessage.Channel = strings.TrimPrefix(message.Params[0], "#")
	}

	return &parsedMessage
}

func parseReconnectMessage(message *ircMessage) Message {
	return &ReconnectMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
	}
}

func parseNamesMessage(message *ircMessage) Message {
	parsedMessage := NamesMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
	}

	if len(message.Params) == 4 {
		parsedMessage.Channel = strings.TrimPrefix(message.Params[2], "#")
		parsedMessage.Users = strings.Split(message.Params[3], " ")
	}

	return &parsedMessage
}

func parsePingMessage(message *ircMessage) Message {
	parsedMessage := PingMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
	}

	if len(message.Params) == 1 {
		parsedMessage.Message = strings.Split(message.Params[0], " ")[0]
	}

	return &parsedMessage
}

func parsePongMessage(message *ircMessage) Message {
	parsedMessage := PongMessage{
		Raw:     message.Raw,
		Type:    parseMessageType(message.Command),
		RawType: message.Command,
	}

	if len(message.Params) == 2 {
		parsedMessage.Message = strings.Split(message.Params[1], " ")[0]
	}

	return &parsedMessage
}

func parseTime(rawTime string) time.Time {
	if rawTime == "" {
		return time.Time{}
	}

	time64, _ := strconv.ParseInt(rawTime, 10, 64)
	return time.Unix(0, time64*1e6)
}

func parseEmotes(rawEmotes, message string) []*Emote {
	var emotes []*Emote

	if rawEmotes == "" {
		return emotes
	}

	runes := []rune(message)

L:
	for _, v := range strings.Split(rawEmotes, "/") {
		split := strings.SplitN(v, ":", 2)
		if len(split) != 2 {
			// We have received bad emote data :(
			continue
		}
		pairs := strings.Split(split[1], ",")
		if len(pairs) < 1 {
			// We have received bad emote data :(
			continue
		}
		pair := strings.SplitN(pairs[0], "-", 2)
		if len(pair) != 2 {
			// We have received bad emote data :(
			continue
		}

		firstIndex, _ := strconv.Atoi(pair[0])
		lastIndex, _ := strconv.Atoi(pair[1])

		if lastIndex+1 > len(runes) {
			lastIndex = len(runes) - 1
		}

		if firstIndex+1 > len(runes) {
			firstIndex = len(runes) - 1
		}

		var positions []EmotePosition
		for _, p := range pairs {
			pos := strings.SplitN(p, "-", 2)
			if len(pos) != 2 {
				// Position is not valid, continue on the outer loop and skip this emote.
				continue L
			}

			// Convert the start and end positions from strings, and bail if it fails.
			start, err := strconv.Atoi(pos[0])
			if err != nil {
				continue L
			}

			end, err := strconv.Atoi(pos[1])
			if err != nil {
				continue L
			}

			positions = append(positions, EmotePosition{
				Start: start,
				End:   end,
			})
		}

		emote := &Emote{
			Name:      string(runes[firstIndex : lastIndex+1]),
			ID:        split[0],
			Count:     strings.Count(split[1], ",") + 1,
			Positions: positions,
		}

		emotes = append(emotes, emote)
	}

	return emotes
}

func parseEmoteSets(message *ircMessage) []string {
	_, ok := message.Tags["emote-sets"]
	if !ok {
		return []string{}
	}

	return strings.Split(message.Tags["emote-sets"], ",")
}
