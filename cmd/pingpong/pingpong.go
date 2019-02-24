package main

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	twitch "github.com/gempir/go-twitch-irc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	clientUsername            = "justinfan123123"
	clientAuthenticationToken = "oauth:123123123"
)

var pingMetric = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "pings_received",
		Help: "Pings received in the last 15 seconds",
	})

func init() {
	prometheus.MustRegister(pingMetric)
}

func main() {
	client := twitch.NewClient(clientUsername, clientAuthenticationToken)

	m := sync.Mutex{}
	pingsReceived := 0

	client.OnNewMessage(func(channel string, user twitch.User, message twitch.Message) {
		if strings.Contains(strings.ToLower(message.Text), "ping") {
			log.Println(user.Username, "PONG", message.Text)
			m.Lock()
			pingsReceived++
			m.Unlock()
		}
	})

	go func() {
		for {
			<-time.After(15 * time.Second)
			m.Lock()
			pingMetric.Set(float64(pingsReceived))
			pingsReceived = 0
			m.Unlock()
		}
	}()

	client.Join("testaccount_420")

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":9101", nil))
	}()

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
