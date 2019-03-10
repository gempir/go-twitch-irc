package twitch

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// ircTwitch constant for twitch irc chat address
	ircTwitchTLS = "irc.chat.twitch.tv:6697"
	ircTwitch    = "irc.chat.twitch.tv:6667"

	pingSignature       = "go-twitch-irc"
	pingMessage         = "PING :" + pingSignature
	expectedPongMessage = ":tmi.twitch.tv PONG tmi.twitch.tv :" + pingSignature
)

var (
	// ErrClientDisconnected returned from Connect() when a Disconnect() was called
	ErrClientDisconnected = errors.New("client called Disconnect()")

	// ErrLoginAuthenticationFailed returned from Connect() when either the wrong or a malformed oauth token is used
	ErrLoginAuthenticationFailed = errors.New("login authentication failed")

	// ErrConnectionIsNotOpen is returned by Disconnect in case you call it without being connected
	ErrConnectionIsNotOpen = errors.New("connection is not open")

	// WriteBufferSize can be modified to change the write channel buffer size. Must be configured before NewClient is called to take effect
	WriteBufferSize = 512

	// ReadBufferSize can be modified to change the read channel buffer size. Must be configured before NewClient is called to take effect
	ReadBufferSize = 64
)

// Internal errors
var (
	errReconnect = errors.New("reconnect")
)

// User data you receive from TMI
type User struct {
	ID          string
	Name        string
	DisplayName string
	Color       string
	Badges      map[string]int
}

// RawMessage data you receive from TMI
type RawMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
}

// WhisperMessage data you receive from WHISPER message type
type WhisperMessage struct {
	Raw       string
	Type      MessageType
	RawType   string
	Tags      map[string]string
	Message   string
	Target    string
	MessageID string
	ThreadID  string
	Emotes    []*Emote
	Action    bool
}

// PrivateMessage data you receive from PRIVMSG message type
type PrivateMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
	Channel string
	RoomID  string
	ID      string
	Time    time.Time
	Emotes  []*Emote
	Bits    int
	Action  bool
}

// ClearChatMessage data you receive from CLEARCHAT message type
type ClearChatMessage struct {
	Raw            string
	Type           MessageType
	RawType        string
	Tags           map[string]string
	Message        string
	Channel        string
	RoomID         string
	Time           time.Time
	BanDuration    int
	TargetUserID   string
	TargetUsername string
}

// RoomStateMessage data you receive from ROOMSTATE message type
type RoomStateMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
	Channel string
	RoomID  string
	State   map[string]int
}

// UserNoticeMessage  data you receive from USERNOTICE message type
type UserNoticeMessage struct {
	Raw       string
	Type      MessageType
	RawType   string
	Tags      map[string]string
	Message   string
	Channel   string
	RoomID    string
	ID        string
	Time      time.Time
	Emotes    []*Emote
	MsgID     string
	MsgParams map[string]string
	SystemMsg string
}

// UserStateMessage data you receive from the USERSTATE message type
type UserStateMessage struct {
	Raw       string
	Type      MessageType
	RawType   string
	Tags      map[string]string
	Message   string
	Channel   string
	EmoteSets []string
}

// NoticeMessage data you receive from the NOTICE message type
type NoticeMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
	Channel string
	MsgID   string
}

