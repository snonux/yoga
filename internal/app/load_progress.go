package app

import "sync"

type loadProgress struct {
	mu        sync.Mutex
	total     int
	processed int
	done      bool
}

func (p *loadProgress) Reset() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.total = 0
	p.processed = 0
	p.done = false
	p.mu.Unlock()
}

func (p *loadProgress) SetTotal(total int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.total = total
	p.mu.Unlock()
}

func (p *loadProgress) Increment() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.processed++
	p.mu.Unlock()
}

func (p *loadProgress) MarkDone() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.done = true
	p.mu.Unlock()
}

func (p *loadProgress) Snapshot() (processed, total int, done bool) {
	if p == nil {
		return 0, 0, true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.processed, p.total, p.done
}
