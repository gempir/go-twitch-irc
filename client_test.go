package twitch

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/textproto"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var startPortMutex sync.Mutex
var startPort = 10000

func newPort() (r int) {
	startPortMutex.Lock()
	r = startPort
	startPort++
	startPortMutex.Unlock()
	return
}

func closeOnConnect(c chan struct{}) func(conn net.Conn) {
	return func(conn net.Conn) {
		close(c)
	}
}

func waitWithTimeout(c chan struct{}) bool {
	select {
	case <-c:
		return true
	case <-time.After(time.Second * 3):
		return false
	}
}

func closeOnPassReceived(pass *string, c chan struct{}) func(message string) {
	return func(message string) {
		if strings.HasPrefix(message, "PASS") {
			*pass = message
			close(c)
		}
	}
}

func nothingOnConnect(conn net.Conn) {
}

func nothingOnMessage(message string) {
}

func clientCloseOnConnect(c chan struct{}) func() {
	return func() {
		close(c)
	}
}

func postMessageOnConnect(message string) func(conn net.Conn) {
	return func(conn net.Conn) {
		fmt.Fprintf(conn, "%s\r\n", message)
	}
}

func postMessagesOnConnect(messages []string) func(conn net.Conn) {
	return func(conn net.Conn) {
		for _, message := range messages {
			fmt.Fprintf(conn, "%s\r\n", message)
		}
	}
}

func newTestClient(host string) *Client {
	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = host

	return client
}

func newAnonymousTestClient(host string) *Client {
	client := NewAnonymousClient()
	client.IrcAddress = host

	return client
}

func connectAndEnsureGoodDisconnect(t *testing.T, client *Client) chan struct{} {
	c := make(chan struct{})

	go func() {
		err := client.Connect()
		assertErrorsEqual(t, ErrClientDisconnected, err)
		close(c)
	}()

	return c
}

func handleTestConnection(t *testing.T, onConnect func(net.Conn), onMessage func(string), listener net.Listener, wg *sync.WaitGroup) {
	conn, err := listener.Accept()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		time.Sleep(100 * time.Millisecond)
		conn.Close()
		wg.Done()
	}()

	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)

	for {
		message, err := tp.ReadLine()
		if err != nil {
			return
		}
		message = strings.Replace(message, "\r\n", "", 1)

		if strings.HasPrefix(message, "NICK") {
			fmt.Fprintf(conn, ":tmi.twitch.tv 001 justinfan123123 :Welcome, GLHF!\r\n")
			onConnect(conn)
			continue
		}

		if strings.HasPrefix(message, "PASS") {
			pass := strings.Split(message, " ")[1]
			if !strings.HasPrefix(pass, "oauth:") {
				fmt.Fprintf(conn, ":tmi.twitch.tv NOTICE * :Improperly formatted auth\r\n")
				return
			} else if pass == "oauth:wrong" {
				fmt.Fprintf(conn, ":tmi.twitch.tv NOTICE * :Login authentication failed\r\n")
				return
			}
		}

		onMessage(message)
	}
}

type testServer struct {
	host string

	stopped chan struct{}
}

func startServer(t *testing.T, onConnect func(net.Conn), onMessage func(string)) string {
	s := startServer2(t, onConnect, onMessage)
	return s.host
}

func startServer2(t *testing.T, onConnect func(net.Conn), onMessage func(string)) *testServer {
	s := &testServer{
		host: "127.0.0.1:" + strconv.Itoa(newPort()),

		stopped: make(chan struct{}),
	}

	cert, err := tls.LoadX509KeyPair("test_resources/server.crt", "test_resources/server.key")
	if err != nil {
		t.Fatal(err)
	}
	config := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	listener, err := tls.Listen("tcp", s.host, config)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go handleTestConnection(t, onConnect, onMessage, listener, &wg)

	go func() {
		wg.Wait()
		listener.Close()

		close(s.stopped)
	}()

	return s
}

func startServerMultiConns(t *testing.T, numConns int, onConnect func(net.Conn), onMessage func(string)) string {
	host := "127.0.0.1:" + strconv.Itoa(newPort())

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

	wg := sync.WaitGroup{}
	wg.Add(numConns)

	for i := 0; i < numConns; i++ {
		go handleTestConnection(t, onConnect, onMessage, listener, &wg)
	}

	go func() {
		wg.Wait()
		listener.Close()
	}()

	return host
}

func startServerMultiConnsNoTLS(t *testing.T, numConns int, onConnect func(net.Conn), onMessage func(string)) string {
	host := "127.0.0.1:" + strconv.Itoa(newPort())

	listener, err := net.Listen("tcp", host)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(numConns)

	for i := 0; i < numConns; i++ {
		go handleTestConnection(t, onConnect, onMessage, listener, &wg)
	}

	go func() {
		wg.Wait()
		listener.Close()
	}()

	return host
}

func startNoTLSServer(t *testing.T, onConnect func(net.Conn), onMessage func(string)) string {
	host := "127.0.0.1:" + strconv.Itoa(newPort())

	listener, err := net.Listen("tcp", host)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go handleTestConnection(t, onConnect, onMessage, listener, &wg)
	go func() {
		wg.Wait()
		listener.Close()
	}()

	return host
}

func TestCanConnectAndAuthenticateWithoutTLS(t *testing.T) {
	t.Parallel()
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
	client.PongTimeout = time.Second * 30
	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, "PASS "+oauthCode, received)
}

