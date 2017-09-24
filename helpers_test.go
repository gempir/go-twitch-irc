package twitch

import "testing"

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
