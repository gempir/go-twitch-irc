# go-twitch-irc [![Build Status](https://travis-ci.org/gempir/go-twitch-irc.svg?branch=master)](https://travis-ci.org/gempir/go-twitch-irc)

This is an irc client for connecting to twitch. It handles the annoying stuff like irc tag parsing.

## In Development

This project is not working as intended yet, if you want to use it anyway, keep in mind a lot of stuff is broken.

## Getting Started

	package main
    
    import (
    	"github.com/gempir/go-twitch-irc"
    	"fmt"
    )
    
    func main() {
    	client := twitch.NewClient("justinfan123123", "oauth:123123123")
    
    	client.OnNewMessage(func(message twitch.Message) {
    		fmt.Println(message.Text)
    	})
    
    	client.Join("gempir")
    
    	client.Connect()
    }

	