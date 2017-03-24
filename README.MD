# go-twitch-irc [![Build Status](https://travis-ci.org/gempir/go-twitch-irc.svg?branch=master)](https://travis-ci.org/gempir/go-twitch-irc)

This is an irc client for connecting to twitch. It handles the annoying stuff like irc tag parsing.

## In Development

This project is not working as intended yet, if you want to use it anyway, keep in mind a lot of stuff is broken.

## Getting Started

	client := NewClient("myUsername", "oauth:123123")
	
	go func() {
		err := client.Connect()
		fmt.Println(err.Error())
	}()

	client.Join("gempir")
	
	client.OnNewMessage(func(message Message) {
		fmt.Println(message.Text)
	})

	