func TestCanChangeOauthToken(t *testing.T) {
	t.Parallel()
	const oauthCode = "oauth:123123132"
	wait := make(chan bool)

	var received string

	host := startNoTLSServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "PASS") {
			received = message
			wait <- true
		}
	})

	client := NewClient("justinfan123123", "wrongoauthcodelol")
	client.TLS = false
	client.IrcAddress = host
	client.SetIRCToken(oauthCode)
	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, "PASS "+oauthCode, received)
}

func TestCanAddSetupCmd(t *testing.T) {
	t.Parallel()
	const oauthCode = "oauth:123123132"
	const setupCmd = "LOGIN kkonabot"
	wait := make(chan bool)

	var received string

	host := startNoTLSServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "LOGIN") {
			received = message
			wait <- true
		}
	})

	client := NewClient("justinfan123123", oauthCode)
	client.TLS = false
	client.IrcAddress = host
	client.SetupCmd = setupCmd
	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, setupCmd, received)
}

func TestCanCreateClient(t *testing.T) {
	t.Parallel()
	client := NewClient("justinfan123123", "oauth:1123123")

	if reflect.TypeOf(client) != reflect.TypeOf(&Client{}) {
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

	client := newTestClient(host)
	client.PongTimeout = time.Second * 30
	connectAndEnsureGoodDisconnect(t, client)
	defer client.Disconnect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, "PASS "+oauthCode, received)
}

// This test is meant to be a blueprint for a test that needs the flow completely from server start to server stop
func TestFullConnectAndDisconnect(t *testing.T) {
	const oauthCode = "oauth:123123132"
	waitPass := make(chan struct{})
	waitServerConnect := make(chan struct{})
	waitClientConnect := make(chan struct{})

	var received string

	server := startServer2(t, closeOnConnect(waitServerConnect), closeOnPassReceived(&received, waitPass))

	client := newTestClient(server.host)
	client.OnConnect(clientCloseOnConnect(waitClientConnect))
	clientDisconnected := connectAndEnsureGoodDisconnect(t, client)

	// Wait for correct password to be read in server
	if !waitWithTimeout(waitPass) {
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, "PASS "+oauthCode, received)

	// Wait for server to acknowledge connection
	if !waitWithTimeout(waitServerConnect) {
		t.Fatal("no successful connection")
	}

	// Wait for client to acknowledge connection
	if !waitWithTimeout(waitClientConnect) {
		t.Fatal("no successful connection")
	}

	// Disconnect client from server
	err := client.Disconnect()
	if err != nil {
		t.Error("Error during disconnect:" + err.Error())
	}

	// Wait for client to be fully disconnected
	<-clientDisconnected

	// Wait for server to be fully disconnected
	<-server.stopped
}

func TestCanConnectAndAuthenticateAnonymous(t *testing.T) {
	const oauthCode = "oauth:59301"
	waitPass := make(chan struct{})
	waitServerConnect := make(chan struct{})
	waitClientConnect := make(chan struct{})

	var received string

	server := startServer2(t, closeOnConnect(waitServerConnect), closeOnPassReceived(&received, waitPass))

	client := newAnonymousTestClient(server.host)
	client.OnConnect(clientCloseOnConnect(waitClientConnect))
	clientDisconnected := connectAndEnsureGoodDisconnect(t, client)

	// Wait for server to acknowledge connection
	if !waitWithTimeout(waitServerConnect) {
		t.Fatal("no successful connection")
	}

	// Wait for client to acknowledge connection
	if !waitWithTimeout(waitClientConnect) {
		t.Fatal("no successful connection")
	}

	// Wait to receive password
	select {
	case <-waitPass:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, "PASS "+oauthCode, received)

	// Disconnect client from server
	err := client.Disconnect()
	if err != nil {
		t.Error("Error during disconnect:" + err.Error())
	}

	// Wait for client to be fully disconnected
	<-clientDisconnected

	// Wait for server to be fully disconnected
	<-server.stopped
}

func TestCanDisconnect(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	client := NewClient("justinfan123123", "oauth:123123132")

	err := client.Disconnect()

	assertErrorsEqual(t, ErrConnectionIsNotOpen, err)
}

