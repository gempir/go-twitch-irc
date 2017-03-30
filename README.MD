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

	type message struct {
    	Type        msgType
    	Time        time.Time
    	Channel     string
    	Username    string
    	DisplayName string
    	UserType    string
    	Color       string
    	Action      bool
    	Badges      map[string]int
    	Emotes      []*emote
    	Tags        map[string]string
    	Text        string
    }
    
    type emote struct {
    	Name  string
    	ID    string
    	Count int
    }

Channel is just a string like "lirik", note the absent #.

### Message Types

If you ever need more than basic PRIVMSG this might be for you.
These are the 3 mayor message types currently supported

	PRIVMSG
	ROOMSTATE
	CLEARCHAT
	
And their corresponding functions 

	onNewMessage          func(channel string, user User, message Message)
	onNewRoomstateMessage func(channel string, user User, message Message)
	onNewClearchatMessage func(channel string, user User, message Message)