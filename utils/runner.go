package utils

import (
	"sync"
)

type Runner struct {
	wg sync.WaitGroup
}

func (r *Runner) Async(fn func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		fn()
	}()
}

func (r *Runner) Wait() { r.wg.Wait() }
