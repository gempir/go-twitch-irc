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

// tAtomInt atomic int for writing/reading across threads
type tAtomInt32 struct{ flag int32 }

func (a *tAtomInt32) set(value int32) {
	atomic.StoreInt32(&(a.flag), value)
}

func (a *tAtomInt32) increment() {
	atomic.StoreInt32(&(a.flag), a.get()+1)
}

func (a *tAtomInt32) get() int32 {
	return atomic.LoadInt32(&(a.flag))
}
