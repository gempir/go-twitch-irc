package twitch

import "testing"

func TestTAtomBoolDefaultValue(t *testing.T) {
	var v tAtomBool
	assertFalse(t, v.get(), "Default value should be false")
}

func TestTAtomBoolSet(t *testing.T) {
	var v tAtomBool
	assertFalse(t, v.get(), "Initial value should be false")
	v.set(true)
	assertTrue(t, v.get(), "Should be true after being set to true")
	v.set(false)
	assertFalse(t, v.get(), "Should be false after being set to true")
}
