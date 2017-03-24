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
	TWITCHIRC = "irc.chat.twitch.tv:6667"
)

type Client struct {
	ircAddress   string
	ircUser      string
	ircToken     string
	connection   *net.Conn
	connActive bool
	onNewMessage func(message Message)
}

func NewClient(username, oauth string) *Client {
	return &Client{
		ircUser:    username,
		ircToken:   oauth,
		ircAddress: TWITCHIRC,
	}
}

func (c *Client) SetIrcAddress(address string) {
	c.ircAddress = address
}

func (c *Client) Say(channel, text, responseType string) {
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
		c.onNewMessage(*ParseMessage(line))
	}
}

func (c *Client) OnNewMessage(callback func(message Message)) {
	c.onNewMessage = callback
}

func (c *Client) Join(channel string) {
	c.send(fmt.Sprintf("JOIN #%s", channel))
}