// Client client to control your connection and attach callbacks
type Client struct {
	IrcAddress             string
	ircUser                string
	ircToken               string
	TLS                    bool
	connActive             tAtomBool
	channels               map[string]bool
	channelUserlistMutex   *sync.RWMutex
	channelUserlist        map[string]map[string]bool
	channelsMtx            *sync.RWMutex
	onConnect              func()
	onNewWhisper           func(user User, message WhisperMessage)
	onNewMessage           func(user User, message PrivateMessage)
	onNewClearChatMessage  func(message ClearChatMessage)
	onNewRoomStateMessage  func(message RoomStateMessage)
	onNewUserNoticeMessage func(user User, message UserNoticeMessage)
	onNewUserStateMessage  func(user User, message UserStateMessage)
	onNewNoticeMessage     func(message NoticeMessage)
	onUserJoin             func(channel, user string)
	onUserPart             func(channel, user string)
	onNewUnsetMessage      func(message RawMessage)

	onPingSent     func()
	onPongReceived func()

	// read is the incoming messages channel, normally buffered with ReadBufferSize
	read chan (string)

	// write is the outgoing messages channel, normally buffered with WriteBufferSize
	write chan (string)

	// clientReconnect is closed whenever the client needs to reconnect for connection issue reasons
	clientReconnect chanCloser

	// userDisconnect is closed when the user calls Disconnect
	userDisconnect chanCloser

	// pongReceived is listened to by the pinger go-routine after it has sent off a ping. will be triggered by handleLine
	pongReceived chan bool

	// messageReceived is listened to by the pinger go-routine to interrupt the idle ping interval
	messageReceived chan bool

	// Option whether to send pings every `IdlePingInterval`. The IdlePingInterval is interrupted every time a message is received from the irc server
	// The variable may only be modified before calling Connect
	SendPings bool

	// IdlePingInterval is the interval at which to send a ping to the irc server to ensure the connection is alive.
	// The variable may only be modified before calling Connect
	IdlePingInterval time.Duration

	// PongTimeout is the time go-twitch-irc waits after sending a ping before issuing a reconnect
	// The variable may only be modified before calling Connect
	PongTimeout time.Duration

	// SetupCmd is the command that is ran on successful connection to Twitch. Useful if you are proxying or something to run a custom command on connect.
	// The variable must be modified before calling Connect or the command will not run.
	SetupCmd string
}

// NewClient to create a new client
func NewClient(username, oauth string) *Client {
	return &Client{
		ircUser:         username,
		ircToken:        oauth,
		TLS:             true,
		channels:        map[string]bool{},
		channelUserlist: map[string]map[string]bool{},
		channelsMtx:     &sync.RWMutex{},
		messageReceived: make(chan bool),

		read:  make(chan string, ReadBufferSize),
		write: make(chan string, WriteBufferSize),

		// NOTE: IdlePingInterval must be higher than PongTimeout
		SendPings:        true,
		IdlePingInterval: time.Second * 15,
		PongTimeout:      time.Second * 5,

		channelUserlistMutex: &sync.RWMutex{},
	}
}

// OnConnect attach callback to when a connection has been established
func (c *Client) OnConnect(callback func()) {
	c.onConnect = callback
}

// OnNewWhisper attach callback to new whisper
func (c *Client) OnNewWhisper(callback func(user User, message WhisperMessage)) {
	c.onNewWhisper = callback
}

// OnNewMessage attach callback to new standard chat messages
func (c *Client) OnNewMessage(callback func(user User, message PrivateMessage)) {
	c.onNewMessage = callback
}

// OnNewClearChatMessage attach callback to new messages such as timeouts
func (c *Client) OnNewClearChatMessage(callback func(message ClearChatMessage)) {
	c.onNewClearChatMessage = callback
}

// OnNewRoomStateMessage attach callback to new messages such as submode enabled
func (c *Client) OnNewRoomStateMessage(callback func(message RoomStateMessage)) {
	c.onNewRoomStateMessage = callback
}

// OnNewUserNoticeMessage attach callback to new usernotice message such as sub, resub, and raids
func (c *Client) OnNewUserNoticeMessage(callback func(user User, message UserNoticeMessage)) {
	c.onNewUserNoticeMessage = callback
}

// OnNewUserStateMessage attach callback to new userstate
func (c *Client) OnNewUserStateMessage(callback func(user User, message UserStateMessage)) {
	c.onNewUserStateMessage = callback
}

// OnNewNoticeMessage attach callback to new notice message such as hosts
func (c *Client) OnNewNoticeMessage(callback func(message NoticeMessage)) {
	c.onNewNoticeMessage = callback
}

// OnUserJoin attaches callback to user joins
func (c *Client) OnUserJoin(callback func(channel, user string)) {
	c.onUserJoin = callback
}

// OnUserPart attaches callback to user parts
func (c *Client) OnUserPart(callback func(channel, user string)) {
	c.onUserPart = callback
}

// OnNewUnsetMessage attaches callback to message types we currently don't support
func (c *Client) OnNewUnsetMessage(callback func(message RawMessage)) {
	c.onNewUnsetMessage = callback
}

// OnPingSent attaches callback that's called whenever the client sends out a ping message
func (c *Client) OnPingSent(callback func()) {
	c.onPingSent = callback
}

