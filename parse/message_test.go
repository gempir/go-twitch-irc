package parse

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCanParseMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :Thrashh5, FeelsWayTooAmazingMan kinda"
	message := parseMessage(testMessage)

	assert.Equal(t, "pajlada", message.Channel)
	assert.Equal(t, 6, message.Badges["subscriber"])
	assert.Equal(t, "#FF0000", message.Color)
	assert.Equal(t, "Redflamingo13", message.DisplayName)
	assert.Equal(t, 0, len(message.Emotes))
	assert.Equal(t, "0", message.Tags["mod"])
	assert.Equal(t, "Thrashh5, FeelsWayTooAmazingMan kinda", message.Text)
	assert.Equal(t, PRIVMSG, message.Type)
	assert.Equal(t, "redflamingo13", message.Username)
	assert.Equal(t, "", message.UserType)
	assert.False(t, message.Action)
}

func TestCanParseActionMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :\u0001ACTION Thrashh5, FeelsWayTooAmazingMan kinda"
	message := parseMessage(testMessage)

	assert.Equal(t, "pajlada", message.Channel)
	assert.Equal(t, 6, message.Badges["subscriber"])
	assert.Equal(t, "#FF0000", message.Color)
	assert.Equal(t, "Redflamingo13", message.DisplayName)
	assert.Equal(t, 0, len(message.Emotes))
	assert.Equal(t, "0", message.Tags["mod"])
	assert.Equal(t, "Thrashh5, FeelsWayTooAmazingMan kinda", message.Text)
	assert.Equal(t, PRIVMSG, message.Type)
	assert.Equal(t, "redflamingo13", message.Username)
	assert.Equal(t, "", message.UserType)
	assert.True(t, message.Action)
}

func TestCantParseNoTagsMessage(t *testing.T) {
	testMessage := "my test message"

	message := parseMessage(testMessage)

	assert.Equal(t, testMessage, message.Text)
}

func TestCantParseInvalidMessage(t *testing.T) {
	testMessage := "@my :test message"

	message := parseMessage(testMessage)

	assert.Equal(t, testMessage, message.Text)
}

func TestCanParseClearChatMessage(t *testing.T) {
	testMessage := `@ban-duration=1;ban-reason=testing\sxd;room-id=11148817;target-user-id=40910607 :tmi.twitch.tv CLEARCHAT #pajlada :ampzyh"`

	message := parseMessage(testMessage)

	assert.Equal(t, CLEARCHAT, message.Type)
}

func TestCanParseEmoteMessage(t *testing.T) {
	testMessage := "@badges=;color=#008000;display-name=Zugren;emotes=120232:0-6,13-19,26-32,39-45,52-58;id=51c290e9-1b50-497c-bb03-1667e1afe6e4;mod=0;room-id=11148817;sent-ts=1490382458685;subscriber=0;tmi-sent-ts=1490382456776;turbo=0;user-id=65897106;user-type= :zugren!zugren@zugren.tmi.twitch.tv PRIVMSG #pajlada :TriHard Clap TriHard Clap TriHard Clap TriHard Clap TriHard Clap"

	message := parseMessage(testMessage)

	assert.Equal(t, 1, len(message.Emotes))
}