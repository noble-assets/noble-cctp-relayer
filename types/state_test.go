package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateHandling(t *testing.T) {
	stateMap := NewStateMap()

	txHash := "123456789"
	msg := MessageState{
		SourceTxHash: txHash,
		IrisLookupId: "123",
		Status:       Filtered,
		MsgSentBytes: []byte("i like turtles"),
	}

	stateMap.Store(txHash, []*MessageState{&msg})

	loadedMsg, _ := stateMap.Load(txHash)
	require.True(t, msg.Equal(loadedMsg[0]))

	loadedMsg[0].Status = Complete

	// Becasue it is a pointer, no need to re-store to state
	// message status should be updated with out re-storing.
	loadedMsg2, _ := stateMap.Load(txHash)
	require.Equal(t, Complete, loadedMsg2[0].Status)

	// even though loadedMsg is a pointer, if we add to the array, we need to re-store in cache.
	msg2 := MessageState{
		SourceTxHash: txHash,
		IrisLookupId: "123",
		Status:       Filtered,
		MsgSentBytes: []byte("mock bytes 2"),
	}

	loadedMsg = append(loadedMsg, &msg2)
	stateMap.Store(txHash, loadedMsg)

	loadedMsg3, _ := stateMap.Load(txHash)
	require.Equal(t, 2, len(loadedMsg3))
}
