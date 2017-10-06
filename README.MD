# go-twitch-irc [![Build Status](https://travis-ci.org/gempir/go-twitch-irc.svg?branch=master)](https://travis-ci.org/gempir/go-twitch-irc) [![Coverage Status](https://coveralls.io/repos/github/gempir/go-twitch-irc/badge.svg?branch=master)](https://coveralls.io/github/gempir/go-twitch-irc?branch=master)

This is an irc client for connecting to twitch. It handles the annoying stuff like irc tag parsing.

godoc: https://godoc.org/github.com/gempir/go-twitch-irc

## Getting Started
```go
package main

import (
	"github.com/gempir/go-twitch-irc"
	"fmt"
)

func main() {
	client := twitch.NewClient("justinfan123123", "oauth:123123123")
	client.ErrChan = make(chan error)

	client.OnNewMessage(func(channel string, user twitch.User, message twitch.Message) {
		fmt.Println(message.Text)
	})
	go client.Connect()
	client.Join("gempir")
	client.Say("gempir", "Hello!")

	// If you wish to read an input message from the command line and then send it to the chat, you could do the following:
	//
	// 	scanner := bufio.NewScanner(os.Stdin)
	// ConnectionLoop:
	// 	for {
	// 		select {
	// 		case error := <-client.ErrChan:
	// 			fmt.Println(error.Error())
	// 			break ConnectionLoop
	// 		default:
	// 			break
	// 		}
	// 		for scanner.Scan() {
	// 			s := scanner.Text()
	// 			client.Say("gempir", s)
	// 		}
	// 		if err := scanner.Err(); err != nil {
	// 			os.Exit(1)
	// 		}
	//	}
}
```
### Available Data

The 2 structs twitch.User and twitch.Message look like this:
```go
type User struct {
	Username    string
	DisplayName string
	UserType    string
	Color       string
	Badges      map[string]int
}

type Message struct {
	Type   msgType
	Time   time.Time
	Action bool
	Emotes []*Emote
	Tags   map[string]string
	Text   string
}
```
Channel is just a string like "lirik", note the absent #.

### Message Types

If you ever need more than basic PRIVMSG this might be for you.
These are the 3 mayor message types currently supported

	PRIVMSG
	ROOMSTATE
	CLEARCHAT
