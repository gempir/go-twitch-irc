package main

import (
	"log"
	"strings"

	twitch "github.com/gempir/go-twitch-irc"
)

func main() {
	client := twitch.NewClient("justinfan123123", "oauth:123123123")

	client.OnNewMessage(func(channel string, user twitch.User, message twitch.Message) {
		if strings.Contains(strings.ToLower(message.Text), "ping") {
			log.Println(user.Username, "PONG", message.Text)
		}
	})

	client.Join("pajlada")

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