func TestCanReceivePRIVMSGMessage(t *testing.T) {
	t.Parallel()
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :Thrashh5, FeelsWayTooAmazingMan kinda"

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnPrivateMessage(func(message PrivateMessage) {
		received = message.Message
		assertMessageTypesEqual(t, PRIVMSG, message.GetType())
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
	t.Parallel()
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv WHISPER gempir :i like memes"

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnWhisperMessage(func(message WhisperMessage) {
		received = message.Message
		assertMessageTypesEqual(t, WHISPER, message.GetType())
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
	t.Parallel()
	testMessage := `@ban-duration=1;ban-reason=testing\sxd;room-id=11148817;target-user-id=40910607 :tmi.twitch.tv CLEARCHAT #pajlada :ampzyh`

	wait := make(chan struct{})
	var received int

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnClearChatMessage(func(message ClearChatMessage) {
		received = message.BanDuration
		assertMessageTypesEqual(t, CLEARCHAT, message.GetType())
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertIntsEqual(t, 1, received)
}

func TestCanReceiveCLEARMSGMessage(t *testing.T) {
	t.Parallel()
	testMessage := `@login=ronni;target-msg-id=abc-123-def :tmi.twitch.tv CLEARMSG #dallas :HeyGuys`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnClearMessage(func(message ClearMessage) {
		received = message.Login
		assertMessageTypesEqual(t, CLEARMSG, message.GetType())
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "ronni", received)
}

func TestCanReceiveROOMSTATEMessage(t *testing.T) {
	t.Parallel()
	testMessage := `@slow=10 :tmi.twitch.tv ROOMSTATE #gempir`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnRoomStateMessage(func(message RoomStateMessage) {
		received = message.Tags["slow"]
		assertMessageTypesEqual(t, ROOMSTATE, message.GetType())
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
	t.Parallel()
	testMessage := `@badges=subscriber/12,premium/1;color=#5F9EA0;display-name=blahh;emotes=;id=9154ac04-c9ad-46d5-97ad-15d2dbf244f0;login=deliquid;mod=0;msg-id=resub;msg-param-months=16;msg-param-sub-plan-name=Channel\sSubscription\s(NOTHING);msg-param-sub-plan=Prime;room-id=23161357;subscriber=1;system-msg=blahh\sjust\ssubscribed\swith\sTwitch\sPrime.\sblahh\ssubscribed\sfor\s16\smonths\sin\sa\srow!;tmi-sent-ts=1517165351175;turbo=0;user-id=1234567890;user-type= :tmi.twitch.tv USERNOTICE #nothing`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserNoticeMessage(func(message UserNoticeMessage) {
		received = message.Tags["msg-param-months"]
		assertMessageTypesEqual(t, USERNOTICE, message.GetType())
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

func TestCanReceiveUSERNOTICEMessageResub(t *testing.T) {
	t.Parallel()
	testMessage := `@badges=moderator/1,subscriber/24;color=#1FD2FF;display-name=Karl_Kons;emotes=28087:0-6;flags=;id=7c95beea-a7ac-4c10-9e0a-d7dbf163c038;login=karl_kons;mod=1;msg-id=resub;msg-param-months=34;msg-param-sub-plan-name=look\sat\sthose\sshitty\semotes,\srip\s$5\sLUL;msg-param-sub-plan=1000;room-id=11148817;subscriber=1;system-msg=Karl_Kons\sjust\ssubscribed\swith\sa\sTier\s1\ssub.\sKarl_Kons\ssubscribed\sfor\s34\smonths\sin\sa\srow!;tmi-sent-ts=1540140252828;turbo=0;user-id=68706331;user-type=mod :tmi.twitch.tv USERNOTICE #pajlada :WutFace`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserNoticeMessage(func(message UserNoticeMessage) {
		received = message.Tags["msg-param-months"]
		close(wait)
	})

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "34", received)
}

func checkNoticeMessage(t *testing.T, testMessage string, requirements map[string]string) {
	received := map[string]string{}
	wait := make(chan struct{})

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNoticeMessage(func(message NoticeMessage) {
		received["msg-id"] = message.Tags["msg-id"]
		received["channel"] = message.Channel
		received["text"] = message.Message
		received["raw"] = message.Raw
		assertMessageTypesEqual(t, NOTICE, message.GetType())
		close(wait)
	})

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, testMessage, received["raw"])
	for key, requirement := range requirements {
		assertStringsEqual(t, requirement, received[key])
	}
}

func TestCanReceiveNOTICEMessage(t *testing.T) {
	t.Parallel()
	testMessage := `@msg-id=host_on :tmi.twitch.tv NOTICE #pajlada :Now hosting KKona.`
	checkNoticeMessage(t, testMessage, map[string]string{
		"msg-id":  "host_on",
		"channel": "pajlada",
		"text":    "Now hosting KKona.",
	})
}

func TestCanReceiveNOTICEMessageTimeout(t *testing.T) {
	t.Parallel()
	testMessage := `@msg-id=timeout_success :tmi.twitch.tv NOTICE #forsen :thedl0rd has been timed out for 8 minutes 11 seconds.`
	checkNoticeMessage(t, testMessage, map[string]string{
		"msg-id":  "timeout_success",
		"channel": "forsen",
		"text":    "thedl0rd has been timed out for 8 minutes 11 seconds.",
	})
}

func TestCanReceiveUSERStateMessage(t *testing.T) {
	t.Parallel()
	testMessage := `@badges=moderator/1;color=;display-name=blahh;emote-sets=0;mod=1;subscriber=0;user-type=mod :tmi.twitch.tv USERSTATE #nothing`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserStateMessage(func(message UserStateMessage) {
		received = message.Tags["mod"]
		assertMessageTypesEqual(t, USERSTATE, message.GetType())
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

func TestCanReceiveGlobalUserStateMessage(t *testing.T) {
	t.Parallel()
	testMessage := `@badge-info=;badges=;color=#00FF7F;display-name=gempbot;emote-sets=0,14417,300206298,300374282,300548762;user-id=99659894;user-type= :tmi.twitch.tv GLOBALUSERSTATE`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnGlobalUserStateMessage(func(message GlobalUserStateMessage) {
		received = message.Tags["user-id"]
		assertMessageTypesEqual(t, GLOBALUSERSTATE, message.GetType())
		close(wait)
	})

	//nolint
	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "99659894", received)
}

func TestCanReceiveJOINMessage(t *testing.T) {
	t.Parallel()
	testMessage := `:username123!username123@username123.tmi.twitch.tv JOIN #mychannel`

	wait := make(chan struct{})
	var received UserJoinMessage

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserJoinMessage(func(message UserJoinMessage) {
		received = message
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "username123", received.User)
	assertStringsEqual(t, "mychannel", received.Channel)
	assertMessageTypesEqual(t, JOIN, received.GetType())
}

func TestDoesNotReceiveJOINMessageFromSelf(t *testing.T) {
	t.Parallel()
	testMessages := []string{
		`:justinfan123123!justinfan123123@justinfan123123.tmi.twitch.tv JOIN #mychannel`,
		`:username123!username123@username123.tmi.twitch.tv JOIN #mychannel`,
	}

	wait := make(chan struct{})
	var received UserJoinMessage

	host := startServer(t, postMessagesOnConnect(testMessages), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserJoinMessage(func(message UserJoinMessage) {
		received = message
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "username123", received.User)
	assertStringsEqual(t, "mychannel", received.Channel)
	assertMessageTypesEqual(t, JOIN, received.GetType())
}

func TestCanReceivePARTMessage(t *testing.T) {
	t.Parallel()
	testMessage := `:username123!username123@username123.tmi.twitch.tv PART #mychannel`

	wait := make(chan struct{})
	var received UserPartMessage

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserPartMessage(func(message UserPartMessage) {
		received = message
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "username123", received.User)
	assertStringsEqual(t, "mychannel", received.Channel)
	assertMessageTypesEqual(t, PART, received.GetType())
}

func TestDoesNotReceivePARTMessageFromSelf(t *testing.T) {
	t.Parallel()
	testMessages := []string{
		`:justinfan123123!justinfan123123@justinfan123123.tmi.twitch.tv PART #mychannel`,
		`:username123!username123@username123.tmi.twitch.tv PART #mychannel`,
	}

	wait := make(chan struct{})
	var received UserPartMessage

	host := startServer(t, postMessagesOnConnect(testMessages), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserPartMessage(func(message UserPartMessage) {
		received = message
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "username123", received.User)
	assertStringsEqual(t, "mychannel", received.Channel)
	assertMessageTypesEqual(t, PART, received.GetType())
}

func TestCanReceiveUNSETMessage(t *testing.T) {
	t.Parallel()
	testMessage := `@badges=moderator/1,subscriber/24;color=#1FD2FF;display-name=Karl_Kons;emotes=28087:0-6;flags=;id=7c95beea-a7ac-4c10-9e0a-d7dbf163c038;login=karl_kons;mod=1;msg-id=resub;msg-param-months=34;msg-param-sub-plan-name=look\sat\sthose\sshitty\semotes,\srip\s$5\sLUL;msg-param-sub-plan=1000;room-id=11148817;subscriber=1;system-msg=Karl_Kons\sjust\ssubscribed\swith\sa\sTier\s1\ssub.\sKarl_Kons\ssubscribed\sfor\s34\smonths\sin\sa\srow!;tmi-sent-ts=1540140252828;turbo=0;user-id=68706331;user-type=mod :tmi.twitch.tv MALFORMEDMESSAGETYPETHISWILLBEUNSET #pajlada :WutFace`

	wait := make(chan struct{})
	var received RawMessage

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnUnsetMessage(func(rawMessage RawMessage) {
		if rawMessage.RawType == "MALFORMEDMESSAGETYPETHISWILLBEUNSET" {
			received = rawMessage
			close(wait)
		}
	})

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, testMessage, received.Raw)
	assertMessageTypesEqual(t, UNSET, received.GetType())
}

func TestCanHandleRECONNECTMessage(t *testing.T) {
	t.Parallel()
	const testMessage = ":tmi.twitch.tv RECONNECT"

	wait := make(chan bool)

	var received ReconnectMessage

	var connCount int32

	host := startServerMultiConns(t, 2, func(conn net.Conn) {
		atomic.AddInt32(&connCount, 1)
		wait <- true
		time.AfterFunc(100*time.Millisecond, func() {
			fmt.Fprintf(conn, "%s\r\n", testMessage)
		})
	}, nothingOnMessage)
	client := newTestClient(host)
	client.OnReconnectMessage(func(msg ReconnectMessage) {
		received = msg
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertInt32sEqual(t, 1, atomic.LoadInt32(&connCount))

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertInt32sEqual(t, 2, atomic.LoadInt32(&connCount))

	assertMessageTypesEqual(t, RECONNECT, received.GetType())
}

func TestCanSayMessage(t *testing.T) {
	t.Parallel()
	const testMessage = "Do not go gentle into that good night."

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
	t.Parallel()
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

	assertStringsEqual(t, "PRIVMSG #justinfan123123 :/w gempir "+testMessage, receivedMsg)
}

func TestCanJoinChannel(t *testing.T) {
	t.Parallel()
	waitEnd := make(chan struct{})
	var receivedMsg string

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "JOIN") {
			receivedMsg = message
			close(waitEnd)
		}
	})

	client := newTestClient(host)

	client.Join("gempiR")

	go client.Connect()

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no join message received")
	}

	assertStringsEqual(t, "JOIN #gempir", receivedMsg)
}

func TestCanRunFollowersOn(t *testing.T) {
	t.Parallel()

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
		client.FollowersOn("gempiR", "30m")
	})

	go client.Connect() //nolint

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no privmsg received")
	}

	assertStringsEqual(t, "PRIVMSG #gempir :/followers 30m", received)
}

func TestCanRunFollowersOff(t *testing.T) {
	t.Parallel()

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
		client.FollowersOff("gempiR")
	})

	go client.Connect() //nolint

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no privmsg received")
	}

	assertStringsEqual(t, "PRIVMSG #gempir :/followersoff", received)
}

func TestCanJoinChannelAfterConnection(t *testing.T) {
	t.Parallel()
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

func TestCanRespectDefaultJoinRateLimits(t *testing.T) {
	t.Parallel()
	waitEnd := make(chan struct{})

	var joinMessages []timedTestMessage
	targetJoinCount := 25

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "JOIN ") {
			joinMessages = append(joinMessages, timedTestMessage{message, time.Now()})

			if len(joinMessages) == targetJoinCount {
				close(waitEnd)
			}
		}
	})

	client := newTestClient(host)
	client.PongTimeout = time.Second * 30
	client.SetRateLimiter(CreateDefaultRateLimiter())
	go client.Connect() //nolint

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 2)
	}

	// send enough messages to ensure we hit the rate limit
	for i := 1; i <= targetJoinCount; i++ {
		client.Join(fmt.Sprintf("gempir%d", i))
	}

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 30):
		t.Fatal("didn't receive all messages in time")
	}

	assertJoinRateLimitRespected(t, client.rateLimiter.joinLimit, joinMessages)
}

func TestCanRespectBulkDefaultJoinRateLimits(t *testing.T) {
	t.Parallel()
	waitEnd := make(chan struct{})

	var joinMessages []timedTestMessage
	targetJoinCount := 50

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "JOIN ") {
			splits := strings.Split(message, ",")
			for _, split := range splits {
				joinMessages = append(joinMessages, timedTestMessage{split, time.Now()})
			}

			if len(joinMessages) == targetJoinCount {
				close(waitEnd)
			}
		}
	})

	client := newTestClient(host)
	client.PongTimeout = time.Second * 60
	client.SetRateLimiter(CreateDefaultRateLimiter())
	go client.Connect() //nolint

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 2)
	}

	perBulk := 25
	// send enough messages to ensure we hit the rate limit
	for i := 1; i <= targetJoinCount; {
		channels := []string{}
		for j := i; j < i+perBulk; j++ {
			channels = append(channels, fmt.Sprintf("gempir%d", j))
		}

		client.Join(channels...)
		i += perBulk
	}

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 60):
		t.Fatal("didn't receive all messages in time")
	}

	assertJoinRateLimitRespected(t, client.rateLimiter.joinLimit, joinMessages)
}