// OnPongReceived attaches callback that's called whenever the client receives a pong to one of its previously sent out ping messages
func (c *Client) OnPongReceived(callback func()) {
	c.onPongReceived = callback
}

// Say write something in a chat
func (c *Client) Say(channel, text string) {
	channel = strings.ToLower(channel)

	c.send(fmt.Sprintf("PRIVMSG #%s :%s", channel, text))
}

// Whisper write something in private to someone on twitch
// whispers are heavily spam protected
// so your message might get blocked because of this
// verify your bot to prevent this
func (c *Client) Whisper(username, text string) {
	c.send(fmt.Sprintf("PRIVMSG #jtv :/w %s %s", username, text))
}

// Join enter a twitch channel to read more messages
func (c *Client) Join(channel string) {
	channel = strings.ToLower(channel)

	// If we don't have the channel in our map AND we have an
	// active connection, explicitly join before we add it to our map
	c.channelsMtx.Lock()
	if !c.channels[channel] && c.connActive.get() {
		go c.send(fmt.Sprintf("JOIN #%s", channel))
	}

	c.channels[channel] = true
	c.channelUserlistMutex.Lock()
	c.channelUserlist[channel] = map[string]bool{}
	c.channelUserlistMutex.Unlock()
	c.channelsMtx.Unlock()
}

// Depart leave a twitch channel
func (c *Client) Depart(channel string) {
	if c.connActive.get() {
		go c.send(fmt.Sprintf("PART #%s", channel))
	}

	c.channelsMtx.Lock()
	delete(c.channels, channel)
	c.channelUserlistMutex.Lock()
	delete(c.channelUserlist, channel)
	c.channelUserlistMutex.Unlock()
	c.channelsMtx.Unlock()
}

// Disconnect close current connection
func (c *Client) Disconnect() error {
	if !c.connActive.get() {
		return ErrConnectionIsNotOpen
	}

	c.userDisconnect.Close()

	return nil
}

// Connect connect the client to the irc server
func (c *Client) Connect() error {
	if c.IrcAddress == "" && c.TLS {
		c.IrcAddress = ircTwitchTLS
	} else if c.IrcAddress == "" && !c.TLS {
		c.IrcAddress = ircTwitch
	}

	dialer := &net.Dialer{
		KeepAlive: time.Second * 10,
	}

	var conf *tls.Config
	// This means we are connecting to "localhost". Disable certificate chain check
	if strings.HasPrefix(c.IrcAddress, "127.0.0.1:") {
		conf = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else {
		conf = &tls.Config{}
	}

	for {
		err := c.makeConnection(dialer, conf)

		switch err {
		case errReconnect:
			continue

		default:
			return err
		}
	}
}

func (c *Client) makeConnection(dialer *net.Dialer, conf *tls.Config) (err error) {
	var conn net.Conn
	if c.TLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", c.IrcAddress, conf)
	} else {
		conn, err = dialer.Dial("tcp", c.IrcAddress)
	}
	if err != nil {
		return
	}

	wg := sync.WaitGroup{}
	c.clientReconnect.Reset()
	c.userDisconnect.Reset()

	// Start the connection reader in a separate go-routine
	wg.Add(1)
	go c.startReader(conn, &wg)

	if c.SendPings {
		// If SendPings is true (which it is by default), start the thread
		// responsible for managing sending pings and reading pongs
		// in a separate go-routine
		wg.Add(1)
		c.startPinger(conn, &wg)
	}

	// Send the initial connection messages (like logging in, getting the CAP REQ stuff)
	c.setupConnection(conn)

	// Start the connection writer in a separate go-routine
	wg.Add(1)
	go c.startWriter(conn, &wg)

	// start the parser in the same go-routine as makeConnection was called from
	// the error returned from parser will be forwarded to the caller of makeConnection
	// and that error will decide whether or not to reconnect
	err = c.startParser()

	conn.Close()
	c.clientReconnect.Close()

	// Wait for the reader, pinger, and writer to close
	wg.Wait()

	return
}

