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

type chatMessage struct {
	roomMessage
	ID   string
	Time time.Time
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

	for _, v := range strings.SplitN(middle, " ", 3) {
		if strings.Contains(v, "!") {
			username = strings.SplitN(v, "!", 2)[0]
		} else if strings.Contains(v, "#") {
			channel = strings.TrimPrefix(v, "#")
		} else {
			rawType = v
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

func parseBadges(badges string) map[string]int {
	m := map[string]int{}
	spl := strings.Split(badges, ",")
	for _, badge := range spl {
		s := strings.SplitN(badge, "/", 2)
		if len(s) < 2 {
			continue
		}
		n, _ := strconv.Atoi(s[1])
		m[s[0]] = n
	}
	return m
}

func parseTwitchEmotes(emoteTag, text string) []*Emote {
	emotes := []*Emote{}

	if emoteTag == "" {
		return emotes
	}

	runes := []rune(text)

	emoteSlice := strings.Split(emoteTag, "/")
	for i := range emoteSlice {
		spl := strings.Split(emoteSlice[i], ":")
		pos := strings.Split(spl[1], ",")
		sp := strings.Split(pos[0], "-")
		start, _ := strconv.Atoi(sp[0])
		end, _ := strconv.Atoi(sp[1])
		id := spl[0]
		e := &Emote{
			ID:    id,
			Count: strings.Count(emoteSlice[i], "-"),
			Name:  string(runes[start : end+1]),
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