func TestCanRespectVerifiedJoinRateLimits(t *testing.T) {
	t.Parallel()
	waitEnd := make(chan struct{})

	var joinMessages []timedTestMessage
	targetJoinCount := 3000

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "JOIN ") {
			joinMessages = append(joinMessages, timedTestMessage{message, time.Now()})

			if len(joinMessages) == targetJoinCount {
				close(waitEnd)
			}
		}
	})

	client := newTestClient(host)
	client.PongTimeout = time.Second * 30
	client.SetRateLimiter(CreateVerifiedRateLimiter())
	go client.Connect() //nolint

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 2)
	}

	// send enough messages to ensure we hit the rate limit
	for i := 1; i <= targetJoinCount; i++ {
		client.Join(fmt.Sprintf("gempir%d", i))
	}

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 30):
		t.Fatal("didn't receive all messages in time")
	}

	assertJoinRateLimitRespected(t, client.rateLimiter.joinLimit, joinMessages)
}

func TestCanIgnoreJoinRateLimits(t *testing.T) {
	t.Parallel()
	waitEnd := make(chan struct{})

	var messages []timedTestMessage
	targetJoinCount := 3000 // this breaks when above 700, why? the fuck?

	host := startServer(t, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "JOIN ") {
			messages = append(messages, timedTestMessage{message, time.Now()})

			if len(messages) == targetJoinCount {
				close(waitEnd)
			}
		}
	})

	client := newTestClient(host)
	client.PongTimeout = time.Second * 30
	client.SetRateLimiter(CreateUnlimitedRateLimiter())
	go client.Connect() //nolint

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 2)
	}

	// send enough messages to ensure we hit the rate limit
	for i := 1; i <= targetJoinCount; i++ {
		client.Join(fmt.Sprintf("gempir%d", i))
	}

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 10):
		t.Fatal("didn't receive all messages in time")
	}

	lastMessageTime := messages[len(messages)-1].time
	firstMessageTime := messages[0].time

	assertTrue(t, lastMessageTime.Sub(firstMessageTime).Seconds() <= 10, fmt.Sprintf("join ratelimit not skipped last message time: %s, first message time: %s", lastMessageTime, firstMessageTime))
}

