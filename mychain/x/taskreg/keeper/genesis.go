package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"poc-cometbft/mychain/x/taskreg/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	k.Logger().Info("[taskreg] InitGenesis", "taskCount", len(gs.Tasks), "nextTaskId", gs.NextTaskId)
	for _, task := range gs.Tasks {
		k.SetTask(ctx, task)
	}
	k.SetNextTaskID(ctx, gs.NextTaskId)
	k.Logger().Info("[taskreg] InitGenesis: complete")
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	k.Logger().Info("[taskreg] ExportGenesis")
	var tasks []types.Task
	k.IterateTasks(ctx, func(task types.Task) bool {
		tasks = append(tasks, task)
		return false
	})
	return &types.GenesisState{
		Tasks:      tasks,
		NextTaskId: k.GetNextTaskID(ctx),
	}
}
