package twitch

import (
	"fmt"
	"regexp"
	"strings"
)

// Maximum supported length of an irc message
const maxMessageLength = 510

type ircMessage struct {
	Raw     string
	Tags    map[string]string
	Source  ircMessageSource
	Command string
	Params  []string
}

type ircMessageSource struct {
	Nickname string
	Username string
	Host     string
}

func parseIRCMessage(line string) (*ircMessage, error) {
	message := ircMessage{
		Raw:    line,
		Tags:   make(map[string]string),
		Params: []string{},
	}

	split := strings.Split(line, " ")
	index := 0

	if strings.HasPrefix(split[index], "@") {
		message.Tags = parseIRCTags(split[index])
		index++
	}

	if index >= len(split) {
		return &message, fmt.Errorf("parseIRCMessage: partial message")
	}

	if strings.HasPrefix(split[index], ":") {
		message.Source = *parseIRCMessageSource(split[index])
		index++
	}

	if index >= len(split) {
		return &message, fmt.Errorf("parseIRCMessage: no command")
	}

	message.Command = split[index]
	index++

	if index >= len(split) {
		return &message, nil
	}

	var params []string
	for i, v := range split[index:] {
		if strings.HasPrefix(v, ":") {
			v = strings.Join(split[index+i:], " ")
			v = strings.TrimPrefix(v, ":")
			params = append(params, v)
			break
		}

		params = append(params, v)
	}

	message.Params = params

	return &message, nil
}

func parseIRCTags(rawTags string) map[string]string {
	tags := make(map[string]string)

	rawTags = strings.TrimPrefix(rawTags, "@")

	for _, tag := range strings.Split(rawTags, ";") {
		pair := strings.SplitN(tag, "=", 2)
		key := pair[0]

		var value string
		if len(pair) == 2 {
			value = parseIRCTagValue(pair[1])
		}

		tags[key] = value
	}

	return tags
}

var tagEscapeCharacters = []struct {
	from string
	to   string
}{
	{`\s`, ` `},
	{`\n`, ``},
	{`\r`, ``},
	{`\:`, `;`},
	{`\\`, `\`},
}

func parseIRCTagValue(rawValue string) string {
	for _, escape := range tagEscapeCharacters {
		rawValue = strings.ReplaceAll(rawValue, escape.from, escape.to)
	}

	rawValue = strings.TrimSuffix(rawValue, "\\")

	// Some Twitch values can end with a trailing \s
	// Example: "system-msg=An\sanonymous\suser\sgifted\sa\sTier\s1\ssub\sto\sTenureCalculator!\s"
	rawValue = strings.TrimSpace(rawValue)

	return rawValue
}

func parseIRCMessageSource(rawSource string) *ircMessageSource {
	var source ircMessageSource

	rawSource = strings.TrimPrefix(rawSource, ":")

	regex := regexp.MustCompile(`!|@`)
	split := regex.Split(rawSource, -1)

	if len(split) == 0 {
		return &source
	}

	switch len(split) {
	case 1:
		source.Host = split[0]
	case 2:
		// Getting 2 items extremely rare, but does happen sometimes.
		// https://github.com/gempir/go-twitch-irc/issues/109
		source.Nickname = split[0]
		source.Host = split[1]
	default:
		source.Nickname = split[0]
		source.Username = split[1]
		source.Host = split[2]
	}

	return &source
}
