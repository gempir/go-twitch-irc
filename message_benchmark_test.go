package twitch

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"testing"
)

var messages []string

func TestMain(m *testing.M) {
	messages = readLog("./test_resources/channel.txt.gz")
	os.Exit(m.Run())
}

func BenchmarkParseBigLog(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, line := range messages {
			ParseMessage(line)
		}
	}
}

func BenchmarkParseWHISPERMessage(b *testing.B) {
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv WHISPER gempir :i like memes"
	for n := 0; n < b.N; n++ {
		ParseMessage(testMessage)
	}
}

func BenchmarkParseMessageType(b *testing.B) {
	testCommand := "RECONNECT"
	for n := 0; n < b.N; n++ {
		parseMessageType(testCommand)
	}
}

func readLog(logFile string) []string {
	f, err := os.Open(logFile)
	if err != nil {
		fmt.Println("logFile not found")
		os.Exit(1)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		fmt.Println("logFile gzip not readable")
		os.Exit(1)
	}

	scanner := bufio.NewScanner(gz)
	if err != nil {
		fmt.Println("logFile not readable")
		os.Exit(1)
	}

	content := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		content = append(content, line)
	}

	return content
}
