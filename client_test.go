package twitch

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/textproto"
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
	wait := make(chan struct{})
	waitPass := make(chan struct{})

	var conn net.Conn
	go createServer(t, ":4321", &conn, wait, func() {
		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "PASS") {
				oauthMsg = message
				close(waitPass)
			}
		}
	})

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("client didn't connect")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4321"
	go client.Connect()

	select {
	case <-waitPass:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	if oauthMsg != "PASS oauth:123123132" {
		t.Fatalf("invalid authentication data: oauth: %s", oauthMsg)
	}
}

func TestCanReceivePRIVMSGMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :Thrashh5, FeelsWayTooAmazingMan kinda"
	wait := make(chan struct{})

	var conn net.Conn
	go createServer(t, ":4322", &conn, wait, func() {
		fmt.Fprintf(conn, "%s\r\n", testMessage)
	})

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("client didn't connect")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4322"
	go client.Connect()

	waitMsg := make(chan string)
	var receivedMsg string

	client.OnNewMessage(func(channel string, user User, message Message) {
		receivedMsg = message.Text
		close(waitMsg)
	})

	// wait for server to start
	select {
	case <-waitMsg:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	if receivedMsg != "Thrashh5, FeelsWayTooAmazingMan kinda" {
		t.Fatal("invalid message text received")
	}
}

func TestCanReceiveCLEARCHATMessage(t *testing.T) {
	testMessage := `@ban-duration=1;ban-reason=testing\sxd;room-id=11148817;target-user-id=40910607 :tmi.twitch.tv CLEARCHAT #pajlada :ampzyh`
	wait := make(chan struct{})

	var conn net.Conn
	go createServer(t, ":4323", &conn, wait, func() {
		fmt.Fprintf(conn, "%s\r\n", testMessage)
	})

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("client didn't connect")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4323"
	go client.Connect()

	waitMsg := make(chan string)
	var receivedMsg string

	client.OnNewClearchatMessage(func(channel string, user User, message Message) {
		receivedMsg = message.Text
		close(waitMsg)
	})

	// wait for server to start
	select {
	case <-waitMsg:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "ampzyh was timed out for 1s: testing xd", receivedMsg)
}

func TestCanReceiveROOMSTATEMessage(t *testing.T) {
	testMessage := `@slow=10 :tmi.twitch.tv ROOMSTATE #gempir`
	wait := make(chan struct{})

	var conn net.Conn
	go createServer(t, ":4324", &conn, wait, func() {
		fmt.Fprintf(conn, "%s\r\n", testMessage)
	})

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("client didn't connect")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4324"
	go client.Connect()

	waitMsg := make(chan string)
	var receivedTag string

	client.OnNewRoomstateMessage(func(channel string, user User, message Message) {
		receivedTag = message.Tags["slow"]
		close(waitMsg)
	})

	// wait for server to start
	select {
	case <-waitMsg:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "10", receivedTag)
}

func TestCanSayMessage(t *testing.T) {
	testMessage := "Do not go gentle into that good night."
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	var conn net.Conn
	go createServer(t, ":4325", &conn, wait, func() {
		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "NICK") {
				fmt.Fprintf(conn, ":tmi.twitch.tv 001 justinfan123123 :Welcome, GLHF!\r\n")
			}
			if strings.HasPrefix(message, "PRIVMSG") {
				receivedMsg = message
				close(waitEnd)
			}
		}
	})

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("testserver didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4325"
	go client.Connect()

	client.Say("gempir", testMessage)

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no privmsg received")
	}

	assertStringsEqual(t, "PRIVMSG #gempir :"+testMessage, receivedMsg)
}

func TestCanJoinChannel(t *testing.T) {
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	var conn net.Conn
	go createServer(t, ":4326", &conn, wait, func() {
		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "NICK") {
				fmt.Fprintf(conn, ":tmi.twitch.tv 001 justinfan123123 :Welcome, GLHF!\r\n")
			}
			if strings.HasPrefix(message, "JOIN") {
				receivedMsg = message
				close(waitEnd)
			}
		}
	})

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("testserver didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4326"
	go client.Connect()

	client.Join("gempir")

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no join message received")
	}

	assertStringsEqual(t, "JOIN #gempir", receivedMsg)
}

func TestCanPong(t *testing.T) {
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	var conn net.Conn
	go createServer(t, ":4327", &conn, wait, func() {
		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "NICK") {
				fmt.Fprintf(conn, ":tmi.twitch.tv 001 justinfan123123 :Welcome, GLHF!\r\n")
				fmt.Fprintf(conn, "PING hello\r\n")
			}
			if strings.HasPrefix(message, "PONG") {
				receivedMsg = message
				close(waitEnd)
			}
		}
	})

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("testserver didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4327"
	go client.Connect()

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no join message received")
	}

	assertStringsEqual(t, "PONG hello", receivedMsg)
}

func TestCanNotDialInvalidAddress(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = "127.0.0.1:123123123123"

	err := client.Connect()
	if !strings.Contains(err.Error(), "invalid port") {
		t.Fatal("invalid Connect() error")
	}
}

func createServer(t *testing.T, port string, conn *net.Conn, wait chan struct{}, extra func()) {
	cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
	if err != nil {
		log.Println(err)
		return
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cer},
	}
	ln, err := tls.Listen("tcp", port, config)
	if err != nil {
		t.Fatal(err)
	}

	close(wait)

	*conn, err = ln.Accept()
	if err != nil {
		t.Fatal(err)
	}

	defer ln.Close()
	defer (*conn).Close()

	extra()
}
