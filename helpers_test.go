package twitch

import (
	"testing"
)

func assertStringsEqual(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("failed asserting that \"%s\" is expected \"%s\"", actual, expected)
	}
}

func assertIntsEqual(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("failed asserting that \"%d\" is expected \"%d\"", actual, expected)
	}
}

func assertInt32sEqual(t *testing.T, expected, actual int32) {
	if expected != actual {
		t.Errorf("failed asserting that \"%d\" is expected \"%d\"", actual, expected)
	}
}

func assertInt64sEqual(t *testing.T, expected, actual int64) {
	if expected != actual {
		t.Errorf("failed asserting that \"%d\" is expected \"%d\"", actual, expected)
	}
}

func assertTrue(t *testing.T, actual bool, errorMessage string) {
	if !actual {
		t.Error(errorMessage)
	}
}

func assertFalse(t *testing.T, actual bool, errorMessage string) {
	if actual {
		t.Error(errorMessage)
	}
}

func assertStringSlicesEqual(t *testing.T, expected, actual []string) {
	if actual == nil {
		t.Errorf("actual slice was nil")
	}

	if len(actual) != len(expected) {
		t.Errorf("actual slice was not the same length as expected slice")
	}

	for i, v := range actual {
		if v != expected[i] {
			t.Errorf("actual slice value \"%s\" was not equal to expected value \"%s\" at index \"%d\"", v, expected[i], i)
		}
	}
}

func assertErrorsEqual(t *testing.T, expected, actual error) {
	if expected != actual {
		t.Errorf("failed asserting that error \"%s\" is expected \"%s\"", actual, expected)
	}
}

// formats a ping-signature (i.e. go-twitch-irc) into a full-fledged pong response (i.e. ":tmi.twitch.tv PONG tmi.twitch.tv :go-twitch-irc")
func formatPong(signature string) string {
	return ":tmi.twitch.tv PONG tmi.twitch.tv :" + signature
}
