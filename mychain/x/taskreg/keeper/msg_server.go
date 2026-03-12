package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"poc-cometbft/mychain/x/taskreg/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (m msgServer) CreateTask(goCtx context.Context, msg *types.MsgCreateTask) (*types.MsgCreateTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	m.Logger().Info("[taskreg] CreateTask", "creator", msg.Creator, "title", msg.Title)

	id := m.GetNextTaskID(ctx)
	task := types.Task{
		Id:          id,
		Title:       msg.Title,
		Description: msg.Description,
		Creator:     msg.Creator,
		Assignee:    "",
		Status:      types.TaskStatus_TASK_STATUS_OPEN,
	}

	m.SetTask(ctx, task)
	m.SetNextTaskID(ctx, id+1)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreateTask,
		sdk.NewAttribute(types.AttributeKeyTaskID, fmt.Sprintf("%d", id)),
		sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
		sdk.NewAttribute(types.AttributeKeyTitle, msg.Title),
	))

	m.Logger().Info("[taskreg] CreateTask: success", "taskId", id, "creator", msg.Creator)
	return &types.MsgCreateTaskResponse{Id: id}, nil
}

func (m msgServer) AssignTask(goCtx context.Context, msg *types.MsgAssignTask) (*types.MsgAssignTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	m.Logger().Info("[taskreg] AssignTask", "taskId", msg.TaskId, "creator", msg.Creator, "assignee", msg.Assignee)

	task, found := m.GetTask(ctx, msg.TaskId)
	if !found {
		m.Logger().Error("[taskreg] AssignTask: task not found", "taskId", msg.TaskId)
		return nil, types.ErrTaskNotFound.Wrapf("task %d not found", msg.TaskId)
	}
	if task.Creator != msg.Creator {
		m.Logger().Error("[taskreg] AssignTask: unauthorized", "taskId", msg.TaskId, "requestor", msg.Creator, "actualCreator", task.Creator)
		return nil, types.ErrUnauthorized.Wrap("only the task creator can assign")
	}
	if task.Status != types.TaskStatus_TASK_STATUS_OPEN {
		m.Logger().Error("[taskreg] AssignTask: invalid status", "taskId", msg.TaskId, "status", task.Status)
		return nil, types.ErrInvalidStatus.Wrap("task must be OPEN to assign")
	}

	task.Assignee = msg.Assignee
	task.Status = types.TaskStatus_TASK_STATUS_ASSIGNED
	m.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAssignTask,
		sdk.NewAttribute(types.AttributeKeyTaskID, fmt.Sprintf("%d", msg.TaskId)),
		sdk.NewAttribute(types.AttributeKeyAssignee, msg.Assignee),
	))

	m.Logger().Info("[taskreg] AssignTask: success", "taskId", msg.TaskId, "assignee", msg.Assignee)
	return &types.MsgAssignTaskResponse{}, nil
}

func (m msgServer) CompleteTask(goCtx context.Context, msg *types.MsgCompleteTask) (*types.MsgCompleteTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	m.Logger().Info("[taskreg] CompleteTask", "taskId", msg.TaskId, "assignee", msg.Assignee)

	task, found := m.GetTask(ctx, msg.TaskId)
	if !found {
		m.Logger().Error("[taskreg] CompleteTask: task not found", "taskId", msg.TaskId)
		return nil, types.ErrTaskNotFound.Wrapf("task %d not found", msg.TaskId)
	}
	if task.Assignee != msg.Assignee {
		m.Logger().Error("[taskreg] CompleteTask: unauthorized", "taskId", msg.TaskId, "requestor", msg.Assignee, "actualAssignee", task.Assignee)
		return nil, types.ErrUnauthorized.Wrap("only the assignee can complete")
	}
	if task.Status != types.TaskStatus_TASK_STATUS_ASSIGNED {
		m.Logger().Error("[taskreg] CompleteTask: invalid status", "taskId", msg.TaskId, "status", task.Status)
		return nil, types.ErrInvalidStatus.Wrap("task must be ASSIGNED to complete")
	}

	task.Status = types.TaskStatus_TASK_STATUS_COMPLETED
	m.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCompleteTask,
		sdk.NewAttribute(types.AttributeKeyTaskID, fmt.Sprintf("%d", msg.TaskId)),
		sdk.NewAttribute(types.AttributeKeyStatus, "completed"),
	))

	m.Logger().Info("[taskreg] CompleteTask: success", "taskId", msg.TaskId)
	return &types.MsgCompleteTaskResponse{}, nil
}
