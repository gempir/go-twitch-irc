package twitch

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
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

// User data you receive from tmi
type User struct {
	UserID      string
	Username    string
	DisplayName string
	UserType    string
	Color       string
	Badges      map[string]int
}

// Message data you receive from tmi
type Message struct {
	Type      MessageType
	Time      time.Time
	Action    bool
	Emotes    []*Emote
	Tags      map[string]string
	Text      string
	Raw       string
	ChannelID string
}

// Client client to control your connection and attach callbacks
type Client struct {
	IrcAddress             string
	ircUser                string
	ircToken               string
	TLS                    bool
	connection             net.Conn
	connActive             tAtomBool
	channels               map[string]bool
	channelUserlist        map[string]map[string]bool
	channelsMtx            *sync.RWMutex
	onConnect              func()
	onNewWhisper           func(user User, message Message)
	onNewMessage           func(channel string, user User, message Message)
	onNewRoomstateMessage  func(channel string, user User, message Message)
	onNewClearchatMessage  func(channel string, user User, message Message)
	onNewUsernoticeMessage func(channel string, user User, message Message)
	onNewNoticeMessage     func(channel string, user User, message Message)
	onNewUserstateMessage  func(channel string, user User, message Message)
	onUserJoin             func(channel, user string)
	onUserPart             func(channel, user string)
	onNewUnsetMessage      func(rawMessage string)

	// read is the incoming messages channel, normally buffered with ReadBufferSize
	read chan (string)

	// write is the outgoing messages channel, normally buffered with WriteBufferSize
	write chan (string)

	// clientReconnect is closed whenever the client needs to reconnect for connection issue reasons
	clientReconnect chanCloser

	// userDisconnect is closed when the user calls Disconnect
	userDisconnect chanCloser

	// pingerRunning indicates whether the pinger go-routine is running or not
	pingerRunning tAtomBool

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
		pongReceived:    make(chan bool),
		messageReceived: make(chan bool),

		read:  make(chan string, ReadBufferSize),
		write: make(chan string, WriteBufferSize),

		SendPings:        true,
		IdlePingInterval: time.Second * 15,
		PongTimeout:      time.Second * 5,
	}
}

// OnNewWhisper attach callback to new whisper
func (c *Client) OnNewWhisper(callback func(user User, message Message)) {
	c.onNewWhisper = callback
}

// OnNewMessage attach callback to new standard chat messages
func (c *Client) OnNewMessage(callback func(channel string, user User, message Message)) {
	c.onNewMessage = callback
}

// OnConnect attach callback to when a connection has been established
func (c *Client) OnConnect(callback func()) {
	c.onConnect = callback
}

// OnNewRoomstateMessage attach callback to new messages such as submode enabled
func (c *Client) OnNewRoomstateMessage(callback func(channel string, user User, message Message)) {
	c.onNewRoomstateMessage = callback
}

// OnNewClearchatMessage attach callback to new messages such as timeouts
func (c *Client) OnNewClearchatMessage(callback func(channel string, user User, message Message)) {
	c.onNewClearchatMessage = callback
}

// OnNewUsernoticeMessage attach callback to new usernotice message such as sub, resub, and raids
func (c *Client) OnNewUsernoticeMessage(callback func(channel string, user User, message Message)) {
	c.onNewUsernoticeMessage = callback
}

// OnNewNoticeMessage attach callback to new notice message such as hosts
func (c *Client) OnNewNoticeMessage(callback func(channel string, user User, message Message)) {
	c.onNewNoticeMessage = callback
}

// OnNewUserstateMessage attach callback to new userstate
func (c *Client) OnNewUserstateMessage(callback func(channel string, user User, message Message)) {
	c.onNewUserstateMessage = callback
}

// OnUserJoin attaches callback to user joins
func (c *Client) OnUserJoin(callback func(channel, user string)) {
	c.onUserJoin = callback
}

// OnUserPart attaches callback to user parts
func (c *Client) OnUserPart(callback func(channel, user string)) {
	c.onUserPart = callback
}

// OnNewUnsetMessage attaches callback to messages that didn't parse properly. Should only be used if you're debugging the message parsing
func (c *Client) OnNewUnsetMessage(callback func(rawMessage string)) {
	c.onNewUnsetMessage = callback
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
	c.channelUserlist[channel] = map[string]bool{}
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

func (c *Client) makeConnection(dialer *net.Dialer, conf *tls.Config) error {
	var err error
	if c.TLS {
		c.connection, err = tls.DialWithDialer(dialer, "tcp", c.IrcAddress, conf)
	} else {
		c.connection, err = dialer.Dial("tcp", c.IrcAddress)
	}
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	c.clientReconnect.Reset()
	c.userDisconnect.Reset()

	defer func() {
		c.connection.Close()
		c.clientReconnect.Close()
		c.userDisconnect.Close()

		wg.Wait()
	}()

	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		// reader xd
		err := c.readConnection(c.connection)
		if err != nil {
			c.clientReconnect.Close()
		}
	}()

	if c.SendPings {
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
			}()
			// pinger xd
			c.startPinger()
		}()
	}

	c.setupConnection()

	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		// writer
		for {
			select {
			case <-c.clientReconnect.channel:
				return

			case <-c.userDisconnect.channel:
				return

			case msg := <-c.write:
				_, err := c.connection.Write([]byte(msg + "\r\n"))
				if err != nil {
					// Attempt to re-send failed messages
					c.write <- msg
					c.connection.Close()
					c.clientReconnect.Close()
					return
				}
			}

		}
	}()

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

