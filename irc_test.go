package twitch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
)

type ircTest struct {
	Input    string
	Expected ircMessage
}

func ircTests() ([]*ircTest, error) {
	tests := struct {
		Tests []*ircTest
	}{}

	data, err := ioutil.ReadFile("test_resources/irctests.json")
	if err != nil {
		return nil, fmt.Errorf("ircTests: failed to read file: %s", err)
	}

	if err := json.Unmarshal(data, &tests); err != nil {
		return nil, fmt.Errorf("ircTests: failed to unmarshal data: %s", err)
	}

	return tests.Tests, nil
}

func TestCanParseIRCMessage(t *testing.T) {
	tests, err := ircTests()
	if err != nil {
		t.Error(err)
		return
	}

	for _, test := range tests {
		actual, err := parseIRCMessage(test.Input)
		if err != nil {
			t.Error(err)
			continue
		}

		assertStringMapsEqual(t, test.Expected.Tags, actual.Tags)
		assertStringsEqual(t, test.Expected.Source.Nickname, actual.Source.Nickname)
		assertStringsEqual(t, test.Expected.Source.Username, actual.Source.Username)
		assertStringsEqual(t, test.Expected.Source.Host, actual.Source.Host)
		assertStringsEqual(t, test.Expected.Command, actual.Command)
		assertStringSlicesEqual(t, test.Expected.Params, actual.Params)
	}
}

func TestCantParsePartialIRCMessage(t *testing.T) {
	testMessage := "@badges=;color=;display-name=ZZZi;emotes=;flags=;id=75bb6b6b-e36c-49af-a293-16024738ab92;mod=0;room-id=36029255;subscriber=0;tmi-sent-ts=1551476573570;turbo"

	actual, err := parseIRCMessage(testMessage)

	expectedTags := map[string]string{
		"badges":       "",
		"color":        "",
		"display-name": "ZZZi",
		"emotes":       "",
		"flags":        "",
		"id":           "75bb6b6b-e36c-49af-a293-16024738ab92",
		"mod":          "0",
		"room-id":      "36029255",
		"subscriber":   "0",
		"tmi-sent-ts":  "1551476573570",
		"turbo":        "",
	}
	assertStringMapsEqual(t, expectedTags, actual.Tags)

	assertStringsEqual(t, "", actual.Source.Nickname)
	assertStringsEqual(t, "", actual.Source.Username)
	assertStringsEqual(t, "", actual.Source.Host)
	assertStringsEqual(t, "", actual.Command)
	assertStringSlicesEqual(t, nil, actual.Params)

	assertStringsEqual(t, "parseIRCMessage: partial message", err.Error())

}

func TestCantParseNoCommandIRCMessage(t *testing.T) {
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv"

	actual, err := parseIRCMessage(testMessage)

	expectedTags := map[string]string{
		"badges":       "",
		"color":        "#00FF7F",
		"display-name": "Danielps1",
		"emotes":       "",
		"message-id":   "20",
		"thread-id":    "32591953_77829817",
		"turbo":        "0",
		"user-id":      "32591953",
		"user-type":    "",
	}
	assertStringMapsEqual(t, expectedTags, actual.Tags)

	assertStringsEqual(t, "danielps1", actual.Source.Nickname)
	assertStringsEqual(t, "danielps1", actual.Source.Username)
	assertStringsEqual(t, "danielps1.tmi.twitch.tv", actual.Source.Host)
	assertStringsEqual(t, "", actual.Command)
	assertStringSlicesEqual(t, nil, actual.Params)

	assertStringsEqual(t, "parseIRCMessage: no command", err.Error())
}
