package types

const (
	EventTypeCreateTask   = "create_task"
	EventTypeAssignTask   = "assign_task"
	EventTypeCompleteTask = "complete_task"

	AttributeKeyTaskID   = "task_id"
	AttributeKeyCreator  = "creator"
	AttributeKeyAssignee = "assignee"
	AttributeKeyTitle    = "title"
	AttributeKeyStatus   = "status"
)