// Userlist returns the userlist for a given channel
func (c *Client) Userlist(channel string) ([]string, error) {
	c.channelUserlistMutex.RLock()
	defer c.channelUserlistMutex.RUnlock()
	usermap, ok := c.channelUserlist[channel]
	if !ok || usermap == nil {
		return nil, fmt.Errorf("Could not find userlist for channel '%s' in client", channel)
	}
	userlist := make([]string, len(usermap))

	i := 0
	for key := range usermap {
		userlist[i] = key
		i++
	}

	return userlist, nil
}

// SetIRCToken updates the oauth token for this client used for authentication
// This will not cause a reconnect, but is meant more for "on next connect, use this new token" in case the old token has expired
func (c *Client) SetIRCToken(ircToken string) {
	c.ircToken = ircToken
}

func (c *Client) startReader(reader io.Reader, wg *sync.WaitGroup) {
	defer func() {
		c.clientReconnect.Close()

		wg.Done()
	}()

	tp := textproto.NewReader(bufio.NewReader(reader))

	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		messages := strings.Split(line, "\r\n")
		for _, msg := range messages {
			if !c.connActive.get() && strings.Contains(msg, ":tmi.twitch.tv 001") {
				c.connActive.set(true)
				c.initialJoins()
				if c.onConnect != nil {
					c.onConnect()
				}
			}
			c.read <- msg
		}
	}
}

func (c *Client) startPinger(closer io.Closer, wg *sync.WaitGroup) {
	c.pongReceived = make(chan bool, 1)

	go func() {
		defer func() {
			wg.Done()
		}()

		for {
			select {
			case <-c.clientReconnect.channel:
				return

			case <-c.userDisconnect.channel:
				return

			case <-c.messageReceived:
				// Interrupt idle ping interval
				continue

			case <-time.After(c.IdlePingInterval):
				if c.onPingSent != nil {
					c.onPingSent()
				}
				c.send(pingMessage)

				select {
				case <-c.pongReceived:
					// Received pong message within the time limit, we're good
					if c.onPongReceived != nil {
						c.onPongReceived()
					}
					continue

				case <-time.After(c.PongTimeout):
					// No pong message was received within the pong timeout, disconnect
					c.clientReconnect.Close()
					closer.Close()
				}
			}
		}
	}()
}

func (c *Client) setupConnection(conn net.Conn) {
	if c.SetupCmd != "" {
		conn.Write([]byte(c.SetupCmd + "\r\n"))
	}
	conn.Write([]byte("PASS " + c.ircToken + "\r\n"))
	conn.Write([]byte("NICK " + c.ircUser + "\r\n"))
	conn.Write([]byte("CAP REQ :twitch.tv/tags\r\n"))
	conn.Write([]byte("CAP REQ :twitch.tv/commands\r\n"))
	conn.Write([]byte("CAP REQ :twitch.tv/membership\r\n"))
}

func (c *Client) startWriter(writer io.WriteCloser, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	for {
		select {
		case <-c.clientReconnect.channel:
			return

		case <-c.userDisconnect.channel:
			return

		case msg := <-c.write:
			_, err := writer.Write([]byte(msg + "\r\n"))
			if err != nil {
				// Attempt to re-send failed messages
				c.write <- msg

				writer.Close()
				c.clientReconnect.Close()
				return
			}
		}
	}
}

func (c *Client) startParser() error {
	for {
		// reader
		select {
		case msg := <-c.read:
			if err := c.handleLine(msg); err != nil {
				return err
			}

		case <-c.clientReconnect.channel:
			return errReconnect

		case <-c.userDisconnect.channel:
			return ErrClientDisconnected
		}
	}
}

func (c *Client) initialJoins() {
	// join or rejoin channels on connection
	c.channelsMtx.RLock()
	for channel := range c.channels {
		c.send(fmt.Sprintf("JOIN #%s", channel))
	}
	c.channelsMtx.RUnlock()
}

func (c *Client) send(line string) bool {
	select {
	case c.write <- line:
		return true
	default:
		return false
	}
}

// Returns how many messages are left in the send buffer. Only used in tests
func (c *Client) sendBufferLength() int {
	return len(c.write)
}

