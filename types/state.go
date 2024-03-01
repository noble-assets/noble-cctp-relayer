package types

import (
	"sync"
)

// StateMap wraps sync.Map with type safety
// maps source tx hash -> TxState
type StateMap struct {
	Mu       sync.Mutex
	internal sync.Map
}

func NewStateMap() *StateMap {
	return &StateMap{
		Mu:       sync.Mutex{},
		internal: sync.Map{},
	}
}

// load loads the message states tied to a specific transaction hash
func (sm *StateMap) Load(key string) (value *TxState, ok bool) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	internalResult, ok := sm.internal.Load(key)
	if !ok {
		return nil, ok
	}
	return internalResult.(*TxState), ok
}

func (sm *StateMap) Delete(key string) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sm.internal.Delete(key)
}

// store stores the message states tied to a specific transaction hash
func (sm *StateMap) Store(key string, value *TxState) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sm.internal.Store(key, value)
}