func TestCanDepartChannel(t *testing.T) {
	t.Parallel()
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

func TestCanGetUserlist(t *testing.T) {
	t.Parallel()
	expectedNames := []string{"username1", "username2"}
	testMessages := []string{
		`:justinfan123123.tmi.twitch.tv 353 justinfan123123 = #channel123 :username1 username2`,
		`@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #anythingbutchannel123 :ok go now`,
	}
	waitEnd := make(chan struct{})

	host := startServer(t, postMessagesOnConnect(testMessages), nothingOnMessage)

	client := newTestClient(host)

	var received NamesMessage

	client.OnNamesMessage(func(message NamesMessage) {
		received = message
	})

	client.Join("channel123")

	client.OnPrivateMessage(func(message PrivateMessage) {
		if message.Message == "ok go now" {
			// test a valid channel
			got, err := client.Userlist("channel123")
			if err != nil {
				t.Fatal("error not nil for client.Userlist")
			}

			sort.Strings(got)

			assertStringSlicesEqual(t, expectedNames, got)

			// test an unknown channel
			got, err = client.Userlist("random_channel123")
			if err == nil || got != nil {
				t.Fatal("error expected on unknown channel for client.Userlist")
			}

			close(waitEnd)
		}
	})

	go client.Connect()

	// wait for the connection to go active
	for !client.connActive.get() {
		time.Sleep(time.Millisecond * 5)
	}

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no userlist received")
	}

	assertStringsEqual(t, "channel123", received.Channel)
	assertStringSlicesEqual(t, expectedNames, received.Users)
	assertMessageTypesEqual(t, NAMES, received.GetType())
}

