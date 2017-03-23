package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCanCreateClient(t *testing.T) {
	client := NewClient()

	assert.IsType(t, Client{}, *client)
}
