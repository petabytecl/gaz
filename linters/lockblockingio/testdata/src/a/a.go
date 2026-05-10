package a

import "sync"

// deferUnlockWithChannelSend should be flagged: defer Unlock + channel send.
func deferUnlockWithChannelSend(mu *sync.Mutex, ch chan int) {
	mu.Lock()
	defer mu.Unlock() // want `defer Unlock followed by blocking operation channel send`
	ch <- 1
}

// deferRUnlockWithWaitGroupWait should be flagged: defer RUnlock + WaitGroup.Wait.
func deferRUnlockWithWaitGroupWait(mu *sync.RWMutex, wg *sync.WaitGroup) {
	mu.RLock()
	defer mu.RUnlock() // want `defer RUnlock followed by blocking operation WaitGroup.Wait`
	wg.Wait()
}

// deferUnlockWithChannelRecv should be flagged: defer Unlock + channel receive.
func deferUnlockWithChannelRecv(mu *sync.Mutex, ch chan int) {
	mu.Lock()
	defer mu.Unlock() // want `defer Unlock followed by blocking operation channel receive`
	<-ch
}

// deferUnlockWithBlockingSelect should be flagged: defer Unlock + blocking select.
func deferUnlockWithBlockingSelect(mu *sync.Mutex, ch1 chan int, ch2 chan int) {
	mu.Lock()
	defer mu.Unlock() // want `defer Unlock followed by blocking operation blocking select`
	select {
	case <-ch1:
	case <-ch2:
	}
}

// --- Below: cases that should NOT be flagged ---

// deferUnlockNoBlocking has defer Unlock but no blocking ops.
func deferUnlockNoBlocking(mu *sync.Mutex) int {
	mu.Lock()
	defer mu.Unlock()
	return 42
}

// blockingNoDefer has blocking ops but no defer unlock.
func blockingNoDefer(ch chan int) {
	ch <- 1
}

// nonBlockingSelect has a select with a default case (non-blocking).
func nonBlockingSelect(mu *sync.Mutex, ch chan int) {
	mu.Lock()
	defer mu.Unlock()
	select {
	case <-ch:
	default:
	}
}
