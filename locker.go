package sets

// Locker interface used to determine if a locked implementation is being used.
type Locker interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
}