func TestDepartNegatesJoinIfNotConnected(t *testing.T) {
	t.Parallel()
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

func TestCanRespondToPING1(t *testing.T) {
	t.Parallel()
	testMessage := `PING`
	expectedMessage := `PONG`
	waitEnd := make(chan struct{})

	host := startServer(t, postMessageOnConnect(testMessage), func(message string) {
		// On message received
		if message == expectedMessage {
			close(waitEnd)
		}
	})

	client := newTestClient(host)

	go client.Connect()

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no pong message received")
	}
}

func TestCanRespondToPING2(t *testing.T) {
	t.Parallel()
	testMessage := `:tmi.twitch.tv PING`
	expectedMessage := `PONG`
	waitEnd := make(chan struct{})

	host := startServer(t, postMessageOnConnect(testMessage), func(message string) {
		// On message received
		if message == expectedMessage {
			close(waitEnd)
		}
	})

	client := newTestClient(host)

	go client.Connect()

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no pong message received")
	}
}

func TestCanAttachToPingMessageCallback(t *testing.T) {
	t.Parallel()
	testMessage := `:tmi.twitch.tv PING`
	wait := make(chan struct{})

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)

	client := newTestClient(host)

	client.OnPingMessage(func(msg PingMessage) {
		close(wait)
	})

	go client.Connect()

	// wait for server to receive message
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no ping message received")
	}
}

func TestCanPong(t *testing.T) {
	t.Parallel()
	testMessage := `PING :hello`
	expectedMessage := `PONG :hello`
	waitEnd := make(chan struct{})

	host := startServer(t, postMessageOnConnect(testMessage), func(message string) {
		// On message received
		if message == expectedMessage {
			close(waitEnd)
		}
	})

	client := newTestClient(host)

	go client.Connect()

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no pong message received")
	}
}

func TestCanNotDialInvalidAddress(t *testing.T) {
	t.Parallel()
	client := NewClient("justinfan123123", "oauth:123123132")
	client.IrcAddress = "127.0.0.1:123123123123"

	err := client.Connect()
	if !strings.Contains(err.Error(), "invalid port") {
		t.Fatal("wrong Connect() error: " + err.Error())
	}
}

func TestCanNotUseImproperlyFormattedOauthPENIS(t *testing.T) {
	t.Parallel()
	host := startServer(t, nothingOnConnect, nothingOnMessage)
	client := NewClient("justinfan123123", "imrpproperlyformattedoauth")
	client.IrcAddress = host

	err := client.Connect()
	if err != ErrLoginAuthenticationFailed {
		t.Fatal("wrong Connect() error: " + err.Error())
	}
}

func TestCanNotUseWrongOauthPENIS123(t *testing.T) {
	t.Parallel()
	host := startServer(t, nothingOnConnect, nothingOnMessage)
	client := NewClient("justinfan123123", "oauth:wrong")
	client.IrcAddress = host

	err := client.Connect()
	if err != ErrLoginAuthenticationFailed {
		t.Fatal("wrong Connect() error: " + err.Error())
	}
}

func TestCanConnectToTwitch(t *testing.T) {
	t.Parallel()
	client := NewClient("justinfan123123", "oauth:123123132")

	client.OnConnect(func() {
		client.Disconnect()
	})

	err := client.Connect()
	assertErrorsEqual(t, ErrClientDisconnected, err)
}

func TestCanConnectToTwitchWithoutTLS(t *testing.T) {
	t.Parallel()
	client := NewClient("justinfan123123", "oauth:123123132")
	client.TLS = false
	wait := make(chan struct{})

	client.OnConnect(func() {
		client.Disconnect()
	})

	go func() {
		err := client.Connect()
		assertErrorsEqual(t, ErrClientDisconnected, err)
		close(wait)
	}()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}
}

func TestCanHandleInvalidNick(t *testing.T) {
	t.Parallel()
	client := NewClient("", "")
	client.TLS = false
	wait := make(chan struct{})

	client.OnConnect(func() {
		t.Fatal("A connection should not be able to be established with an invalid nick")
	})

	go func() {
		err := client.Connect()
		assertErrorsEqual(t, ErrLoginAuthenticationFailed, err)
		close(wait)
	}()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}
}

func TestLocalSendingPingsReceivedPong(t *testing.T) {
	t.Parallel()
	const idlePingInterval = 300 * time.Millisecond

	wait := make(chan bool)

	var conn net.Conn

	host := startServer(t, func(c net.Conn) {
		conn = c
	}, func(message string) {
		if message == pingMessage {
			// Send an emulated pong
			fmt.Fprintf(conn, formatPong(strings.Split(message, " :")[1])+"\r\n")
			wait <- true
		}
	})
	client := newTestClient(host)
	client.IdlePingInterval = idlePingInterval

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}

	client.Disconnect()
}