// Errors returned from handleLine break out of readConnections, which starts a reconnect
// This means that we should only return fatal errors as errors here
func (c *Client) handleLine(line string) error {
	go func() {
		// Send a message on the `messageReceived` channel, but do not block in case no one is receiving on the other end
		select {
		case c.messageReceived <- true:
		default:
		}
	}()

	// Handle PING
	if strings.HasPrefix(line, "PING") {
		c.send(strings.Replace(line, "PING", "PONG", 1))

		return nil
	}

	// Handle PONG
	if line == expectedPongMessage {
		// Received a pong that was sent by us
		select {
		case c.pongReceived <- true:
		default:
		}

		return nil
	}

	if strings.HasPrefix(line, "@") {
		user, message := ParseMessage(line)

		switch message.(type) {
		case *WhisperMessage:
			if c.onNewWhisper != nil {
				c.onNewWhisper(*user, *message.(*WhisperMessage))
			}
		case *PrivateMessage:
			if c.onNewMessage != nil {
				c.onNewMessage(*user, *message.(*PrivateMessage))
			}
		case *ClearChatMessage:
			if c.onNewClearChatMessage != nil {
				c.onNewClearChatMessage(*message.(*ClearChatMessage))
			}
		case *RoomStateMessage:
			if c.onNewRoomStateMessage != nil {
				c.onNewRoomStateMessage(*message.(*RoomStateMessage))
			}
		case *UserNoticeMessage:
			if c.onNewUserNoticeMessage != nil {
				c.onNewUserNoticeMessage(*user, *message.(*UserNoticeMessage))
			}
		case *UserStateMessage:
			if c.onNewUserStateMessage != nil {
				c.onNewUserStateMessage(*user, *message.(*UserStateMessage))
			}
		case *NoticeMessage:
			if c.onNewNoticeMessage != nil {
				c.onNewNoticeMessage(*message.(*NoticeMessage))
			}
		case *RawMessage:
			if c.onNewUnsetMessage != nil {
				c.onNewUnsetMessage(*message.(*RawMessage))
			}
		}

		return nil
	}

	if strings.HasPrefix(line, ":") {
		if strings.Contains(line, "tmi.twitch.tv JOIN") {
			channel, username := parseJoinPart(line)

			c.channelUserlistMutex.Lock()
			if c.channelUserlist[channel] == nil {
				c.channelUserlist[channel] = map[string]bool{}
			}

			_, ok := c.channelUserlist[channel][username]
			if !ok && username != c.ircUser {
				c.channelUserlist[channel][username] = true
			}
			c.channelUserlistMutex.Unlock()

			if c.onUserJoin != nil {
				c.onUserJoin(channel, username)
			}
		}
		if strings.Contains(line, "tmi.twitch.tv PART") {
			channel, username := parseJoinPart(line)

			c.channelUserlistMutex.Lock()
			delete(c.channelUserlist[channel], username)
			c.channelUserlistMutex.Unlock()

			if c.onUserPart != nil {
				c.onUserPart(channel, username)
			}
		}
		if strings.Contains(line, "tmi.twitch.tv RECONNECT") {
			// https://dev.twitch.tv/docs/irc/commands/#reconnect-twitch-commands
			return errReconnect
		}
		if strings.Contains(line, "353 "+c.ircUser) {
			channel, users := parseNames(line)

			c.channelUserlistMutex.Lock()
			for _, user := range users {
				c.channelUserlist[channel][user] = true
			}
			c.channelUserlistMutex.Unlock()
		}
		if strings.Contains(line, "tmi.twitch.tv NOTICE * :Login authentication failed") ||
			strings.Contains(line, "tmi.twitch.tv NOTICE * :Improperly formatted auth") ||
			line == ":tmi.twitch.tv NOTICE * :Invalid NICK" {
			return ErrLoginAuthenticationFailed
		}
	}

	return nil
}

// tAtomBool atomic bool for writing/reading across threads
type tAtomBool struct{ flag int32 }

func (b *tAtomBool) set(value bool) {
	var i int32
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), int32(i))
}

func (b *tAtomBool) get() bool {
	if atomic.LoadInt32(&(b.flag)) != 0 {
		return true
	}
	return false
}

// chanCloser is a helper function for abusing channels for notifications
// this is an easy "notify many" channel
type chanCloser struct {
	mutex sync.Mutex

	o       *sync.Once
	channel chan (struct{})
}

func (c *chanCloser) Reset() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.o = &sync.Once{}
	c.channel = make(chan (struct{}))
}

func (c *chanCloser) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.o.Do(func() {
		close(c.channel)
	})
}
