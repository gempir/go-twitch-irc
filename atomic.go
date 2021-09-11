package twitch

import "sync/atomic"

// tAtomBool atomic bool for writing/reading across threads
type tAtomBool struct{ flag int32 }

func (a *tAtomBool) set(value bool) {
	var i int32
	if value {
		i = 1
	}
	atomic.StoreInt32(&(a.flag), i)
}

func (a *tAtomBool) get() bool {
	return atomic.LoadInt32(&(a.flag)) != 0
}
