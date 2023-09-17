package types

import (
	"fmt"
	"testing"
)

var stateMap StateMap

func TestX(t *testing.T) {
	stateMap := NewStateMap()
	msg := MessageState{IrisLookupId: "123", Status: Filtered}
	stateMap.Store("123", &msg)

	lMsg, _ := stateMap.Load("123")
	fmt.Println(lMsg)

	msg.Status = Complete

	f, _ := stateMap.Load("123")

	lMsg.Status = Created

	f, _ = stateMap.Load("123")

	fmt.Println(f)

}
