package xc

type waiter struct {
	consumer chan []byte
	seq      int
}

type waithandler struct {
	waiters []waiter
	next    int
}

func (w *waithandler) Add(consumer chan []byte) int {
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

	w.waiters = append(w.waiters, waiter{consumer, w.next})

	return w.next
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
