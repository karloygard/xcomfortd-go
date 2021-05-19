package xc

import (
	"sync"
	"sync/atomic"
)

type Queue struct {
	m       sync.Mutex
	waiters int32
}

// Lock() returns true if no one got in line after us
func (m *Queue) Lock() bool {
	w := atomic.AddInt32(&m.waiters, 1)
	m.m.Lock()
	return atomic.CompareAndSwapInt32(&m.waiters, w, w)
}

func (m *Queue) Unlock() {
	m.m.Unlock()
}
