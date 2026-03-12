package keeper

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"poc-cometbft/mychain/x/taskreg/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger
	bankKeeper   types.BankKeeper
	authority    string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		logger:       logger,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) SetTask(ctx sdk.Context, task types.Task) {
	k.Logger().Debug("[taskreg-store] SetTask", "id", task.Id, "title", task.Title, "status", task.Status)
	st := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&task)
	st.Set(types.KeyTask(task.Id), bz)
}

func (k Keeper) GetTask(ctx sdk.Context, id uint64) (types.Task, bool) {
	st := k.storeService.OpenKVStore(ctx)
	bz, err := st.Get(types.KeyTask(id))
	if err != nil || bz == nil {
		k.Logger().Debug("[taskreg-store] GetTask: not found", "id", id)
		return types.Task{}, false
	}
	var task types.Task
	k.cdc.MustUnmarshal(bz, &task)
	k.Logger().Debug("[taskreg-store] GetTask: found", "id", id, "status", task.Status)
	return task, true
}

func (k Keeper) GetNextTaskID(ctx sdk.Context) uint64 {
	st := k.storeService.OpenKVStore(ctx)
	bz, err := st.Get([]byte(types.TaskCountKey))
	if err != nil || bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetNextTaskID(ctx sdk.Context, id uint64) {
	st := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	st.Set([]byte(types.TaskCountKey), bz)
}

func (k Keeper) IterateTasks(ctx sdk.Context, cb func(task types.Task) bool) {
	st := k.storeService.OpenKVStore(ctx)
	iter, err := st.Iterator([]byte(types.TaskKeyPrefix), nil)
	if err != nil {
		return
	}
	defer iter.Close()

	prefix := []byte(types.TaskKeyPrefix)
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		// Stop if we've gone past the prefix
		if len(key) < len(prefix) {
			break
		}
		match := true
		for i := 0; i < len(prefix); i++ {
			if key[i] != prefix[i] {
				match = false
				break
			}
		}
		if !match {
			break
		}

		var task types.Task
		k.cdc.MustUnmarshal(iter.Value(), &task)
		if cb(task) {
			break
		}
	}
}
