package types

import (
	"encoding/binary"
)

const (
	ModuleName = "taskreg"
	StoreKey   = ModuleName
	RouterKey  = ModuleName

	TaskKeyPrefix = "Task/value/"
	TaskCountKey  = "Task/count"
)

func KeyTask(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return append([]byte(TaskKeyPrefix), bz...)
}
