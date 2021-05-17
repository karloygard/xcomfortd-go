package xc

import "time"

type waiter struct {
	consumer chan []byte
	seq      int
	started  time.Time
}

type waithandler struct {
	waiters []waiter
	next    int
}

func (w *waithandler) Add(consumer chan []byte) (int, int) {
restart:
	for {
		w.next = (w.next + 1) % 16

		for i := range w.waiters {
			if w.waiters[i].seq == w.next {
				continue restart
			}
		}
		break
	}

	w.waiters = append(w.waiters, waiter{consumer, w.next, time.Now()})

	return w.next, len(w.waiters)
}

func (w waithandler) Close() {
	for i := range w.waiters {
		w.waiters[i].consumer <- nil
	}
}

func (w *waithandler) Resume(data []byte, seq int) bool {
	for i := range w.waiters {
		if w.waiters[i].seq == seq {
			consumer := w.waiters[i].consumer
			w.waiters = append(w.waiters[:i], w.waiters[i+1:]...)
			consumer <- data
			return true
		}
	}
	return false
}

func (w *waithandler) OldestExpiring(seconds int) <-chan time.Time {
	if len(w.waiters) == 0 {
		return make(chan time.Time)
	}
	var since time.Duration
	for i := range w.waiters {
		d := time.Since(w.waiters[i].started)
		if since < d {
			since = d
		}
	}
	return time.After(since + time.Duration(seconds)*time.Second)
}

func (w *waithandler) ResumeOldest(data []byte) bool {
	seq := -1
	timestamp := time.Now()
	for i := range w.waiters {
		if w.waiters[i].started.Before(timestamp) {
			timestamp = w.waiters[i].started
			seq = w.waiters[i].seq
		}
	}
	return w.Resume(data, seq)
}
