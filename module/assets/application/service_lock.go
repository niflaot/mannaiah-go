package application

import "sync"

// locker defines keyed lock behavior for in-process write serialization.
type locker interface {
	// Lock acquires a lock for the provided key and returns its unlock function.
	Lock(key string) func()
}

// keyedLocker serializes operations by arbitrary lock keys.
type keyedLocker struct {
	// mu protects items.
	mu sync.Mutex
	// items stores lock refs by key.
	items map[string]*lockRef
}

// lockRef stores keyed mutex values with reference tracking.
type lockRef struct {
	// mu serializes critical sections for the key.
	mu sync.Mutex
	// refs tracks active holders and waiters.
	refs int
}

// newKeyedLocker creates keyed in-memory lock managers.
func newKeyedLocker() *keyedLocker {
	return &keyedLocker{
		items: map[string]*lockRef{},
	}
}

// Lock acquires key-scoped mutexes and returns paired unlock functions.
func (k *keyedLocker) Lock(key string) func() {
	trimmedKey := key
	k.mu.Lock()
	ref, exists := k.items[trimmedKey]
	if !exists {
		ref = &lockRef{}
		k.items[trimmedKey] = ref
	}
	ref.refs++
	k.mu.Unlock()

	ref.mu.Lock()

	return func() {
		ref.mu.Unlock()
		k.mu.Lock()
		ref.refs--
		if ref.refs == 0 {
			delete(k.items, trimmedKey)
		}
		k.mu.Unlock()
	}
}
