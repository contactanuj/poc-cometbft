package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrTaskNotFound  = errors.Register(ModuleName, 2, "task not found")
	ErrUnauthorized  = errors.Register(ModuleName, 3, "unauthorized")
	ErrInvalidStatus = errors.Register(ModuleName, 4, "invalid task status for operation")
)
