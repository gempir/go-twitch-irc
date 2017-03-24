package twitch

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCanCreateClient(t *testing.T) {
	client := NewClient("username", "oauth:1123123")

	assert.IsType(t, Client{}, *client)
}
