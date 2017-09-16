package twitch

import (
	"reflect"
	"testing"
	"time"
)

func TestCanCreateClient(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:1123123")

	if reflect.TypeOf(*client) != reflect.TypeOf(Client{}) {
		t.Error("client is not of type Client")
	}
}

func TestCanConnect(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:123123132")

	client.SetIrcAddress("irc.chat.twitch.tv:6667")

	go client.Connect()
	time.Sleep(time.Millisecond * 100)
}

func TestCanJoinChannel(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:123123132")

	client.OnNewMessage(func(channel string, user User, message Message) {

	})

	client.OnNewRoomstateMessage(func(channel string, user User, message Message) {

	})

	client.OnNewClearchatMessage(func(channel string, user User, message Message) {

	})

	client.Join("gempir")
	time.Sleep(time.Millisecond * 100)
}
