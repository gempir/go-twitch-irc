package twitch

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
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
	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4321", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "PASS") {
				oauthMsg = message
				close(waitPass)
			}
		}
	}()

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

	assertStringsEqual(t, "PASS oauth:123123132", oauthMsg)
}

func TestCanDisconnect(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :Thrashh5, FeelsWayTooAmazingMan kinda"
	wait := make(chan struct{})

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4328", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		fmt.Fprintf(conn, "%s\r\n", testMessage)
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4328"
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

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4322", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		fmt.Fprintf(conn, "%s\r\n", testMessage)
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
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

	assertStringsEqual(t, "Thrashh5, FeelsWayTooAmazingMan kinda", receivedMsg)
}

func TestCanReceiveWHISPERMessage(t *testing.T) {
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv WHISPER gempir :i like memes"
	wait := make(chan struct{})

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4330", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		fmt.Fprintf(conn, "%s\r\n", testMessage)
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4330"
	go client.Connect()

	waitMsg := make(chan string)
	var receivedMsg string

	client.OnNewWhisper(func(user User, message Message) {
		receivedMsg = message.Text
		close(waitMsg)
	})

	// wait for server to start
	select {
	case <-waitMsg:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "i like memes", receivedMsg)
}

func TestCanReceiveCLEARCHATMessage(t *testing.T) {
	testMessage := `@ban-duration=1;ban-reason=testing\sxd;room-id=11148817;target-user-id=40910607 :tmi.twitch.tv CLEARCHAT #pajlada :ampzyh`
	wait := make(chan struct{})

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4323", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		fmt.Fprintf(conn, "%s\r\n", testMessage)
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
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

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4324", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		fmt.Fprintf(conn, "%s\r\n", testMessage)
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
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

func TestCanReceiveUSERNOTICEMessage(t *testing.T) {
	testMessage := `@badges=subscriber/12,premium/1;color=#5F9EA0;display-name=blahh;emotes=;id=9154ac04-c9ad-46d5-97ad-15d2dbf244f0;login=deliquid;mod=0;msg-id=resub;msg-param-months=16;msg-param-sub-plan-name=Channel\sSubscription\s(NOTHING);msg-param-sub-plan=Prime;room-id=23161357;subscriber=1;system-msg=blahh\sjust\ssubscribed\swith\sTwitch\sPrime.\sblahh\ssubscribed\sfor\s16\smonths\sin\sa\srow!;tmi-sent-ts=1517165351175;turbo=0;user-id=1234567890;user-type= :tmi.twitch.tv USERNOTICE #nothing`
	wait := make(chan struct{})

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4324", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		fmt.Fprintf(conn, "%s\r\n", testMessage)
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4324"
	go client.Connect()

	waitMsg := make(chan string)
	var receivedTag string

	client.OnNewUsernoticeMessage(func(channel string, user User, message Message) {
		receivedTag = message.Tags["msg-param-months"]
		close(waitMsg)
	})

	// wait for server to start
	select {
	case <-waitMsg:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "16", receivedTag)
}

func TestCanSayMessage(t *testing.T) {
	testMessage := "Do not go gentle into that good night."
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4325", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
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
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
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

func TestCanWhisperMessage(t *testing.T) {
	testMessage := "Do not go gentle into that good night."
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4329", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
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
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4329"
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
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4326", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
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
	}()

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

func TestCanJoinChannelAfterConnection(t *testing.T) {
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4350", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
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
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("testserver didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4350"
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
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4331", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "NICK") {
				fmt.Fprintf(conn, ":tmi.twitch.tv 001 justinfan123123 :Welcome, GLHF!\r\n")
			}
			if strings.HasPrefix(message, "PART") {
				receivedMsg = message
				close(waitEnd)
			}
		}
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("testserver didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4331"
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
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var partMessageReceived bool
	var joinMessageReceived bool

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4332", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			message = strings.Replace(message, "\r\n", "", 1)
			if strings.HasPrefix(message, "NICK") {
				fmt.Fprintf(conn, ":tmi.twitch.tv 001 justinfan123123 :Welcome, GLHF!\r\n")
			}
			if strings.HasPrefix(message, "PART") {
				partMessageReceived = true
				close(waitEnd)
			}
			if strings.HasPrefix(message, "JOIN") {
				joinMessageReceived = true
				close(waitEnd)
			}
		}
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("testserver didn't start")
	}

	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = ":4332"

	client.Join("gempir")
	client.Depart("gempir")

	go client.Connect()

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 2)
	}

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 1):
		if partMessageReceived {
			t.Fatal("erroneously received part message")
		}
		if joinMessageReceived {
			t.Fatal("erroneously received join message")
		}
	}
}

func TestCanPong(t *testing.T) {
	wait := make(chan struct{})

	waitEnd := make(chan struct{})
	var receivedMsg string

	go func() {
		cer, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		ln, err := tls.Listen("tcp", ":4327", config)
		if err != nil {
			t.Fatal(err)
		}
		close(wait)
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		defer conn.Close()

		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		for {
			message, err := tp.ReadLine()
			if err != nil && err != io.EOF {
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
	}()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("server didn't start")
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
