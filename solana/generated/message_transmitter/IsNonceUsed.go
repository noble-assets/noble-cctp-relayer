// Code generated by https://github.com/gagliardetto/anchor-go. DO NOT EDIT.

package message_transmitter

import (
	"errors"
	ag_binary "github.com/gagliardetto/binary"
	ag_solanago "github.com/gagliardetto/solana-go"
	ag_format "github.com/gagliardetto/solana-go/text/format"
	ag_treeout "github.com/gagliardetto/treeout"
)

// IsNonceUsed is the `isNonceUsed` instruction.
type IsNonceUsed struct {
	Params *IsNonceUsedParams

	// [0] = [] usedNonces
	// ··········· Account will be explicitly loaded to avoid error when it's not initialized
	ag_solanago.AccountMetaSlice `bin:"-"`
}

// NewIsNonceUsedInstructionBuilder creates a new `IsNonceUsed` instruction builder.
func NewIsNonceUsedInstructionBuilder() *IsNonceUsed {
	nd := &IsNonceUsed{
		AccountMetaSlice: make(ag_solanago.AccountMetaSlice, 1),
	}
	return nd
}

// SetParams sets the "params" parameter.
func (inst *IsNonceUsed) SetParams(params IsNonceUsedParams) *IsNonceUsed {
	inst.Params = &params
	return inst
}

// SetUsedNoncesAccount sets the "usedNonces" account.
// Account will be explicitly loaded to avoid error when it's not initialized
func (inst *IsNonceUsed) SetUsedNoncesAccount(usedNonces ag_solanago.PublicKey) *IsNonceUsed {
	inst.AccountMetaSlice[0] = ag_solanago.Meta(usedNonces)
	return inst
}

// GetUsedNoncesAccount gets the "usedNonces" account.
// Account will be explicitly loaded to avoid error when it's not initialized
func (inst *IsNonceUsed) GetUsedNoncesAccount() *ag_solanago.AccountMeta {
	return inst.AccountMetaSlice.Get(0)
}

func (inst IsNonceUsed) Build() *Instruction {
	return &Instruction{BaseVariant: ag_binary.BaseVariant{
		Impl:   inst,
		TypeID: Instruction_IsNonceUsed,
	}}
}

// ValidateAndBuild validates the instruction parameters and accounts;
// if there is a validation error, it returns the error.
// Otherwise, it builds and returns the instruction.
func (inst IsNonceUsed) ValidateAndBuild() (*Instruction, error) {
	if err := inst.Validate(); err != nil {
		return nil, err
	}
	return inst.Build(), nil
}

func (inst *IsNonceUsed) Validate() error {
	// Check whether all (required) parameters are set:
	{
		if inst.Params == nil {
			return errors.New("Params parameter is not set")
		}
	}

	// Check whether all (required) accounts are set:
	{
		if inst.AccountMetaSlice[0] == nil {
			return errors.New("accounts.UsedNonces is not set")
		}
	}
	return nil
}

func (inst *IsNonceUsed) EncodeToTree(parent ag_treeout.Branches) {
	parent.Child(ag_format.Program(ProgramName, ProgramID)).
		//
		ParentFunc(func(programBranch ag_treeout.Branches) {
			programBranch.Child(ag_format.Instruction("IsNonceUsed")).
				//
				ParentFunc(func(instructionBranch ag_treeout.Branches) {

					// Parameters of the instruction:
					instructionBranch.Child("Params[len=1]").ParentFunc(func(paramsBranch ag_treeout.Branches) {
						paramsBranch.Child(ag_format.Param("Params", *inst.Params))
					})

					// Accounts of the instruction:
					instructionBranch.Child("Accounts[len=1]").ParentFunc(func(accountsBranch ag_treeout.Branches) {
						accountsBranch.Child(ag_format.Meta("usedNonces", inst.AccountMetaSlice.Get(0)))
					})
				})
		})
}

func (obj IsNonceUsed) MarshalWithEncoder(encoder *ag_binary.Encoder) (err error) {
	// Serialize `Params` param:
	err = encoder.Encode(obj.Params)
	if err != nil {
		return err
	}
	return nil
}
func (obj *IsNonceUsed) UnmarshalWithDecoder(decoder *ag_binary.Decoder) (err error) {
	// Deserialize `Params`:
	err = decoder.Decode(&obj.Params)
	if err != nil {
		return err
	}
	return nil
}

// NewIsNonceUsedInstruction declares a new IsNonceUsed instruction with the provided parameters and accounts.
func NewIsNonceUsedInstruction(
	// Parameters:
	params IsNonceUsedParams,
	// Accounts:
	usedNonces ag_solanago.PublicKey) *IsNonceUsed {
	return NewIsNonceUsedInstructionBuilder().
		SetParams(params).
		SetUsedNoncesAccount(usedNonces)
}