func TestLocalCanReconnectAfterNoPongResponse(t *testing.T) {
	t.Parallel()
	const idlePingInterval = 300 * time.Millisecond
	const pongTimeout = 300 * time.Millisecond

	wait := make(chan bool)

	var connCount int32

	host := startServerMultiConns(t, 3, func(conn net.Conn) {
		atomic.AddInt32(&connCount, 1)
		wait <- true
	}, nothingOnMessage)
	client := newTestClient(host)
	client.IdlePingInterval = idlePingInterval
	client.PongTimeout = pongTimeout

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}

	assertInt32sEqual(t, 1, atomic.LoadInt32(&connCount))

	// Wait for reconnect based on lack of ping response
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}

	assertInt32sEqual(t, 2, atomic.LoadInt32(&connCount))

	// Wait for another reconnect based on lack of ping response
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}

	assertInt32sEqual(t, 3, atomic.LoadInt32(&connCount))
}

func TestLocalSendingPingsReceivedPongAlsoDisconnect(t *testing.T) {
	t.Parallel()
	const idlePingInterval = 300 * time.Millisecond

	wait := make(chan bool)

	var conn net.Conn

	host := startServer(t, func(c net.Conn) {
		conn = c
	}, func(message string) {
		if message == pingMessage {
			// Send an emulated pong
			fmt.Fprintf(conn, formatPong(strings.Split(message, " :")[1])+"\r\n")
			conn.Close()
			wait <- true
		}
	})
	client := newTestClient(host)
	client.IdlePingInterval = idlePingInterval

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}

	client.Disconnect()
}

func TestPinger(t *testing.T) {
	t.Parallel()
	const idlePingInterval = 300 * time.Millisecond

	wait := make(chan bool)

	var conn net.Conn

	var pingpongMutex sync.Mutex
	var pingsSent int
	var pongsReceived int

	host := startServer(t, func(c net.Conn) {
		conn = c
	}, func(message string) {
		if message == pingMessage {
			// Send an emulated pong
			fmt.Fprintf(conn, formatPong(strings.Split(message, " :")[1])+"\r\n")
			wait <- true
		}
	})
	client := newTestClient(host)
	client.IdlePingInterval = idlePingInterval
	client.OnPingSent(func() {
		pingpongMutex.Lock()
		pingsSent++
		pingpongMutex.Unlock()
	})

	client.OnPongMessage(func(msg PongMessage) {
		pingpongMutex.Lock()
		pongsReceived++
		pingpongMutex.Unlock()
	})

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}

	wait = make(chan bool)

	// Ping has been sent by server
	go func() {
		for {
			<-time.After(5 * time.Millisecond)
			pingpongMutex.Lock()
			if pingsSent == pongsReceived {
				wait <- pingsSent == 1
				pingpongMutex.Unlock()
				return
			}
			pingpongMutex.Unlock()
		}
	}()

	select {
	case res := <-wait:
		assertTrue(t, res, "did not send a ping??????")
	case <-time.After(time.Second * 3):
		t.Fatal("Did not receive a pong")
	}

	client.Disconnect()
}

func TestCanAttachToPongMessageCallback(t *testing.T) {
	t.Parallel()

	pongMessage := `:tmi.twitch.tv PONG tmi.twitch.tv :go-twitch-irc`

	wait := make(chan struct{})

	host := startServer(t, postMessageOnConnect(pongMessage), nothingOnMessage)

	client := newTestClient(host)

	var received string

	client.OnPongMessage(func(msg PongMessage) {
		received = msg.Message
		close(wait)
	})

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}

	client.Disconnect()

	assertStringsEqual(t, "go-twitch-irc", received)
}

type createJoinMessageResult struct {
	messages []string
	joined   []string
}

