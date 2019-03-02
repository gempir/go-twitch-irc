package twitch

import (
	"strconv"
	"strings"
	"time"
)

// MessageType different message types possible to receive via IRC
type MessageType int

const (
	// ERROR is for messages that didn't parse properly
	ERROR MessageType = -2
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

// Unsure of a better name, but this isn't entirely descriptive of the contents
type chatMessage struct {
	roomMessage
	ID   string // Not in CLEARCHAT
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
	RawMessage rawMessage
	Channel    string
	Username   string
}

func parseMessage(line string) *message {
	if !strings.HasPrefix(line, "@") {
		return &message{
			RawMessage: rawMessage{
				Type:    UNSET,
				Raw:     line,
				Message: line,
			},
		}
	}

	split := strings.SplitN(line, " :", 3)

	// Sometimes Twitch can fail to send the full value
	// @badges=;color=;display-name=ZZZi;emotes=;flags=;id=75bb6b6b-e36c-49af-a293-16024738ab92;mod=0;room-id=36029255;subscriber=0;tmi-sent-ts=1551476573570;turbo
	if len(split) == 1 {
		return &message{
			RawMessage: rawMessage{
				Type:    ERROR,
				Raw:     line,
				Message: line,
			},
		}
	}

	// If there is only two values, message is empty. Fill it with a blank string so we can assign it to rawMessage.
	if len(split) == 2 {
		for i := 0; i < 3-len(split); i++ {
			split = append(split, "")
		}
	}

	rawType, channel, username := parseMiddle(split[1])

	rawMessage := rawMessage{
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

func (m *message) parseWhisperMessage() (*User, *WhisperMessage) {
	whisperMessage := WhisperMessage{
		rawMessage:  m.RawMessage,
		userMessage: *m.parseUserMessage(),
	}

	if whisperMessage.Action {
		// Remove "\u0001ACTION" from the beginning and "\u0001" from the end
		whisperMessage.Message = whisperMessage.Message[8 : len(whisperMessage.Message)-1]
	}

	return m.parseUser(), &whisperMessage
}

func (m *message) parsePrivateMessage() (*User, *PrivateMessage) {
	privateMessage := PrivateMessage{
		chatMessage: *m.parseChatMessage(),
		userMessage: *m.parseUserMessage(),
	}

	if privateMessage.Action {
		// Remove "\u0001ACTION" from the beginning and "\u0001" from the end
		privateMessage.Message = privateMessage.Message[8 : len(privateMessage.Message)-1]
	}

	rawBits, ok := m.RawMessage.Tags["bits"]
	if ok {
		bits, _ := strconv.Atoi(rawBits)
		privateMessage.Bits = bits
	}

	return m.parseUser(), &privateMessage
}

func (m *message) parseClearChatMessage() *ClearChatMessage {
	clearchatMessage := ClearChatMessage{
		chatMessage:  *m.parseChatMessage(),
		MsgID:        "Clear",
		TargetUserID: m.RawMessage.Tags["target-user-id"],
	}

	clearchatMessage.TargetUsername = clearchatMessage.Message
	clearchatMessage.Message = ""

	rawBanDuration, ok := m.RawMessage.Tags["ban-duration"]
	if ok {
		banDuration, _ := strconv.Atoi(rawBanDuration)
		clearchatMessage.BanDuration = banDuration
	}

	if clearchatMessage.TargetUsername != "" {
		clearchatMessage.MsgID = "Ban"
	}

	if clearchatMessage.BanDuration != 0 {
		clearchatMessage.MsgID = "Timeout"
	}

	return &clearchatMessage
}

func (m *message) parseRoomStateMessage() *RoomStateMessage {
	roomstateMessage := RoomStateMessage{
		roomMessage: *m.parseRoomMessage(),
		Language:    m.RawMessage.Tags["broadcaster-lang"],
		State:       make(map[string]int),
	}

	roomstateMessage.addState("emote-only")
	roomstateMessage.addState("followers-only")
	roomstateMessage.addState("r9k")
	roomstateMessage.addState("rituals")
	roomstateMessage.addState("slow")
	roomstateMessage.addState("subs-only")

	return &roomstateMessage
}

func (m *RoomStateMessage) addState(tag string) {
	rawValue, ok := m.Tags[tag]
	if !ok {
		return
	}

	value, _ := strconv.Atoi(rawValue)
	m.State[tag] = value
}

func (m *message) parseUserNoticeMessage() (*User, *UserNoticeMessage) {
	usernoticeMessage := UserNoticeMessage{
		chatMessage: *m.parseChatMessage(),
		userMessage: *m.parseUserMessage(),
		MsgID:       m.RawMessage.Tags["msg-id"],
	}

	usernoticeMessage.parseMsgParams()

	rawSystemMsg, ok := usernoticeMessage.Tags["system-msg"]
	if ok {
		rawSystemMsg = strings.ReplaceAll(rawSystemMsg, "\\s", " ")
		rawSystemMsg = strings.ReplaceAll(rawSystemMsg, "\\n", "")
		usernoticeMessage.SystemMsg = strings.TrimSpace(rawSystemMsg)
	}

	return m.parseUser(), &usernoticeMessage
}

func (m *UserNoticeMessage) parseMsgParams() {
	m.MsgParams = make(map[string]interface{})

	for tag, value := range m.Tags {
		if strings.Contains(tag, "msg-param") {
			m.MsgParams[tag] = strings.ReplaceAll(value, "\\s", " ")
		}
	}

	m.paramToInt("msg-param-bits-amount")
	m.paramToInt("msg-param-cumulative-months")
	m.paramToInt("msg-param-mass-gift-count")
	m.paramToInt("msg-param-months")
	m.paramToInt("msg-param-selected-count")
	m.paramToInt("msg-param-sender-count")
	m.paramToBool("msg-param-should-share-streak")
	m.paramToInt("msg-param-streak-months")
	m.paramToInt("msg-param-threshold")
	m.paramToInt("msg-param-viewerCount")
}

func (m *UserNoticeMessage) paramToBool(tag string) {
	rawValue, ok := m.MsgParams[tag]
	if !ok {
		return
	}

	m.MsgParams[tag] = rawValue.(string) == "1"
}

func (m *UserNoticeMessage) paramToInt(tag string) {
	rawValue, ok := m.MsgParams[tag]
	if !ok {
		return
	}

	m.MsgParams[tag], _ = strconv.Atoi(rawValue.(string))
}

func (m *message) parseUserStateMessage() (*User, *UserStateMessage) {
	userstateMessage := UserStateMessage{
		channelMessage: *m.parseChannelMessage(),
	}

	rawEmoteSets, ok := userstateMessage.Tags["emote-sets"]
	if ok {
		userstateMessage.EmoteSets = strings.Split(rawEmoteSets, ",")
	}

	return m.parseUser(), &userstateMessage
}

func (m *message) parseNoticeMessage() *NoticeMessage {
	return &NoticeMessage{
		channelMessage: *m.parseChannelMessage(),
		MsgID:          m.RawMessage.Tags["msg-id"],
	}
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
	if ok {
		for _, v := range strings.Split(rawBadges, ",") {
			badge := strings.SplitN(v, "/", 2)
			if len(badge) < 2 {
				continue
			}

			badges[badge[0]], _ = strconv.Atoi(badge[1])
		}
	}

	return badges
}

func (m *message) parseChatMessage() *chatMessage {
	chatMessage := chatMessage{
		roomMessage: *m.parseRoomMessage(),
		ID:          m.RawMessage.Tags["id"],
	}

	rawTime, err := strconv.ParseInt(m.RawMessage.Tags["tmi-sent-ts"], 10, 64)
	if err != nil {
		return &chatMessage
	}

	chatMessage.Time = time.Unix(0, int64(rawTime*1e6))
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
		rawMessage: m.RawMessage,
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
			Name:  string(runes[firstIndex : lastIndex+1]),
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
