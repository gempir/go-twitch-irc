package twitch

import "sync/atomic"

// tAtomBool atomic bool for writing/reading across threads
type tAtomBool struct{ flag int32 }

func (b *tAtomBool) set(value bool) {
	var i int32
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), i)
}

func (b *tAtomBool) get() bool {
	return atomic.LoadInt32(&(b.flag)) != 0
}