// Userlist returns the userlist for a given channel
func (c *Client) Userlist(channel string) ([]string, error) {
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

func (c *Client) readConnection(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return err
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

func (c *Client) startPinger() {
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
			c.send(pingMessage)

			select {
			case <-c.pongReceived:
				// Received pong message within the time limit, we're good
				continue

			case <-time.After(c.PongTimeout):
				// No pong message was received within the pong timeout, disconnect
				c.clientReconnect.Close()
				c.connection.Close()
			}
		}
	}
}

func (c *Client) setupConnection() {
	if c.SetupCmd != "" {
		c.connection.Write([]byte(c.SetupCmd + "\r\n"))
	}
	c.connection.Write([]byte("PASS " + c.ircToken + "\r\n"))
	c.connection.Write([]byte("NICK " + c.ircUser + "\r\n"))
	c.connection.Write([]byte("CAP REQ :twitch.tv/tags\r\n"))
	c.connection.Write([]byte("CAP REQ :twitch.tv/commands\r\n"))
	c.connection.Write([]byte("CAP REQ :twitch.tv/membership\r\n"))
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
		go func() {
			select {
			case c.pongReceived <- true:
			default:
			}
		}()

		return nil
	}

	if strings.HasPrefix(line, "@") {
		channel, user, clientMessage := ParseMessage(line)

		switch clientMessage.Type {
		case PRIVMSG:
			if c.onNewMessage != nil {
				c.onNewMessage(channel, *user, *clientMessage)
			}
		case WHISPER:
			if c.onNewWhisper != nil {
				c.onNewWhisper(*user, *clientMessage)
			}
		case ROOMSTATE:
			if c.onNewRoomstateMessage != nil {
				c.onNewRoomstateMessage(channel, *user, *clientMessage)
			}
		case CLEARCHAT:
			if c.onNewClearchatMessage != nil {
				c.onNewClearchatMessage(channel, *user, *clientMessage)
			}
		case USERNOTICE:
			if c.onNewUsernoticeMessage != nil {
				c.onNewUsernoticeMessage(channel, *user, *clientMessage)
			}
		case NOTICE:
			if c.onNewNoticeMessage != nil {
				c.onNewNoticeMessage(channel, *user, *clientMessage)
			}
		case USERSTATE:
			if c.onNewUserstateMessage != nil {
				c.onNewUserstateMessage(channel, *user, *clientMessage)
			}
		case UNSET:
			if c.onNewUnsetMessage != nil {
				c.onNewUnsetMessage(clientMessage.Raw)
			}
		}

		return nil
	}

	if strings.HasPrefix(line, ":") {
		if strings.Contains(line, "tmi.twitch.tv JOIN") {
			channel, username := parseJoinPart(line)

			if c.channelUserlist[channel] == nil {
				c.channelUserlist[channel] = map[string]bool{}
			}

			_, ok := c.channelUserlist[channel][username]
			if !ok && username != c.ircUser {
				c.channelUserlist[channel][username] = true
			}

			if c.onUserJoin != nil {
				c.onUserJoin(channel, username)
			}
		}
		if strings.Contains(line, "tmi.twitch.tv PART") {
			channel, username := parseJoinPart(line)

			delete(c.channelUserlist[channel], username)

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

			for _, user := range users {
				c.channelUserlist[channel][user] = true
			}
		}
		if strings.Contains(line, "tmi.twitch.tv NOTICE * :Login authentication failed") ||
			strings.Contains(line, "tmi.twitch.tv NOTICE * :Improperly formatted auth") ||
			line == ":tmi.twitch.tv NOTICE * :Invalid NICK" {
			return ErrLoginAuthenticationFailed
		}
	}

	return nil
}

// ParseMessage parse a raw ircv3 twitch
func ParseMessage(line string) (string, *User, *Message) {
	message := parseMessage(line)

	channel := message.Channel

	user := &User{
		UserID:      message.UserID,
		Username:    message.Username,
		DisplayName: message.DisplayName,
		UserType:    message.UserType,
		Color:       message.Color,
		Badges:      message.Badges,
	}

	clientMessage := &Message{
		Type:      message.Type,
		Time:      message.Time,
		Action:    message.Action,
		Emotes:    message.Emotes,
		Tags:      message.Tags,
		Text:      message.Text,
		Raw:       message.Raw,
		ChannelID: message.ChannelID,
	}

	return channel, user, clientMessage
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
	c.o.Do(func() { close(c.channel) })
}
