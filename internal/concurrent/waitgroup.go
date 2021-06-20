package concurrent

import "sync"

// WhenDone returns a channel that is closed when the WaitGroup counter is zero.
func WhenDone(wg *sync.WaitGroup) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	return done
}
