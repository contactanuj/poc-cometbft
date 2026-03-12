package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/types/query"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"poc-cometbft/mychain/x/taskreg/types"
)

type queryServer struct {
	Keeper
}

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (q queryServer) Task(goCtx context.Context, req *types.QueryTaskRequest) (*types.QueryTaskResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	q.Logger().Info("[taskreg] Query.Task", "id", req.Id)

	ctx := sdk.UnwrapSDKContext(goCtx)
	task, found := q.GetTask(ctx, req.Id)
	if !found {
		q.Logger().Info("[taskreg] Query.Task: not found", "id", req.Id)
		return nil, types.ErrTaskNotFound.Wrapf("task %d not found", req.Id)
	}

	q.Logger().Info("[taskreg] Query.Task: found", "id", req.Id, "status", task.Status, "title", task.Title)
	return &types.QueryTaskResponse{Task: task}, nil
}

func (q queryServer) ListTasks(goCtx context.Context, req *types.QueryListTasksRequest) (*types.QueryListTasksResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	q.Logger().Info("[taskreg] Query.ListTasks")

	ctx := sdk.UnwrapSDKContext(goCtx)

	var allTasks []types.Task
	q.IterateTasks(ctx, func(task types.Task) bool {
		allTasks = append(allTasks, task)
		return false
	})

	// Apply simple pagination
	start, end := 0, len(allTasks)
	if req.Pagination != nil && req.Pagination.Offset > 0 {
		start = int(req.Pagination.Offset)
		if start > len(allTasks) {
			start = len(allTasks)
		}
	}
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		end = start + int(req.Pagination.Limit)
		if end > len(allTasks) {
			end = len(allTasks)
		}
	}

	paginated := allTasks[start:end]
	q.Logger().Info("[taskreg] Query.ListTasks: result", "total", len(allTasks), "returned", len(paginated))

	return &types.QueryListTasksResponse{
		Tasks: paginated,
		Pagination: &query.PageResponse{
			Total: uint64(len(allTasks)),
		},
	}, nil
}
