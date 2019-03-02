package twitch

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
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
var startPort = 3000

func newPort() (r int) {
	startPortMutex.Lock()
	r = startPort
	startPort++
	startPortMutex.Unlock()
	return
}

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

func connectAndEnsureGoodDisconnect(t *testing.T, client *Client) {
	go func() {
		err := client.Connect()
		assertErrorsEqual(t, ErrClientDisconnected, err)
	}()
}

func handleTestConnection(t *testing.T, onConnect func(net.Conn), onMessage func(string), listener net.Listener, wg *sync.WaitGroup) {
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
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

func startServer(t *testing.T, onConnect func(net.Conn), onMessage func(string)) string {
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
	wg.Add(1)
	go handleTestConnection(t, onConnect, onMessage, listener, &wg)

	go func() {
		wg.Wait()
		listener.Close()
	}()

	return host
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

	if reflect.TypeOf(*client) != reflect.TypeOf(Client{}) {
		t.Error("client is not of type Client")
	}
}

func TestCanConnectAndAuthenticate(t *testing.T) {
	t.Parallel()
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
	connectAndEnsureGoodDisconnect(t, client)
	defer client.Disconnect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no oauth read")
	}

	assertStringsEqual(t, "PASS "+oauthCode, received)
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
	t.Parallel()
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
	t.Parallel()
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

	assertStringsEqual(t, "ampzyh was timed out for 1: testing xd", received)
}

