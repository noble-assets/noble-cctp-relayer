// Code generated by https://github.com/gagliardetto/anchor-go. DO NOT EDIT.

package token_messenger_minter

import (
	"errors"
	ag_binary "github.com/gagliardetto/binary"
	ag_solanago "github.com/gagliardetto/solana-go"
	ag_format "github.com/gagliardetto/solana-go/text/format"
	ag_treeout "github.com/gagliardetto/treeout"
)

// TransferOwnership is the `transferOwnership` instruction.
type TransferOwnership struct {
	Params *TransferOwnershipParams

	// [0] = [SIGNER] owner
	//
	// [1] = [WRITE] tokenMessenger
	//
	// [2] = [] eventAuthority
	//
	// [3] = [] program
	ag_solanago.AccountMetaSlice `bin:"-"`
}

// NewTransferOwnershipInstructionBuilder creates a new `TransferOwnership` instruction builder.
func NewTransferOwnershipInstructionBuilder() *TransferOwnership {
	nd := &TransferOwnership{
		AccountMetaSlice: make(ag_solanago.AccountMetaSlice, 4),
	}
	return nd
}

// SetParams sets the "params" parameter.
func (inst *TransferOwnership) SetParams(params TransferOwnershipParams) *TransferOwnership {
	inst.Params = &params
	return inst
}

// SetOwnerAccount sets the "owner" account.
func (inst *TransferOwnership) SetOwnerAccount(owner ag_solanago.PublicKey) *TransferOwnership {
	inst.AccountMetaSlice[0] = ag_solanago.Meta(owner).SIGNER()
	return inst
}

// GetOwnerAccount gets the "owner" account.
func (inst *TransferOwnership) GetOwnerAccount() *ag_solanago.AccountMeta {
	return inst.AccountMetaSlice.Get(0)
}

// SetTokenMessengerAccount sets the "tokenMessenger" account.
func (inst *TransferOwnership) SetTokenMessengerAccount(tokenMessenger ag_solanago.PublicKey) *TransferOwnership {
	inst.AccountMetaSlice[1] = ag_solanago.Meta(tokenMessenger).WRITE()
	return inst
}

// GetTokenMessengerAccount gets the "tokenMessenger" account.
func (inst *TransferOwnership) GetTokenMessengerAccount() *ag_solanago.AccountMeta {
	return inst.AccountMetaSlice.Get(1)
}

// SetEventAuthorityAccount sets the "eventAuthority" account.
func (inst *TransferOwnership) SetEventAuthorityAccount(eventAuthority ag_solanago.PublicKey) *TransferOwnership {
	inst.AccountMetaSlice[2] = ag_solanago.Meta(eventAuthority)
	return inst
}

// GetEventAuthorityAccount gets the "eventAuthority" account.
func (inst *TransferOwnership) GetEventAuthorityAccount() *ag_solanago.AccountMeta {
	return inst.AccountMetaSlice.Get(2)
}

// SetProgramAccount sets the "program" account.
func (inst *TransferOwnership) SetProgramAccount(program ag_solanago.PublicKey) *TransferOwnership {
	inst.AccountMetaSlice[3] = ag_solanago.Meta(program)
	return inst
}

// GetProgramAccount gets the "program" account.
func (inst *TransferOwnership) GetProgramAccount() *ag_solanago.AccountMeta {
	return inst.AccountMetaSlice.Get(3)
}

func (inst TransferOwnership) Build() *Instruction {
	return &Instruction{BaseVariant: ag_binary.BaseVariant{
		Impl:   inst,
		TypeID: Instruction_TransferOwnership,
	}}
}

// ValidateAndBuild validates the instruction parameters and accounts;
// if there is a validation error, it returns the error.
// Otherwise, it builds and returns the instruction.
func (inst TransferOwnership) ValidateAndBuild() (*Instruction, error) {
	if err := inst.Validate(); err != nil {
		return nil, err
	}
	return inst.Build(), nil
}

func (inst *TransferOwnership) Validate() error {
	// Check whether all (required) parameters are set:
	{
		if inst.Params == nil {
			return errors.New("Params parameter is not set")
		}
	}

	// Check whether all (required) accounts are set:
	{
		if inst.AccountMetaSlice[0] == nil {
			return errors.New("accounts.Owner is not set")
		}
		if inst.AccountMetaSlice[1] == nil {
			return errors.New("accounts.TokenMessenger is not set")
		}
		if inst.AccountMetaSlice[2] == nil {
			return errors.New("accounts.EventAuthority is not set")
		}
		if inst.AccountMetaSlice[3] == nil {
			return errors.New("accounts.Program is not set")
		}
	}
	return nil
}

func (inst *TransferOwnership) EncodeToTree(parent ag_treeout.Branches) {
	parent.Child(ag_format.Program(ProgramName, ProgramID)).
		//
		ParentFunc(func(programBranch ag_treeout.Branches) {
			programBranch.Child(ag_format.Instruction("TransferOwnership")).
				//
				ParentFunc(func(instructionBranch ag_treeout.Branches) {

					// Parameters of the instruction:
					instructionBranch.Child("Params[len=1]").ParentFunc(func(paramsBranch ag_treeout.Branches) {
						paramsBranch.Child(ag_format.Param("Params", *inst.Params))
					})

					// Accounts of the instruction:
					instructionBranch.Child("Accounts[len=4]").ParentFunc(func(accountsBranch ag_treeout.Branches) {
						accountsBranch.Child(ag_format.Meta("         owner", inst.AccountMetaSlice.Get(0)))
						accountsBranch.Child(ag_format.Meta("tokenMessenger", inst.AccountMetaSlice.Get(1)))
						accountsBranch.Child(ag_format.Meta("eventAuthority", inst.AccountMetaSlice.Get(2)))
						accountsBranch.Child(ag_format.Meta("       program", inst.AccountMetaSlice.Get(3)))
					})
				})
		})
}

func (obj TransferOwnership) MarshalWithEncoder(encoder *ag_binary.Encoder) (err error) {
	// Serialize `Params` param:
	err = encoder.Encode(obj.Params)
	if err != nil {
		return err
	}
	return nil
}
func (obj *TransferOwnership) UnmarshalWithDecoder(decoder *ag_binary.Decoder) (err error) {
	// Deserialize `Params`:
	err = decoder.Decode(&obj.Params)
	if err != nil {
		return err
	}
	return nil
}

// NewTransferOwnershipInstruction declares a new TransferOwnership instruction with the provided parameters and accounts.
func NewTransferOwnershipInstruction(
	// Parameters:
	params TransferOwnershipParams,
	// Accounts:
	owner ag_solanago.PublicKey,
	tokenMessenger ag_solanago.PublicKey,
	eventAuthority ag_solanago.PublicKey,
	program ag_solanago.PublicKey) *TransferOwnership {
	return NewTransferOwnershipInstructionBuilder().
		SetParams(params).
		SetOwnerAccount(owner).
		SetTokenMessengerAccount(tokenMessenger).
		SetEventAuthorityAccount(eventAuthority).
		SetProgramAccount(program)
}