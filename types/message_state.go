package types

import "time"

type Status uint32

const (
	Burned   Status = 0
	Pending  Status = 1
	Attested Status = 2
	Complete Status = 3
)

type MessageType string

const (
	Mint    MessageType = "mint"
	Forward MessageType = "forward"
)

type MessageState struct {
	IrisLookupId string
	Type         MessageType
	Status       Status
	SourceDomain uint32
	DestDomain   uint32
	SourceTxHash string
	DestTxHash   string
	MsgSentBytes []byte
	Created      time.Time
	Updated      time.Time
}
