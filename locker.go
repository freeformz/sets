package sets

type locker interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
	Wait()
	Broadcast()
}
