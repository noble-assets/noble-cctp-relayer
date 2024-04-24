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
		IrisLookupID: "123",
		Status:       Filtered,
		MsgSentBytes: []byte("i like turtles"),
	}

	stateMap.Store(txHash, &TxState{
		TxHash: txHash,
		Msgs: []*MessageState{
			&msg,
		},
	})

	loadedMsg, _ := stateMap.Load(txHash)
	require.True(t, msg.Equal(loadedMsg.Msgs[0]))

	loadedMsg.Msgs[0].Status = Complete

	// Because it is a pointer, no need to re-store to state
	// message status should be updated with out re-storing.
	loadedMsg2, _ := stateMap.Load(txHash)
	require.Equal(t, Complete, loadedMsg2.Msgs[0].Status)

	// even though loadedMsg is a pointer, if we add to the array, we need to re-store in cache.
	msg2 := MessageState{
		SourceTxHash: txHash,
		IrisLookupID: "123",
		Status:       Filtered,
		MsgSentBytes: []byte("mock bytes 2"),
	}

	loadedMsg.Msgs = append(loadedMsg.Msgs, &msg2)
	stateMap.Store(txHash, loadedMsg)

	loadedMsg3, _ := stateMap.Load(txHash)
	require.Len(t, loadedMsg3.Msgs, 2)
}
