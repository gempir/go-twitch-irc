package twitch

import (
	"fmt"
	"time"
)

func main() {
	client := NewClient("justinfan123", "oauth:123123")
	go func() {
		err := client.Connect()
		fmt.Println(err.Error())
	}()

	client.Join("pajlada")

	client.OnNewMessage(func(message Message) {
		fmt.Println(message.Text)
	})

	time.Sleep(time.Minute)
}
