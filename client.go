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
)

var (
	// ErrClientDisconnected returned from Connect() when a Disconnect() was called
	ErrClientDisconnected = errors.New("client called Disconnect()")
)

// User data you receive from tmi
type User struct {
	UserID      int64
	Username    string
	DisplayName string
	UserType    string
	Color       string
	Badges      map[string]int
}

// Message data you receive from tmi
type Message struct {
	Type   msgType
	Time   time.Time
	Action bool
	Emotes []*Emote
	Tags   map[string]string
	Text   string
}

// Client client to control your connection and attach callbacks
type Client struct {
	IrcAddress             string
	ircUser                string
	ircToken               string
	TLS                    bool
	connection             net.Conn
	connActive             tAtomBool
	disconnected           tAtomBool
	channels               map[string]bool
	channelsMtx            *sync.RWMutex
	onConnect              func()
	onNewWhisper           func(user User, message Message)
	onNewMessage           func(channel string, user User, message Message)
	onNewRoomstateMessage  func(channel string, user User, message Message)
	onNewClearchatMessage  func(channel string, user User, message Message)
	onNewUsernoticeMessage func(channel string, user User, message Message)
	onNewUserstateMessage  func(channel string, user User, message Message)
}

// NewClient to create a new client
func NewClient(username, oauth string) *Client {
	return &Client{
		ircUser:     username,
		ircToken:    oauth,
		TLS:         true,
		channels:    map[string]bool{},
		channelsMtx: &sync.RWMutex{},
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

// OnNewUserstateMessage attach callback to new userstate
func (c *Client) OnNewUserstateMessage(callback func(channel string, user User, message Message)) {
	c.onNewUserstateMessage = callback
}

// Say write something in a chat
func (c *Client) Say(channel, text string) {
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
	c.channelsMtx.Unlock()
}

// Depart leave a twitch channel
func (c *Client) Depart(channel string) {
	if c.connActive.get() {
		go c.send(fmt.Sprintf("PART #%s", channel))
	}

	c.channelsMtx.Lock()
	delete(c.channels, channel)
	c.channelsMtx.Unlock()
}

// Disconnect close current connection
func (c *Client) Disconnect() error {
	c.connActive.set(false)
	c.disconnected.set(true)
	if c.connection != nil {
		return c.connection.Close()
	}
	return errors.New("connection not open")
}

// Connect connect the client to the irc server
func (c *Client) Connect() error {
	if c.IrcAddress == "" && c.TLS {
		c.IrcAddress = ircTwitchTLS
	} else if c.IrcAddress == "" && !c.TLS {
		c.IrcAddress = ircTwitch
	}

	c.disconnected.set(false)

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
		if c.disconnected.get() {
			return ErrClientDisconnected
		}

		var err error
		if c.TLS {
			c.connection, err = tls.DialWithDialer(dialer, "tcp", c.IrcAddress, conf)
		} else {
			c.connection, err = dialer.Dial("tcp", c.IrcAddress)
		}
		if err != nil {
			return err
		}

		go c.setupConnection()

		err = c.readConnection(c.connection)
		if err != nil {
			time.Sleep(time.Millisecond * 200)
			continue
		}
	}
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
				if c.onConnect != nil {
					c.onConnect()
				}
			}
			c.handleLine(msg)
		}
	}
}

func (c *Client) setupConnection() {
	c.connection.Write([]byte("PASS " + c.ircToken + "\r\n"))
	c.connection.Write([]byte("NICK " + c.ircUser + "\r\n"))
	c.connection.Write([]byte("CAP REQ :twitch.tv/tags\r\n"))
	c.connection.Write([]byte("CAP REQ :twitch.tv/commands\r\n"))

	// join or rejoin channels on connection
	c.channelsMtx.RLock()
	for channel := range c.channels {
		c.send(fmt.Sprintf("JOIN #%s", channel))
	}
	c.channelsMtx.RUnlock()
}

func (c *Client) send(line string) {
	for i := 0; i < 1000; i++ {
		if !c.connActive.get() {
			time.Sleep(time.Millisecond * 2)
			continue
		}
		c.connection.Write([]byte(line + "\r\n"))
		return
	}
}

func (c *Client) handleLine(line string) {
	if strings.HasPrefix(line, "PING") {
		c.send(strings.Replace(line, "PING", "PONG", 1))
	}
	if strings.HasPrefix(line, "@") {
		message := parseMessage(line)

		Channel := message.Channel

		User := &User{
			UserID:      message.UserID,
			Username:    message.Username,
			DisplayName: message.DisplayName,
			UserType:    message.UserType,
			Color:       message.Color,
			Badges:      message.Badges,
		}

		clientMessage := &Message{
			Type:   message.Type,
			Time:   message.Time,
			Action: message.Action,
			Emotes: message.Emotes,
			Tags:   message.Tags,
			Text:   message.Text,
		}

		switch message.Type {
		case PRIVMSG:
			if c.onNewMessage != nil {
				c.onNewMessage(Channel, *User, *clientMessage)
			}
		case WHISPER:
			if c.onNewWhisper != nil {
				c.onNewWhisper(*User, *clientMessage)
			}
		case ROOMSTATE:
			if c.onNewRoomstateMessage != nil {
				c.onNewRoomstateMessage(Channel, *User, *clientMessage)
			}
		case CLEARCHAT:
			if c.onNewClearchatMessage != nil {
				c.onNewClearchatMessage(Channel, *User, *clientMessage)
			}
		case USERNOTICE:
			if c.onNewUsernoticeMessage != nil {
				c.onNewUsernoticeMessage(Channel, *User, *clientMessage)
			}
		case USERSTATE:
			if c.onNewUserstateMessage != nil {
				c.onNewUserstateMessage(Channel, *User, *clientMessage)
			}
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
