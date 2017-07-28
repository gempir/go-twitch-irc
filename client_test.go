package twitch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCanCreateClient(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:1123123")

	assert.IsType(t, Client{}, *client)
}

func TestCanConnect(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:123123132")

	go client.Connect()
	time.Sleep(time.Second)
	assert.True(t, true)
}

func TestCanJoinChannel(t *testing.T) {
	client := NewClient("justinfan123123", "oauth:123123132")

	client.Join("gempir")
	time.Sleep(time.Second)
	assert.True(t, true)
}
