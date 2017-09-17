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
	var oauthMsg string

	go func() {
		ln, err := net.Listen("tcp", ":4321")
		if err != nil {
			t.Fatal(err)
		}
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		for {
			message, _ := bufio.NewReader(conn).ReadString('\n')
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "PASS") {
				oauthMsg = message
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

	if oauthMsg != "PASS oauth:123123132" {
		t.Fatalf("invalid authentication data: oauth: %s", oauthMsg)
	}
}
