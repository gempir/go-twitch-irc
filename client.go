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

	// WriteBufferSize can be modified to change the write channel buffer size.
	// Must be configured before NewClient is called to take effect
	WriteBufferSize = 512

	// ReadBufferSize can be modified to change the read channel buffer size.
	// Must be configured before NewClient is called to take effect
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

// Message interface that all messages implement
type Message interface {
	GetType() MessageType
}

// RawMessage data you receive from TMI
type RawMessage struct {
	Raw     string
	Type    MessageType
	RawType string
	Tags    map[string]string
	Message string
}

// GetType implements the Message interface, and returns this message's type
func (msg *RawMessage) GetType() MessageType {
	return msg.Type
}

// WhisperMessage data you receive from WHISPER message type
type WhisperMessage struct {
	User User

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

// GetType implements the Message interface, and returns this message's type
func (msg *WhisperMessage) GetType() MessageType {
	return msg.Type
}

// PrivateMessage data you receive from PRIVMSG message type
type PrivateMessage struct {
	User User

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

// GetType implements the Message interface, and returns this message's type
func (msg *PrivateMessage) GetType() MessageType {
	return msg.Type
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

// GetType implements the Message interface, and returns this message's type
func (msg *ClearChatMessage) GetType() MessageType {
	return msg.Type
}

// ClearMessage data you receive from CLEARMSG message type
type ClearMessage struct {
	Raw         string
	Type        MessageType
	RawType     string
	Tags        map[string]string
	Message     string
	Channel     string
	Login       string
	TargetMsgID string
}

// GetType implements the Message interface, and returns this message's type
func (msg *ClearMessage) GetType() MessageType {
	return msg.Type
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

// GetType implements the Message interface, and returns this message's type
func (msg *RoomStateMessage) GetType() MessageType {
	return msg.Type
}

// UserNoticeMessage  data you receive from USERNOTICE message type
type UserNoticeMessage struct {
	User User

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

// GetType implements the Message interface, and returns this message's type
func (msg *UserNoticeMessage) GetType() MessageType {
	return msg.Type
}

// UserStateMessage data you receive from the USERSTATE message type
type UserStateMessage struct {
	User User

	Raw       string
	Type      MessageType
	RawType   string
	Tags      map[string]string
	Message   string
	Channel   string
	EmoteSets []string
}

// GetType implements the Message interface, and returns this message's type
func (msg *UserStateMessage) GetType() MessageType {
	return msg.Type
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

// GetType implements the Message interface, and returns this message's type
func (msg *NoticeMessage) GetType() MessageType {
	return msg.Type
}

// UserJoinMessage desJoines the message that is sent whenever a user joins a channel we're connected to
// See https://dev.twitch.tv/docs/irc/membership/#join-twitch-membership
type UserJoinMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	// Channel name
	Channel string

	// User name
	User string
}

// GetType implements the Message interface, and returns this message's type
func (msg *UserJoinMessage) GetType() MessageType {
	return msg.Type
}

// UserPartMessage describes the message that is sent whenever a user leaves a channel we're connected to
// See https://dev.twitch.tv/docs/irc/membership/#part-twitch-membership
type UserPartMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	// Channel name
	Channel string

	// User name
	User string
}

// GetType implements the Message interface, and returns this message's type
func (msg *UserPartMessage) GetType() MessageType {
	return msg.Type
}

// ReconnectMessage describes the
type ReconnectMessage struct {
	Raw     string
	Type    MessageType
	RawType string
}

// GetType implements the Message interface, and returns this message's type
func (msg *ReconnectMessage) GetType() MessageType {
	return msg.Type
}

// NamesMessage describes the data posted in response to a /names command
// See https://www.alien.net.au/irc/irc2numerics.html#353
type NamesMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	// Channel name
	Channel string

	// List of user names
	Users []string
}

// GetType implements the Message interface, and returns this message's type
func (msg *NamesMessage) GetType() MessageType {
	return msg.Type
}

// PingMessage describes an IRC PING message
type PingMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	Message string
}

// GetType implements the Message interface, and returns this message's type
func (msg *PingMessage) GetType() MessageType {
	return msg.Type
}

// PongMessage describes an IRC PONG message
type PongMessage struct {
	Raw     string
	Type    MessageType
	RawType string

	Message string
}

// GetType implements the Message interface, and returns this message's type
func (msg *PongMessage) GetType() MessageType {
	return msg.Type
}

// Client client to control your connection and attach callbacks
type Client struct {
	IrcAddress           string
	ircUser              string
	ircToken             string
	TLS                  bool
	connActive           tAtomBool
	channels             map[string]bool
	channelUserlistMutex *sync.RWMutex
	channelUserlist      map[string]map[string]bool
	channelsMtx          *sync.RWMutex
	onConnect            func()
	onWhisperMessage     func(message WhisperMessage)
	onPrivateMessage     func(message PrivateMessage)
	onClearChatMessage   func(message ClearChatMessage)
	onRoomStateMessage   func(message RoomStateMessage)
	onClearMessage       func(message ClearMessage)
	onUserNoticeMessage  func(message UserNoticeMessage)
	onUserStateMessage   func(message UserStateMessage)
	onNoticeMessage      func(message NoticeMessage)
	onUserJoinMessage    func(message UserJoinMessage)
	onUserPartMessage    func(message UserPartMessage)
	onReconnectMessage   func(message ReconnectMessage)
	onNamesMessage       func(message NamesMessage)
	onPingMessage        func(message PingMessage)
	onPongMessage        func(message PongMessage)
	onUnsetMessage       func(message RawMessage)

	onPingSent func()

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

// OnWhisperMessage attach callback to new whisper
func (c *Client) OnWhisperMessage(callback func(message WhisperMessage)) {
	c.onWhisperMessage = callback
}

// OnPrivateMessage attach callback to new standard chat messages
func (c *Client) OnPrivateMessage(callback func(message PrivateMessage)) {
	c.onPrivateMessage = callback
}

// OnClearChatMessage attach callback to new messages such as timeouts
func (c *Client) OnClearChatMessage(callback func(message ClearChatMessage)) {
	c.onClearChatMessage = callback
}

// OnClearMessage attach callback when a single message is deleted
func (c *Client) OnClearMessage(callback func(message ClearMessage)) {
	c.onClearMessage = callback
}

// OnRoomStateMessage attach callback to new messages such as submode enabled
func (c *Client) OnRoomStateMessage(callback func(message RoomStateMessage)) {
	c.onRoomStateMessage = callback
}

// OnUserNoticeMessage attach callback to new usernotice message such as sub, resub, and raids
func (c *Client) OnUserNoticeMessage(callback func(message UserNoticeMessage)) {
	c.onUserNoticeMessage = callback
}

// OnUserStateMessage attach callback to new userstate
func (c *Client) OnUserStateMessage(callback func(message UserStateMessage)) {
	c.onUserStateMessage = callback
}

// OnNoticeMessage attach callback to new notice message such as hosts
func (c *Client) OnNoticeMessage(callback func(message NoticeMessage)) {
	c.onNoticeMessage = callback
}

// OnUserJoinMessage attaches callback to user joins
func (c *Client) OnUserJoinMessage(callback func(message UserJoinMessage)) {
	c.onUserJoinMessage = callback
}

// OnUserPartMessage attaches callback to user parts
func (c *Client) OnUserPartMessage(callback func(message UserPartMessage)) {
	c.onUserPartMessage = callback
}

// OnReconnectMessage attaches callback that is triggered whenever the twitch servers tell us to reconnect
func (c *Client) OnReconnectMessage(callback func(message ReconnectMessage)) {
	c.onReconnectMessage = callback
}

// OnNamesMessage attaches callback to /names response
func (c *Client) OnNamesMessage(callback func(message NamesMessage)) {
	c.onNamesMessage = callback
}

// OnPingMessage attaches callback to PING message
func (c *Client) OnPingMessage(callback func(message PingMessage)) {
	c.onPingMessage = callback
}

// OnPongMessage attaches callback to PONG message
func (c *Client) OnPongMessage(callback func(message PongMessage)) {
	c.onPongMessage = callback
}

// OnUnsetMessage attaches callback to message types we currently don't support
func (c *Client) OnUnsetMessage(callback func(message RawMessage)) {
	c.onUnsetMessage = callback
}

// OnPingSent attaches callback that's called whenever the client sends out a ping message
func (c *Client) OnPingSent(callback func()) {
	c.onPingSent = callback
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
	c.send(fmt.Sprintf("PRIVMSG #%s :/w %s %s", c.ircUser, username, text))
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
	c.connActive.set(false)
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
	conn.Write([]byte("CAP REQ :twitch.tv/tags twitch.tv/commands twitch.tv/membership\r\n"))
	conn.Write([]byte("PASS " + c.ircToken + "\r\n"))
	conn.Write([]byte("NICK " + c.ircUser + "\r\n"))
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

	message := ParseMessage(line)

	switch msg := message.(type) {
	case *WhisperMessage:
		if c.onWhisperMessage != nil {
			c.onWhisperMessage(*msg)
		}
		return nil

	case *PrivateMessage:
		if c.onPrivateMessage != nil {
			c.onPrivateMessage(*msg)
		}
		return nil

	case *ClearChatMessage:
		if c.onClearChatMessage != nil {
			c.onClearChatMessage(*msg)
		}
		return nil

	case *ClearMessage:
		if c.onClearMessage != nil {
			c.onClearMessage(*msg)
		}
		return nil

	case *RoomStateMessage:
		if c.onRoomStateMessage != nil {
			c.onRoomStateMessage(*msg)
		}
		return nil

	case *UserNoticeMessage:
		if c.onUserNoticeMessage != nil {
			c.onUserNoticeMessage(*msg)
		}
		return nil

	case *UserStateMessage:
		if c.onUserStateMessage != nil {
			c.onUserStateMessage(*msg)
		}
		return nil

	case *NoticeMessage:
		if c.onNoticeMessage != nil {
			c.onNoticeMessage(*msg)
		}
		return c.handleNoticeMessage(*msg)

	case *UserJoinMessage:
		if c.handleUserJoinMessage(*msg) {
			if c.onUserJoinMessage != nil {
				c.onUserJoinMessage(*msg)
			}
		}
		return nil

	case *UserPartMessage:
		if c.handleUserPartMessage(*msg) {
			if c.onUserPartMessage != nil {
				c.onUserPartMessage(*msg)
			}
		}
		return nil

	case *ReconnectMessage:
		// https://dev.twitch.tv/docs/irc/commands/#reconnect-twitch-commands
		if c.onReconnectMessage != nil {
			c.onReconnectMessage(*msg)
		}
		return errReconnect

	case *NamesMessage:
		if c.onNamesMessage != nil {
			c.onNamesMessage(*msg)
		}
		c.handleNamesMessage(*msg)
		return nil

	case *PingMessage:
		if c.onPingMessage != nil {
			c.onPingMessage(*msg)
		}
		c.handlePingMessage(*msg)
		return nil

	case *PongMessage:
		if c.onPongMessage != nil {
			c.onPongMessage(*msg)
		}
		c.handlePongMessage(*msg)
		return nil

	case *RawMessage:
		if c.onUnsetMessage != nil {
			c.onUnsetMessage(*msg)
		}
	}

	return nil
}

func (c *Client) handleNoticeMessage(msg NoticeMessage) error {
	if msg.Channel == "*" {
		if msg.Message == "Login authentication failed" || msg.Message == "Improperly formatted auth" || msg.Message == "Invalid NICK" {
			return ErrLoginAuthenticationFailed
		}
	}

	return nil
}

func (c *Client) handleUserJoinMessage(msg UserJoinMessage) bool {
	// Ignore own joins
	if msg.User == c.ircUser {
		return false
	}

	c.channelUserlistMutex.Lock()
	defer c.channelUserlistMutex.Unlock()

	if c.channelUserlist[msg.Channel] == nil {
		c.channelUserlist[msg.Channel] = map[string]bool{}
	}

	if _, ok := c.channelUserlist[msg.Channel][msg.User]; !ok {
		c.channelUserlist[msg.Channel][msg.User] = true
	}

	return true
}

func (c *Client) handleUserPartMessage(msg UserPartMessage) bool {
	// Ignore own parts
	if msg.User == c.ircUser {
		return false
	}

	c.channelUserlistMutex.Lock()
	defer c.channelUserlistMutex.Unlock()

	delete(c.channelUserlist[msg.Channel], msg.User)

	return true
}

func (c *Client) handleNamesMessage(msg NamesMessage) {
	c.channelUserlistMutex.Lock()
	defer c.channelUserlistMutex.Unlock()

	for _, user := range msg.Users {
		c.channelUserlist[msg.Channel][user] = true
	}
}

func (c *Client) handlePingMessage(msg PingMessage) {
	if msg.Message == "" {
		c.send("PONG")
	} else {
		c.send("PONG :" + msg.Message)
	}
}

func (c *Client) handlePongMessage(msg PongMessage) {
	if msg.Message == pingSignature {
		// Received a pong that was sent by us
		select {
		case c.pongReceived <- true:
		default:
		}
	}
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
