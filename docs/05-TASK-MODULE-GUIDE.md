# Task Registry Module Guide (x/taskreg)

The Task Registry is a **custom Cosmos SDK module** that demonstrates how to build native blockchain functionality for task management.

> **Analogy**: Think of it as a project management board (like Jira) where every action is recorded on a permanent, tamper-proof ledger.

---

## 1. Task Lifecycle

### State Machine

```
                   MsgCreateTask
                   (by anyone)
                        |
                        v
                  +----------+
                  |   OPEN   |  Task created, waiting for assignment
                  |  (1)     |
                  +----+-----+
                       |
                  MsgAssignTask
                  (by creator only)
                       |
                       v
                  +----------+
                  | ASSIGNED |  Task assigned to a worker
                  |  (2)     |
                  +----+-----+
                       |
                  MsgCompleteTask
                  (by assignee only)
                       |
                       v
                  +----------+
                  | COMPLETED|  Task finished
                  |  (3)     |
                  +----------+
```

### Transition Rules

| Transition        | Who Can Do It   | From Status | To Status  |
|-------------------|-----------------|-------------|------------|
| Create            | Anyone          | (none)      | OPEN       |
| Assign            | Task creator    | OPEN        | ASSIGNED   |
| Complete          | Task assignee   | ASSIGNED    | COMPLETED  |

---

## 2. Data Model

### Task (Protobuf Definition)

```protobuf
message Task {
  uint64 id          = 1;   // Auto-incrementing unique ID
  string title       = 2;   // Task title (required)
  string description = 3;   // Task description (optional)
  string creator     = 4;   // Cosmos address of the creator
  string assignee    = 5;   // Cosmos address of the assignee (empty if OPEN)
  TaskStatus status  = 6;   // Current status (OPEN, ASSIGNED, COMPLETED)
}
```

### Storage Layout

```
KV Store:
  "Task/value/\x00\x00\x00\x00\x00\x00\x00\x01" -> Task{id:1, ...} (protobuf bytes)
  "Task/value/\x00\x00\x00\x00\x00\x00\x00\x02" -> Task{id:2, ...}
  "Task/count"                                     -> uint64(3)  (next ID)
```

Tasks are stored under the `Task/value/` prefix, with the ID encoded as 8-byte big-endian. The counter `Task/count` tracks the next available ID.

---

## 3. API Reference

### POST /api/tasks -- Create a Task

Creates a new task in OPEN status.

**Request:**
```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Fix login bug",
    "description": "Users cannot login with email"
  }'
```

| Field         | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| `title`       | string | Yes      | Task title            |
| `description` | string | No       | Task description      |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "tx_hash": "A1B2C3D4E5F6..."
  }
}
```

**Errors:**
| Code | Condition                          |
|------|------------------------------------|
| 400  | Missing or empty `title`           |
| 500  | Chain transaction broadcast failed |

**Note:** The task is not immediately queryable. Wait ~5 seconds (one block) for it to be committed.

---

### GET /api/tasks -- List All Tasks

Returns all tasks with pagination.

**Request:**
```bash
curl http://localhost:8080/api/tasks
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "tasks": [
      {
        "id": "1",
        "title": "Fix login bug",
        "description": "Users cannot login with email",
        "creator": "cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za",
        "assignee": "",
        "status": "TASK_STATUS_OPEN"
      }
    ],
    "pagination": {
      "total": "1"
    }
  }
}
```

---

### GET /api/tasks/{id} -- Get a Single Task

**Request:**
```bash
curl http://localhost:8080/api/tasks/1
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "task": {
      "id": "1",
      "title": "Fix login bug",
      "description": "Users cannot login with email",
      "creator": "cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za",
      "assignee": "",
      "status": "TASK_STATUS_OPEN"
    }
  }
}
```

**Errors:**
| Code | Condition              |
|------|------------------------|
| 400  | Invalid ID format      |
| 500  | Task not found on chain|

---

### POST /api/tasks/{id}/assign -- Assign a Task

Assigns an OPEN task to a worker. Only the task creator can do this.

**Request:**
```bash
curl -X POST http://localhost:8080/api/tasks/1/assign \
  -H "Content-Type: application/json" \
  -d '{
    "assignee": "cosmos1cypznegvhp348zaqfcntmkvm0tg4pjc7kl4yxr"
  }'
```

| Field      | Type   | Required | Description              |
|------------|--------|----------|--------------------------|
| `assignee` | string | Yes      | Cosmos address of worker |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "tx_hash": "F6E5D4C3B2A1..."
  }
}
```

**Errors:**
| Code | Condition                                    |
|------|----------------------------------------------|
| 400  | Invalid ID or missing assignee               |
| 500  | Task not found, not OPEN, or unauthorized    |

---

### POST /api/tasks/{id}/complete -- Complete a Task

Marks an ASSIGNED task as COMPLETED. Only the assignee can do this.

**Request:**
```bash
curl -X POST http://localhost:8080/api/tasks/1/complete
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "tx_hash": "1A2B3C4D5E6F..."
  }
}
```

**Errors:**
| Code | Condition                                         |
|------|---------------------------------------------------|
| 400  | Invalid ID                                        |
| 500  | Task not found, not ASSIGNED, or not the assignee |

---

## 4. Internal Architecture

