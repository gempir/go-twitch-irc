package twitch

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

var startPort = 3000

func nothingOnConnect(conn net.Conn) {
}

func nothingOnMessage(message string) {
}

func postMessageOnConnect(message string) func(conn net.Conn) {
	return func(conn net.Conn) {
		fmt.Fprintf(conn, "%s\r\n", message)
	}
}

func newTestClient(host string) *Client {
	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = host

	return client
}

func startServer(t *testing.T, onConnect func(net.Conn), onMessage func(string)) string {
	host := ":" + strconv.Itoa(startPort)
	startPort++

	cert, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
	if err != nil {
		t.Fatal(err)
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	listener, err := tls.Listen("tcp", host, config)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		defer listener.Close()
		defer conn.Close()
		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "NICK") {
				fmt.Fprintf(conn, ":tmi.twitch.tv 001 justinfan123123 :Welcome, GLHF!\r\n")
				onConnect(conn)
			} else {
				onMessage(message)
			}
		}
	}()

	return host
}

func startNoTLSServer(t *testing.T, onConnect func(net.Conn), onMessage func(string)) string {
	host := ":" + strconv.Itoa(startPort)
	startPort++

	listener, err := net.Listen("tcp", host)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		defer listener.Close()
		defer conn.Close()
		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "NICK") {
				fmt.Fprintf(conn, ":tmi.twitch.tv 001 justinfan123123 :Welcome, GLHF!\r\n")
				onConnect(conn)
			} else {
				onMessage(message)
			}
		}
	}()

	return host
}

func TestCanConnectAndAuthenticateWithoutTLS(t *testing.T) {
	const oauthCode = "oauth:123123132"
	wait := make(chan struct{})

	var received string

	host := startNoTLSServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "PASS") {
			received = message
			close(wait)
		}
	})

	client := NewClient("justinfan123123", oauthCode)
	client.TLS = false
	client.IrcAddress = host
	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, "PASS "+oauthCode, received)
}

func TestCanCreateClient(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:1123123")

	if reflect.TypeOf(*client) != reflect.TypeOf(Client{}) {
		t.Error("client is not of type Client")
	}
}

func TestCanConnectAndAuthenticate(t *testing.T) {
	const oauthCode = "oauth:123123132"
	wait := make(chan struct{})

	var received string

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "PASS") {
			received = message
			close(wait)
		}
	})

	client := NewClient("justinfan123123", oauthCode)
	client.IrcAddress = host
	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, "PASS "+oauthCode, received)
}

func TestCanDisconnect(t *testing.T) {
	wait := make(chan struct{})

	host := startServer(t, nothingOnConnect, nothingOnMessage)
	client := newTestClient(host)

	client.OnConnect(func() {
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("OnConnect did not fire")
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("couldn't disconnect: %s", err.Error())
	}
}

func TestCanNotDisconnectOnClosedConnection(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:123123132")

	if err := client.Disconnect(); !strings.Contains(err.Error(), "connection not open") {
		t.Fatal("no error on disconnecting closed connection")
	}
}

func TestCanReceivePRIVMSGMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :Thrashh5, FeelsWayTooAmazingMan kinda"

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNewMessage(func(channel string, user User, message Message) {
		received = message.Text
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "Thrashh5, FeelsWayTooAmazingMan kinda", received)
}

func TestCanReceiveWHISPERMessage(t *testing.T) {
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv WHISPER gempir :i like memes"

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNewWhisper(func(user User, message Message) {
		received = message.Text
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "i like memes", received)
}

func TestCanReceiveCLEARCHATMessage(t *testing.T) {
	testMessage := `@ban-duration=1;ban-reason=testing\sxd;room-id=11148817;target-user-id=40910607 :tmi.twitch.tv CLEARCHAT #pajlada :ampzyh`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNewClearchatMessage(func(channel string, user User, message Message) {
		received = message.Text
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "ampzyh was timed out for 1s: testing xd", received)
}

func TestCanReceiveROOMSTATEMessage(t *testing.T) {
	testMessage := `@slow=10 :tmi.twitch.tv ROOMSTATE #gempir`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNewRoomstateMessage(func(channel string, user User, message Message) {
		received = message.Tags["slow"]
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "10", received)
}

func TestCanReceiveUSERNOTICEMessage(t *testing.T) {
	testMessage := `@badges=subscriber/12,premium/1;color=#5F9EA0;display-name=blahh;emotes=;id=9154ac04-c9ad-46d5-97ad-15d2dbf244f0;login=deliquid;mod=0;msg-id=resub;msg-param-months=16;msg-param-sub-plan-name=Channel\sSubscription\s(NOTHING);msg-param-sub-plan=Prime;room-id=23161357;subscriber=1;system-msg=blahh\sjust\ssubscribed\swith\sTwitch\sPrime.\sblahh\ssubscribed\sfor\s16\smonths\sin\sa\srow!;tmi-sent-ts=1517165351175;turbo=0;user-id=1234567890;user-type= :tmi.twitch.tv USERNOTICE #nothing`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNewUsernoticeMessage(func(channel string, user User, message Message) {
		received = message.Tags["msg-param-months"]
		close(wait)
	})

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "16", received)
}

func TestCanReceiveUSERStateMessage(t *testing.T) {
	testMessage := `@badges=moderator/1;color=;display-name=blahh;emote-sets=0;mod=1;subscriber=0;user-type=mod :tmi.twitch.tv USERSTATE #nothing`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNewUserstateMessage(func(channel string, user User, message Message) {
		received = message.Tags["mod"]
		close(wait)
	})

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "1", received)
}

func TestCanSayMessage(t *testing.T) {
	testMessage := "Do not go gentle into that good night."

	waitEnd := make(chan struct{})
	var received string

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "PRIVMSG") {
			received = message
			close(waitEnd)
		}
	})

	client := newTestClient(host)

	client.OnConnect(func() {
		client.Say("gempir", testMessage)
	})

	go client.Connect()

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no privmsg received")
	}

	assertStringsEqual(t, "PRIVMSG #gempir :"+testMessage, received)
}

