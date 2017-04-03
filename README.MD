# go-twitch-irc [![Build Status](https://travis-ci.org/gempir/go-twitch-irc.svg?branch=master)](https://travis-ci.org/gempir/go-twitch-irc)

This is an irc client for connecting to twitch. It handles the annoying stuff like irc tag parsing.

## Getting Started

	package main
    
    import (
    	"github.com/gempir/go-twitch-irc"
    	"fmt"
    )
    
    func main() {
    	client := twitch.NewClient("justinfan123123", "oauth:123123123")
    
    	client.OnNewMessage(func(channel string, user twitch.User, message twitch.Message) {
    		fmt.Println(message.Text)
    	})
    
    	client.Join("gempir")
    
    	client.Connect()
    }

### Available Data

The 2 structs twitch.User and twitch.Message look like this:

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
    	Emotes []*emote
    	Tags   map[string]string
    	Text   string
    }

Channel is just a string like "lirik", note the absent #.

### Message Types

If you ever need more than basic PRIVMSG this might be for you.
These are the 3 mayor message types currently supported

	PRIVMSG
	ROOMSTATE
	CLEARCHAT
	
And their corresponding functions 

	OnNewMessage          func(channel string, user User, message Message)
	OnNewRoomstateMessage func(channel string, user User, message Message)
	OnNewClearchatMessage func(channel string, user User, message Message)