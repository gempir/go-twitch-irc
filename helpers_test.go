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