func TestCanWhisperMessage(t *testing.T) {
	testMessage := "Do not go gentle into that good night."

	waitEnd := make(chan struct{})
	var receivedMsg string

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "PRIVMSG") {
			receivedMsg = message
			close(waitEnd)
		}
	})

	client := newTestClient(host)
	go client.Connect()

	client.Whisper("gempir", testMessage)

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no privmsg received")
	}

	assertStringsEqual(t, "PRIVMSG #jtv :/w gempir "+testMessage, receivedMsg)
}

func TestCanJoinChannel(t *testing.T) {
	waitEnd := make(chan struct{})
	var receivedMsg string

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "JOIN") {
			receivedMsg = message
			close(waitEnd)
		}
	})

	client := newTestClient(host)
	go client.Connect()

	client.Join("gempiR")

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no join message received")
	}

	assertStringsEqual(t, "JOIN #gempir", receivedMsg)
}

func TestCanJoinChannelAfterConnection(t *testing.T) {
	waitEnd := make(chan struct{})
	var receivedMsg string

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "JOIN") {
			receivedMsg = message
			close(waitEnd)
		}
	})

	client := newTestClient(host)
	go client.Connect()

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 2)
	}
	client.Join("gempir")

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no join message received")
	}

	assertStringsEqual(t, "JOIN #gempir", receivedMsg)
}

func TestCanDepartChannel(t *testing.T) {
	waitEnd := make(chan struct{})
	var receivedMsg string

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "PART") {
			receivedMsg = message
			close(waitEnd)
		}
	})

	client := newTestClient(host)
	go client.Connect()

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 2)
	}
	client.Depart("gempir")

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no depart message received")
	}

	assertStringsEqual(t, "PART #gempir", receivedMsg)
}

func TestDepartNegatesJoinIfNotConnected(t *testing.T) {
	waitErrorPart := make(chan struct{})
	waitErrorJoin := make(chan struct{})

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "PART") {
			close(waitErrorPart)
		}
		if strings.HasPrefix(message, "JOIN") {
			close(waitErrorJoin)
		}
	})

	client := newTestClient(host)

	client.Join("gempir")
	client.Depart("gempir")

	go client.Connect()

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 2)
	}

	// wait for server to receive message
	select {
	case <-waitErrorPart:
		t.Fatal("erroneously received part message")
	case <-waitErrorJoin:
		t.Fatal("erroneously received join message")
	case <-time.After(time.Millisecond * 100):
	}
}

func TestCanPong(t *testing.T) {
	testMessage := `PING hello`
	var receivedMsg string
	waitEnd := make(chan struct{})

	host := startServer(t, postMessageOnConnect(testMessage), func(message string) {
		// On message received
		if strings.HasPrefix(message, "PONG") {
			receivedMsg = message
			close(waitEnd)
		}
	})

	client := newTestClient(host)
	client.OnConnect(func() {
		client.send("PING hello")
	})

	go client.Connect()

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no pong message received")
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
