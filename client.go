package twitch

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/textproto"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// ircTwitch constant for twitch irc chat address
	ircTwitch = "irc.chat.twitch.tv:443"
)

// User data you receive from tmi
type User struct {
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
	IrcAddress            string
	ircUser               string
	ircToken              string
	connection            *tls.Conn
	connActive            tAtomBool
	onNewWhisper          func(user User, message Message)
	onNewMessage          func(channel string, user User, message Message)
	onNewRoomstateMessage func(channel string, user User, message Message)
	onNewClearchatMessage func(channel string, user User, message Message)
}

// NewClient to create a new client
func NewClient(username, oauth string) *Client {
	return &Client{
		ircUser:    username,
		ircToken:   oauth,
		IrcAddress: ircTwitch,
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

// OnNewRoomstateMessage attach callback to new messages such as submode enabled
func (c *Client) OnNewRoomstateMessage(callback func(channel string, user User, message Message)) {
	c.onNewRoomstateMessage = callback
}

// OnNewClearchatMessage attach callback to new messages such as timeouts
func (c *Client) OnNewClearchatMessage(callback func(channel string, user User, message Message)) {
	c.onNewClearchatMessage = callback
}

// Say write something in a chat
func (c *Client) Say(channel, text string) {
	c.send(fmt.Sprintf("PRIVMSG #%s :%s", channel, text))
}

// Join enter a twitch channel to read more messages
func (c *Client) Join(channel string) {
	go c.send(fmt.Sprintf("JOIN #%s", channel))
}

// Connect connect the client to the irc server
func (c *Client) Connect() error {
	var conf *tls.Config
	// This means we are connecting to "localhost". Disable certificate chain check
	if strings.HasPrefix(c.IrcAddress, ":") {
		conf = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else {
		conf = &tls.Config{}
	}
	for {
		conn, err := tls.Dial("tcp", c.IrcAddress, conf)
		c.connection = conn
		if err != nil {
			return err
		}

		go c.setupConnection()

		err = c.readConnection(conn)
		if err != nil {
			time.Sleep(time.Millisecond * 200)
			continue
		}
	}
}

func (c *Client) readConnection(conn *tls.Conn) error {
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