func TestCreateJoinMessagesCreatesMessages(t *testing.T) {
	cases := []struct {
		channels []string
		expected createJoinMessageResult
	}{
		{
			channels: nil,
			expected: createJoinMessageResult{
				messages: []string{},
				joined:   []string{},
			},
		},
		{
			channels: []string{},
			expected: createJoinMessageResult{
				messages: []string{},
				joined:   []string{},
			},
		},
		{
			channels: []string{"pajlada", "forsen"},
			expected: createJoinMessageResult{
				messages: []string{"JOIN #pajlada,#forsen"},
				joined:   []string{"pajlada", "forsen"},
			},
		},
		{
			channels: []string{
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaa",
			},
			expected: createJoinMessageResult{
				messages: []string{"JOIN #aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaa"},
				joined: []string{
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaa",
				},
			},
		},
		{
			channels: []string{
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
			expected: createJoinMessageResult{
				messages: []string{
					"JOIN #aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa,#aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"JOIN #bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
				joined: []string{
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
			},
		},
	}

	client := NewAnonymousClient()
	client.channels = make(map[string]bool)

	for _, test := range cases {
		messages, joined := client.createJoinMessages(test.channels...)
		assertStringSlicesEqual(t, test.expected.messages, messages)
		assertStringSlicesEqual(t, test.expected.joined, joined)
	}
}

func TestCreateJoinMessageReturnsLowercase(t *testing.T) {
	channels := []string{"PAJLADA", "FORSEN"}
	joined := make(map[string]bool)
	expected := []string{"JOIN #pajlada,#forsen"}
	expectedJoined := []string{"pajlada", "forsen"}

	client := NewAnonymousClient()
	client.channels = joined
	actual, actualJoined := client.createJoinMessages(channels...)
	assertStringSlicesEqual(t, expected, actual)
	assertStringSlicesEqual(t, expectedJoined, actualJoined)
}

func TestCreateJoinMessageSkipsJoinedChannels(t *testing.T) {
	channels := []string{"pajlada", "forsen", "nymn"}
	joined := map[string]bool{
		"pajlada": true,
		"forsen":  false,
	}
	expected := []string{"forsen", "nymn"}

	client := NewAnonymousClient()
	client.channels = joined
	_, actual := client.createJoinMessages(channels...)
	assertStringSlicesEqual(t, expected, actual)
}

func TestRejoinOnReconnect(t *testing.T) {
	t.Parallel()
	waitEnd := make(chan struct{})
	var receivedMsg string

	host := startServerMultiConns(t, 2, nothingOnConnect, func(message string) {
		if strings.HasPrefix(message, "JOIN") {
			receivedMsg = message
			close(waitEnd)
		}
	})

	client := newTestClient(host)

	client.Join("gempiR")

	clientDisconnected := connectAndEnsureGoodDisconnect(t, client)

	// wait for server to receive message
	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no join message received")
	}

	// Server received first JOIN message
	assertStringsEqual(t, "JOIN #gempir", receivedMsg)

	receivedMsg = ""

	// Manually disconnect
	client.Disconnect()

	<-clientDisconnected

	waitEnd = make(chan struct{})

	// Manually reconnect
	go client.Connect()

	select {
	case <-waitEnd:
	case <-time.After(time.Second * 3):
		t.Fatal("no join message received 2")
	}

	// Server received second JOIN message
	assertStringsEqual(t, "JOIN #gempir", receivedMsg)
}

func TestCapabilities(t *testing.T) {
	type testTable struct {
		name     string
		in       []string
		expected string
	}
	var tests = []testTable{
		{
			"Default Capabilities (not modifying)",
			nil,
			"CAP REQ :" + strings.Join([]string{TagsCapability, CommandsCapability, MembershipCapability}, " "),
		},
		{
			"Modified Capabilities",
			[]string{CommandsCapability, MembershipCapability},
			"CAP REQ :" + strings.Join([]string{CommandsCapability, MembershipCapability}, " "),
		},
	}

	for _, tt := range tests {
		func(tt testTable) {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				waitRecv := make(chan struct{})
				waitServerConnect := make(chan struct{})
				waitClientConnect := make(chan struct{})

				var received string

				server := startServer2(t, closeOnConnect(waitServerConnect), func(message string) {
					if strings.HasPrefix(message, "CAP REQ") {
						received = message
						close(waitRecv)
					}
				})

				client := newTestClient(server.host)
				if tt.in != nil {
					client.Capabilities = tt.in
				}
				client.OnConnect(clientCloseOnConnect(waitClientConnect))
				clientDisconnected := connectAndEnsureGoodDisconnect(t, client)

				// Wait for correct password to be read in server
				if !waitWithTimeout(waitRecv) {
					t.Fatal("no oauth read")
				}

				assertStringsEqual(t, tt.expected, received)

				// Wait for server to acknowledge connection
				if !waitWithTimeout(waitServerConnect) {
					t.Fatal("no successful connection")
				}

				// Wait for client to acknowledge connection
				if !waitWithTimeout(waitClientConnect) {
					t.Fatal("no successful connection")
				}

				// Disconnect client from server
				err := client.Disconnect()
				if err != nil {
					t.Error("Error during disconnect:" + err.Error())
				}

				// Wait for client to be fully disconnected
				<-clientDisconnected

				// Wait for server to be fully disconnected
				<-server.stopped
			})
		}(tt)
	}
}

func TestEmptyCapabilities(t *testing.T) {
	type testTable struct {
		name string
		in   []string
	}
	var tests = []testTable{
		{"nil", nil},
		{"Empty list", []string{}},
	}

	for _, tt := range tests {
		func(tt testTable) {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				// we will modify the clients caps to only send commands and membership
				receivedCapabilities := false
				waitRecv := make(chan struct{})
				waitServerConnect := make(chan struct{})
				waitClientConnect := make(chan struct{})

				server := startServer2(t, closeOnConnect(waitServerConnect), func(message string) {
					if strings.HasPrefix(message, "CAP REQ") {
						receivedCapabilities = true
					} else if strings.HasPrefix(message, "PASS") {
						close(waitRecv)
					}
				})

				client := newTestClient(server.host)
				client.Capabilities = tt.in
				client.OnConnect(clientCloseOnConnect(waitClientConnect))
				clientDisconnected := connectAndEnsureGoodDisconnect(t, client)

				// Wait for correct password to be read in server
				if !waitWithTimeout(waitRecv) {
					t.Fatal("no oauth read")
				}

				assertFalse(t, receivedCapabilities, "We should NOT have received caps since we sent an empty list of caps")

				// Wait for server to acknowledge connection
				if !waitWithTimeout(waitServerConnect) {
					t.Fatal("no successful connection")
				}

				// Wait for client to acknowledge connection
				if !waitWithTimeout(waitClientConnect) {
					t.Fatal("no successful connection")
				}

				// Disconnect client from server
				err := client.Disconnect()
				if err != nil {
					t.Error("Error during disconnect:" + err.Error())
				}

				// Wait for client to be fully disconnected
				<-clientDisconnected

				// Wait for server to be fully disconnected
				<-server.stopped
			})
		}(tt)
	}
}
