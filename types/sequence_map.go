package types

import (
	"sync"
)

// SequenceMap holds a minter account's txn count to avoid account sequence mismatch errors
type SequenceMap struct {
	mu sync.Mutex
	// map destination domain -> minter account sequence
	sequenceMap map[Domain]int64
}

func NewSequenceMap() *SequenceMap {
	return &SequenceMap{
		sequenceMap: map[Domain]int64{},
	}
}

func (m *SequenceMap) Put(destDomain Domain, val int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sequenceMap[destDomain] = val
}

func (m *SequenceMap) Next(destDomain Domain) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := m.sequenceMap[destDomain]
	m.sequenceMap[destDomain]++
	return result
}
