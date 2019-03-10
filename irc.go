package twitch

import (
	"fmt"
	"regexp"
	"strings"
)

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
		if !strings.Contains(tag, "=") {
			continue
		}

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

var escapeCharacters = map[string]string{
	"\\s":  " ",
	"\\n":  "",
	"\\:":  ";",
	"\\\\": "\\",
}

func parseIRCTagValue(rawValue string) string {
	for char, value := range escapeCharacters {
		rawValue = strings.Replace(rawValue, char, value, -1)
	}

	rawValue = strings.TrimSuffix(rawValue, "\\")
	rawValue = strings.TrimSpace(rawValue)

	return rawValue
}

func parseIRCMessageSource(rawSource string) *ircMessageSource {
	var source ircMessageSource

	rawSource = strings.TrimPrefix(rawSource, ":")

	regex := regexp.MustCompile(`!|@`)
	split := regex.Split(rawSource, -1)

	if len(split) == 1 {
		source.Host = split[0]
	} else {
		source.Nickname = split[0]
		source.Username = split[1]
		source.Host = split[2]
	}

	return &source
}