### How CreateTask Works Inside the Chain

```
MsgCreateTask{Creator: "cosmos1...", Title: "Fix bug", Description: "..."}
    |
    v
msg_server.go: CreateTask()
    |
    +-- keeper.GetNextTaskID(ctx)           -> returns 1
    |
    +-- Create Task{
    |     Id: 1,
    |     Title: "Fix bug",
    |     Description: "...",
    |     Creator: "cosmos1...",
    |     Status: TASK_STATUS_OPEN,
    |   }
    |
    +-- keeper.SetTask(ctx, task)           -> stores in KV store
    |
    +-- keeper.SetNextTaskID(ctx, 2)        -> increment counter
    |
    +-- Emit Event{
    |     Type: "create_task",
    |     Attributes: [
    |       {Key: "task_id", Value: "1"},
    |       {Key: "creator", Value: "cosmos1..."},
    |       {Key: "title", Value: "Fix bug"},
    |     ]
    |   }
    |
    +-- Return MsgCreateTaskResponse{Id: 1}
```

### How AssignTask Works Inside the Chain

```
MsgAssignTask{Creator: "cosmos1abc...", TaskId: 1, Assignee: "cosmos1xyz..."}
    |
    v
msg_server.go: AssignTask()
    |
    +-- keeper.GetTask(ctx, 1)              -> returns Task
    |
    +-- Validate: task.Creator == msg.Creator?   (authorization check)
    |   NO  -> return ErrUnauthorized
    |   YES -> continue
    |
    +-- Validate: task.Status == OPEN?
    |   NO  -> return ErrInvalidStatus
    |   YES -> continue
    |
    +-- task.Assignee = "cosmos1xyz..."
    +-- task.Status = TASK_STATUS_ASSIGNED
    +-- keeper.SetTask(ctx, task)
    +-- Emit Event "assign_task"
    +-- Return MsgAssignTaskResponse{}
```

---

## 5. Events

Events are emitted on-chain and can be subscribed to via WebSocket or queried via RPC.

| Event              | Attributes                        | When                    |
|--------------------|-----------------------------------|-------------------------|
| `create_task`      | task_id, creator, title           | New task created        |
| `assign_task`      | task_id, assignee                 | Task assigned to worker |
| `complete_task`    | task_id, status                   | Task marked complete    |

### Querying Events via CometBFT RPC

```bash
# Find all task creation events
curl "http://localhost:26657/tx_search?query=\"create_task.creator='cosmos1...'\"" | jq
```

---

## 6. Error Reference

| Error              | Code | When                                                |
|--------------------|------|-----------------------------------------------------|
| `ErrTaskNotFound`  | 1    | GetTask/AssignTask/CompleteTask with invalid ID      |
| `ErrUnauthorized`  | 2    | AssignTask by non-creator, CompleteTask by non-assignee |
| `ErrInvalidStatus` | 3    | Assign a non-OPEN task, Complete a non-ASSIGNED task |

---

## 7. Step-by-Step Tutorial

### Full Task Lifecycle

```bash
# 1. Create Task A
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Write documentation", "description": "Create project docs"}'

# Wait 5 seconds for block commit
sleep 5

# 2. Create Task B
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Fix CSS layout", "description": "Header misaligned on mobile"}'

sleep 5

# 3. List all tasks -- both should be OPEN
curl http://localhost:8080/api/tasks | jq

# 4. Assign Task 1
curl -X POST http://localhost:8080/api/tasks/1/assign \
  -H "Content-Type: application/json" \
  -d '{"assignee": "cosmos1cypznegvhp348zaqfcntmkvm0tg4pjc7kl4yxr"}'

sleep 5

# 5. Check Task 1 -- should be ASSIGNED
curl http://localhost:8080/api/tasks/1 | jq

# 6. Complete Task 1
curl -X POST http://localhost:8080/api/tasks/1/complete

sleep 5

# 7. Check Task 1 -- should be COMPLETED
curl http://localhost:8080/api/tasks/1 | jq

# 8. Check Task 2 -- should still be OPEN
curl http://localhost:8080/api/tasks/2 | jq
```

### Expected Final State

| Task | Title                | Status    |
|------|----------------------|-----------|
| 1    | Write documentation  | COMPLETED |
| 2    | Fix CSS layout       | OPEN      |

---

## 8. Source Code Reference

| File                                    | Purpose                                |
|-----------------------------------------|----------------------------------------|
| `mychain/x/taskreg/module.go`          | Module registration and interfaces     |
| `mychain/x/taskreg/keeper/keeper.go`   | State access (SetTask, GetTask, etc.)  |
| `mychain/x/taskreg/keeper/msg_server.go`| Message handlers (Create, Assign, Complete) |
| `mychain/x/taskreg/keeper/grpc_query.go`| Query handlers (Task, ListTasks)       |
| `mychain/x/taskreg/keeper/genesis.go`  | Genesis import/export                  |
| `mychain/x/taskreg/types/`             | All type definitions, errors, events   |
| `mychain/proto/mychain/taskreg/v1/`    | Protobuf source definitions            |
| `api-backend/handlers/tasks.go`        | HTTP handlers for task endpoints       |
| `api-backend/client/chain.go`          | Chain client (CreateTask, AssignTask, etc.) |
