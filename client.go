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
	IRCTWITCH = "irc.chat.twitch.tv:6667"
)

type User struct {
	Username    string
	DisplayName string
	UserType    string
	Color       string
	Badges      map[string]int
}

type Message struct {
	Type   msgType
	Time   time.Time
	Action bool
	Emotes []*emote
	Tags   map[string]string
	Text   string
}

type Client struct {
	ircAddress            string
	ircUser               string
	ircToken              string
	connection            *net.Conn
	connActive            bool
	onNewMessage          func(channel string, user User, message Message)
	onNewRoomstateMessage func(channel string, user User, message Message)
	onNewClearchatMessage func(channel string, user User, message Message)
}

func NewClient(username, oauth string) *Client {
	return &Client{
		ircUser:    username,
		ircToken:   oauth,
		ircAddress: IRCTWITCH,
	}
}

func (c *Client) SetIrcAddress(address string) {
	c.ircAddress = address
}

func (c *Client) Say(channel, text string) {
	c.send(fmt.Sprintf("PRIVMSG #%s :%s", channel, text))
}

func (c *Client) Connect() error {
	for {
		conn, err := net.Dial("tcp", c.ircAddress)
		c.connection = &conn
		if err != nil {
			fmt.Println(conn)
			return err
		}

		go c.setupConnection()

		err = c.readConnection(conn)
		if err != nil {
			fmt.Println("connection read error, reconnecting...")
			continue
		}
	}
	return nil
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
		if len(messages) == 0 {
			continue
		}
		for _, msg := range messages {
			if !c.connActive && strings.HasPrefix(msg, ":tmi.twitch.tv 001") {
				c.connActive = true
			}
			c.handleLine(msg)
		}
	}
}

func (c *Client) setupConnection() {
	fmt.Fprint(*c.connection, fmt.Sprintf("PASS %s\r\n", c.ircToken))
	fmt.Fprint(*c.connection, fmt.Sprintf("NICK %s\r\n", c.ircUser))
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
		c.send(fmt.Sprintf(strings.Replace(line, "PING", "PONG", 1)))
	}
	if strings.HasPrefix(line, "@") {
		message := parseMessage(line)

		Channel := message.Channel

		User := User{
			Username:    message.Username,
			DisplayName: message.DisplayName,
			UserType:    message.UserType,
			Color:       message.Color,
			Badges:      message.Badges,
		}

		clientMessage := Message{
			Type:   message.Type,
			Time:   message.Time,
			Action: message.Action,
			Emotes: message.Emotes,
			Tags:   message.Tags,
			Text:   message.Text,
		}

		switch message.Type {
		case PRIVMSG:
			c.onNewMessage(Channel, User, clientMessage)
			break
		case ROOMSTATE:
			c.onNewRoomstateMessage(Channel, User, clientMessage)
			break
		case CLEARCHAT:
			c.onNewClearchatMessage(Channel, User, clientMessage)
			break
		}
	}
}

func (c *Client) OnNewMessage(callback func(channel string, user User, message Message)) {
	c.onNewMessage = callback
}

func (c *Client) OnNewRoomstateMessage(callback func(channel string, user User, message Message)) {
	c.onNewRoomstateMessage = callback
}

func (c *Client) OnNewClearchatMessage(callback func(channel string, user User, message Message)) {
	c.onNewClearchatMessage = callback
}

func (c *Client) Join(channel string) {
	go c.send(fmt.Sprintf("JOIN #%s", channel))
}
