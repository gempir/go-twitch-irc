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
		if expected == nil {
			return
		}

		t.Errorf("actual slice was nil")
	}

	if len(actual) != len(expected) {
		t.Errorf("actual slice(%#v)(%d) was not the same length as expected slice(%#v)(%d)", actual, len(actual), expected, len(expected))
	}

	for i, v := range actual {
		if v != expected[i] {
			t.Errorf("actual slice value \"%s\" was not equal to expected value \"%s\" at index \"%d\"", v, expected[i], i)
		}
	}
}

func assertStringMapsEqual(t *testing.T, expected, actual map[string]string) {
	if actual == nil {
		if expected == nil {
			return
		}

		t.Errorf("actual map was nil")
	}

	if len(expected) != len(actual) {
		t.Errorf("actual map was not the same length as the expected map")
	}

	for key, want := range expected {
		got, ok := actual[key]
		if !ok {
			t.Errorf("actual map doesn't contain key \"%s\"", key)
			continue
		}

		if want != got {
			t.Errorf("actual map value \"%s\" was not equal to expected value \"%s\" in key \"%s\"", got, want, key)
			continue
		}
	}
}

func assertStringIntMapsEqual(t *testing.T, expected, actual map[string]int) {
	if actual == nil {
		t.Errorf("actual map was nil")
	}

	if len(expected) != len(actual) {
		t.Errorf("actual map was not the same length as the expected map")
	}

	for k, v := range expected {
		got, ok := actual[k]
		if !ok {
			t.Errorf("actual map doesn't contain key \"%s\"", k)
			continue
		}

		if v != got {
			t.Errorf("actual map value \"%d\" was not equal to expected value \"%d\" in key \"%s\"", got, v, k)
			continue
		}
	}
}

func assertErrorsEqual(t *testing.T, expected, actual error) {
	if expected != actual {
		t.Errorf("failed asserting that error \"%s\" is expected \"%s\"", actual, expected)
	}
}

func assertMessageTypesEqual(t *testing.T, expected, actual MessageType) {
	if expected != actual {
		t.Errorf("failed asserting that MessageType \"%d\" is expected \"%d\"", actual, expected)
	}
}

// formats a ping-signature (i.e. go-twitch-irc) into a full-fledged pong response (i.e. ":tmi.twitch.tv PONG tmi.twitch.tv :go-twitch-irc")
func formatPong(signature string) string {
	return ":tmi.twitch.tv PONG tmi.twitch.tv :" + signature
}
