package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgCreateTask{}
	_ sdk.Msg = &MsgAssignTask{}
	_ sdk.Msg = &MsgCompleteTask{}
)

func (msg *MsgCreateTask) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return err
	}
	if msg.Title == "" {
		return ErrInvalidStatus.Wrap("title cannot be empty")
	}
	return nil
}

func (msg *MsgAssignTask) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return err
	}
	_, err = sdk.AccAddressFromBech32(msg.Assignee)
	if err != nil {
		return err
	}
	return nil
}

func (msg *MsgCompleteTask) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Assignee)
	if err != nil {
		return err
	}
	return nil
}
