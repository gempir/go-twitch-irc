package twitch

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"time"
)

const (
	// ircTwitch constant for irc chat address
	ircTwitch = "irc.chat.twitch.tv:6667"
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
	Emotes []*emote
	Tags   map[string]string
	Text   string
}

// Client client to control your connection and attach callbacks
type Client struct {
	ircAddress            string
	ircUser               string
	ircToken              string
	connection            *net.Conn
	connActive            bool
	onNewMessage          MessageCallback
	onNewRoomstateMessage MessageCallback
	onNewClearchatMessage MessageCallback
}

// MessageCallback is called when a new message is received
type MessageCallback func(channel string, user User, message Message)

// NewClient to create a new client
func NewClient(username, oauth string) *Client {
	return &Client{
		ircUser:    username,
		ircToken:   oauth,
		ircAddress: ircTwitch,
	}
}

// SetIrcAddress overwrite the standard tmi irc address
func (c *Client) SetIrcAddress(address string) {
	c.ircAddress = address
}

// OnNewMessage attach callback to new standard chat messages
func (c *Client) OnNewMessage(callback MessageCallback) {
	c.onNewMessage = callback
}

// OnNewRoomstateMessage attach callback to new messages such as submode enabled
func (c *Client) OnNewRoomstateMessage(callback MessageCallback) {
	c.onNewRoomstateMessage = callback
}

// OnNewClearchatMessage attach callback to new messages such as timeouts
func (c *Client) OnNewClearchatMessage(callback MessageCallback) {
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

// Connect open a connection to the tmi irc address set
func (c *Client) Connect() error {
	for {
		conn, err := net.Dial("tcp", c.ircAddress)
		c.connection = &conn
		if err != nil {
			return err
		}

		go c.setupConnection()

		err = c.readConnection(conn)
		if err != nil {
			fmt.Println("connection read error, reconnecting...")
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
			if !c.connActive && strings.HasPrefix(msg, ":tmi.twitch.tv 001") {
				c.connActive = true
			}
			c.handleLine(msg)
		}
	}
}

func (c *Client) setupConnection() {
	fmt.Fprintf(*c.connection, "PASS %s\r\n", c.ircToken)
	fmt.Fprintf(*c.connection, "NICK %s\r\n", c.ircUser)
	fmt.Fprint(*c.connection, "CAP REQ :twitch.tv/tags\r\n")
	fmt.Fprint(*c.connection, "CAP REQ :twitch.tv/commands\r\n")
}

func (c *Client) send(line string) {
	if !c.connActive {
		time.Sleep(time.Second * 1)
		c.send(line)
		return
	}
	fmt.Fprint(*c.connection, line+"\r\n")
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
			if c.OnNewMessage != nil {
				c.onNewMessage(Channel, *User, *clientMessage)
			}
		case ROOMSTATE:
			if c.onNewRoomstateMessage != nil {
				c.onNewRoomstateMessage(Channel, *User, *clientMessage)
			}
		case CLEARCHAT:
			if c.onNewRoomstateMessage != nil {
				c.onNewClearchatMessage(Channel, *User, *clientMessage)
			}
		}
	}
}