func TestCanReceiveROOMSTATEMessage(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestCanReceiveUSERNOTICEMessageResub(t *testing.T) {
	t.Parallel()
	testMessage := `@badges=moderator/1,subscriber/24;color=#1FD2FF;display-name=Karl_Kons;emotes=28087:0-6;flags=;id=7c95beea-a7ac-4c10-9e0a-d7dbf163c038;login=karl_kons;mod=1;msg-id=resub;msg-param-months=34;msg-param-sub-plan-name=look\sat\sthose\sshitty\semotes,\srip\s$5\sLUL;msg-param-sub-plan=1000;room-id=11148817;subscriber=1;system-msg=Karl_Kons\sjust\ssubscribed\swith\sa\sTier\s1\ssub.\sKarl_Kons\ssubscribed\sfor\s34\smonths\sin\sa\srow!;tmi-sent-ts=1540140252828;turbo=0;user-id=68706331;user-type=mod :tmi.twitch.tv USERNOTICE #pajlada :WutFace`

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

	assertStringsEqual(t, "34", received)
}

func checkNoticeMessage(t *testing.T, testMessage string, requirements map[string]string) {
	received := map[string]string{}
	wait := make(chan struct{})

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNewNoticeMessage(func(channel string, user User, message Message) {
		received["msg-id"] = message.Tags["msg-id"]
		received["channel"] = channel
		received["text"] = message.Text
		received["raw"] = message.Raw
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

func TestCanReceiveJOINMessage(t *testing.T) {
	t.Parallel()
	testMessage := `:username123!username123@username123.tmi.twitch.tv JOIN #mychannel`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserJoin(func(channel, user string) {
		received = user
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "username123", received)
}

func TestCanReceivePARTMessage(t *testing.T) {
	t.Parallel()
	testMessage := `:username123!username123@username123.tmi.twitch.tv PART #mychannel`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnUserPart(func(channel, user string) {
		received = user
		close(wait)
	})

	go client.Connect()

	// wait for server to start
	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, "username123", received)
}

func TestCanReceiveUNSETMessage(t *testing.T) {
	t.Parallel()
	testMessage := `@badges=moderator/1,subscriber/24;color=#1FD2FF;display-name=Karl_Kons;emotes=28087:0-6;flags=;id=7c95beea-a7ac-4c10-9e0a-d7dbf163c038;login=karl_kons;mod=1;msg-id=resub;msg-param-months=34;msg-param-sub-plan-name=look\sat\sthose\sshitty\semotes,\srip\s$5\sLUL;msg-param-sub-plan=1000;room-id=11148817;subscriber=1;system-msg=Karl_Kons\sjust\ssubscribed\swith\sa\sTier\s1\ssub.\sKarl_Kons\ssubscribed\sfor\s34\smonths\sin\sa\srow!;tmi-sent-ts=1540140252828;turbo=0;user-id=68706331;user-type=mod :tmi.twitch.tv MALFORMEDMESSAGETYPETHISWILLBEUNSET #pajlada :WutFace`

	wait := make(chan struct{})
	var received string

	host := startServer(t, postMessageOnConnect(testMessage), nothingOnMessage)
	client := newTestClient(host)

	client.OnNewUnsetMessage(func(rawMessage string) {
		received = rawMessage
		close(wait)
	})

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("no message sent")
	}

	assertStringsEqual(t, testMessage, received)
}

func TestCanHandleRECONNECTMessage(t *testing.T) {
	t.Parallel()
	const testMessage = ":tmi.twitch.tv RECONNECT"

	wait := make(chan bool)

	var connCount int32

	host := startServerMultiConns(t, 2, func(conn net.Conn) {
		atomic.AddInt32(&connCount, 1)
		wait <- true
		time.AfterFunc(100*time.Millisecond, func() {
			fmt.Fprintf(conn, "%s\r\n", testMessage)
		})
	}, nothingOnMessage)
	client := newTestClient(host)

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
}

func TestCanSayMessage(t *testing.T) {
	t.Parallel()
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

	assertStringsEqual(t, "PRIVMSG #jtv :/w gempir "+testMessage, receivedMsg)
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
	testString := `:justinfan123123.tmi.twitch.tv 353 justinfan123123 = #channel123 :username1 username2`
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #anythingbutchannel123 :ok go now"
	waitEnd := make(chan struct{})

	host := startServer(t, func(conn net.Conn) {
		fmt.Fprintf(conn, "%s\r\n", testString)
		fmt.Fprintf(conn, "%s\r\n", testMessage)
	}, nothingOnMessage)

	client := newTestClient(host)

	client.Join("channel123")

	client.OnNewMessage(func(channel string, user User, message Message) {
		if message.Text == "ok go now" {
			// test a valid channel
			got, err := client.Userlist("channel123")
			if err != nil {
				t.Fatal("error not nil for client.Userlist")
			}
			expected := []string{"username1", "username2"}

			sort.Strings(got)

			assertStringSlicesEqual(t, expected, got)

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

func TestSendReturnsIfBufferIsFull(t *testing.T) {
	t.Parallel()
	client := newTestClient("127.0.0.1:5")

	for i := 0; i < WriteBufferSize; i++ {
		success := client.send("Pepega")
		assertTrue(t, success, "Send must succeed in storing messages up to its max buffer size")
	}

	success := client.send("Pepega")
	assertFalse(t, success, "Send must not be able to add messages above its buffer")
}

func TestLocalSendBuffer(t *testing.T) {
	t.Parallel()
	const numNumbersToSend = 250

	wait := make(chan bool)

	var connMutex sync.Mutex
	var conn net.Conn

	host := startServerMultiConnsNoTLS(t, numNumbersToSend/10, func(c net.Conn) {
		connMutex.Lock()
		defer connMutex.Unlock()
		conn = c
	}, func(message string) {
		if len(message) < 3 {
			receivedNumber, err := strconv.Atoi(message)
			assertErrorsEqual(t, nil, err)

			if receivedNumber%10 == 0 {
				connMutex.Lock()
				defer connMutex.Unlock()
				conn.Close()
				return
			}
		}
	})
	client := newTestClient(host)
	client.TLS = false

	go func() {
		// sends messages 0 to 10 with a 250 ms delay
		// we should be able to reasonably expect at least 50% of them to make it through
		for i := 0; i < numNumbersToSend; i++ {
			client.send(strconv.Itoa(i))
		}

		// Wait for the full buffer to be sent
		for client.sendBufferLength() > 0 {
			time.Sleep(10 * time.Millisecond)
		}
		close(wait)
	}()

	go client.Connect()

	select {
	case <-wait:
	case <-time.After(time.Second * 3):
		t.Fatal("Did not establish a connection")
	}

	client.Disconnect()
}

type WriteOnlyClient struct {
	*Client
}

func (wo *WriteOnlyClient) startReader(reader io.Reader, wg *sync.WaitGroup) {
	wg.Done()
}

func (wo *WriteOnlyClient) startPinger(closer io.Closer, wg *sync.WaitGroup) {
	wg.Done()
}

func TestWriter(t *testing.T) {
	t.Parallel()
	const numNumbersToSend = 250

	wait := make(chan bool)

	var connMutex sync.Mutex
	var conn net.Conn

	host := startServerMultiConnsNoTLS(t, numNumbersToSend/10, func(c net.Conn) {
		connMutex.Lock()
		defer connMutex.Unlock()
		conn = c
	}, func(message string) {
		if len(message) < 3 {
			receivedNumber, err := strconv.Atoi(message)
			assertErrorsEqual(t, nil, err)

			if receivedNumber%10 == 0 {
				connMutex.Lock()
				defer connMutex.Unlock()
				conn.Close()
				return
			}
		}
	})
	client := WriteOnlyClient{newTestClient(host)}
	client.TLS = false

	go func() {
		// sends messages 0 to 10 with a 250 ms delay
		// we should be able to reasonably expect at least 50% of them to make it through
		for i := 0; i < numNumbersToSend; i++ {
			client.send(strconv.Itoa(i))
		}

		// Wait for the full buffer to be sent
		for client.sendBufferLength() > 0 {
			time.Sleep(10 * time.Millisecond)
		}
		close(wait)
	}()

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

	client.OnPongReceived(func() {
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
