package twitch

import (
	"bufio"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCanCreateClient(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:1123123")

	if reflect.TypeOf(*client) != reflect.TypeOf(Client{}) {
		t.Error("client is not of type Client")
	}
}

func TestCanConnectAndAuthenticate(t *testing.T) {
	var nicknameMsg string
	var oauthMsg string

	go func() {
		ln, _ := net.Listen("tcp", ":4321")
		conn, _ := ln.Accept()

		for {
			message, _ := bufio.NewReader(conn).ReadString('\n')
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "NICK") {
				nicknameMsg = message
			}
			if strings.HasPrefix(message, "PASS") {
				oauthMsg = message
			}
			if nicknameMsg != "" && oauthMsg != "" {
				ln.Close()
			}
		}
	}()
	// wait for server to start
	time.Sleep(time.Millisecond * 100)

	client := NewClient("justinfan123123", "oauth:123123132")
	client.SetIrcAddress(":4321")
	go client.Connect()

	// wait for client to connect and server to read messages
	time.Sleep(time.Second)

	if nicknameMsg != "NICK justinfan123123" || oauthMsg != "PASS oauth:123123132" {
		t.Fatalf("invalid authentication data: username: %s, oauth: %s", nicknameMsg, oauthMsg)
	}
}
