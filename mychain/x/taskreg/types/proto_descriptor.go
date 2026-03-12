package types

import (
	"bytes"
	"compress/gzip"

	gogoproto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/encoding/protowire"
	protov2 "google.golang.org/protobuf/proto"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

// Package-level gzipped file descriptor bytes, used by Descriptor() methods.
var (
	fileDescriptor_tx      []byte
	fileDescriptor_types   []byte
	fileDescriptor_query   []byte
	fileDescriptor_genesis []byte
)

func init() {
	fileDescriptor_tx = registerTxFileDescriptor()
	fileDescriptor_types = registerTypesFileDescriptor()
	fileDescriptor_query = registerQueryFileDescriptor()
	fileDescriptor_genesis = registerGenesisFileDescriptor()
}

func strp(s string) *string { return &s }
func int32p(i int32) *int32 { return &i }

func signerOpts(signerField string) *descriptorpb.MessageOptions {
	opts := &descriptorpb.MessageOptions{}
	var raw []byte
	raw = protowire.AppendTag(raw, 11110000, protowire.BytesType)
	raw = protowire.AppendString(raw, signerField)
	opts.ProtoReflect().SetUnknown(raw)
	return opts
}

func serviceOpts() *descriptorpb.ServiceOptions {
	opts := &descriptorpb.ServiceOptions{}
	var raw []byte
	raw = protowire.AppendTag(raw, 11110001, protowire.VarintType)
	raw = protowire.AppendVarint(raw, 1)
	opts.ProtoReflect().SetUnknown(raw)
	return opts
}

func strField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     strp(name),
		Number:   int32p(num),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		JsonName: strp(name),
	}
}

func uint64Field(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     strp(name),
		Number:   int32p(num),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		JsonName: strp(name),
	}
}

func enumField(name string, num int32, typeName string) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     strp(name),
		Number:   int32p(num),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
		TypeName: strp(typeName),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		JsonName: strp(name),
	}
}

func msgField(name string, num int32, typeName string) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     strp(name),
		Number:   int32p(num),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName: strp(typeName),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		JsonName: strp(name),
	}
}

func repeatedMsgField(name string, num int32, typeName string) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     strp(name),
		Number:   int32p(num),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName: strp(typeName),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
		JsonName: strp(name),
	}
}

func registerProtoFile(fd *descriptorpb.FileDescriptorProto) []byte {
	b, err := protov2.Marshal(fd)
	if err != nil {
		panic("failed to marshal file descriptor: " + err.Error())
	}

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		panic("failed to gzip file descriptor: " + err.Error())
	}
	w.Close()

	gzipped := buf.Bytes()
	gogoproto.RegisterFile(*fd.Name, gzipped)
	return gzipped
}

// tx.proto: MessageType order: MsgCreateTask(0), MsgCreateTaskResponse(1),
// MsgAssignTask(2), MsgAssignTaskResponse(3), MsgCompleteTask(4), MsgCompleteTaskResponse(5)
func registerTxFileDescriptor() []byte {
	fd := &descriptorpb.FileDescriptorProto{
		Name:       strp("mychain/taskreg/v1/tx.proto"),
		Package:    strp("mychain.taskreg.v1"),
		Syntax:     strp("proto3"),
		Dependency: []string{"cosmos/msg/v1/msg.proto", "gogoproto/gogo.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: strp("poc-cometbft/mychain/x/taskreg/types"),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name:    strp("Msg"),
				Options: serviceOpts(),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: strp("CreateTask"), InputType: strp(".mychain.taskreg.v1.MsgCreateTask"), OutputType: strp(".mychain.taskreg.v1.MsgCreateTaskResponse")},
					{Name: strp("AssignTask"), InputType: strp(".mychain.taskreg.v1.MsgAssignTask"), OutputType: strp(".mychain.taskreg.v1.MsgAssignTaskResponse")},
					{Name: strp("CompleteTask"), InputType: strp(".mychain.taskreg.v1.MsgCompleteTask"), OutputType: strp(".mychain.taskreg.v1.MsgCompleteTaskResponse")},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:    strp("MsgCreateTask"),
				Options: signerOpts("creator"),
				Field:   []*descriptorpb.FieldDescriptorProto{strField("creator", 1), strField("title", 2), strField("description", 3)},
			},
			{
				Name:  strp("MsgCreateTaskResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{uint64Field("id", 1)},
			},
			{
				Name:    strp("MsgAssignTask"),
				Options: signerOpts("creator"),
				Field:   []*descriptorpb.FieldDescriptorProto{strField("creator", 1), uint64Field("task_id", 2), strField("assignee", 3)},
			},
			{
				Name: strp("MsgAssignTaskResponse"),
			},
			{
				Name:    strp("MsgCompleteTask"),
				Options: signerOpts("assignee"),
				Field:   []*descriptorpb.FieldDescriptorProto{strField("assignee", 1), uint64Field("task_id", 2)},
			},
			{
				Name: strp("MsgCompleteTaskResponse"),
			},
		},
	}
	return registerProtoFile(fd)
}

// types.proto: MessageType order: Task(0)
func registerTypesFileDescriptor() []byte {
	fd := &descriptorpb.FileDescriptorProto{
		Name:       strp("mychain/taskreg/v1/types.proto"),
		Package:    strp("mychain.taskreg.v1"),
		Syntax:     strp("proto3"),
		Dependency: []string{"gogoproto/gogo.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: strp("poc-cometbft/mychain/x/taskreg/types"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strp("TaskStatus"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strp("TASK_STATUS_UNSPECIFIED"), Number: int32p(0)},
					{Name: strp("TASK_STATUS_OPEN"), Number: int32p(1)},
					{Name: strp("TASK_STATUS_ASSIGNED"), Number: int32p(2)},
					{Name: strp("TASK_STATUS_COMPLETED"), Number: int32p(3)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strp("Task"),
				Field: []*descriptorpb.FieldDescriptorProto{
					uint64Field("id", 1),
					strField("title", 2),
					strField("description", 3),
					strField("creator", 4),
					strField("assignee", 5),
					enumField("status", 6, ".mychain.taskreg.v1.TaskStatus"),
				},
			},
		},
	}
	return registerProtoFile(fd)
}

// query.proto: MessageType order: QueryTaskRequest(0), QueryTaskResponse(1),
// QueryListTasksRequest(2), QueryListTasksResponse(3)
func registerQueryFileDescriptor() []byte {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strp("mychain/taskreg/v1/query.proto"),
		Package: strp("mychain.taskreg.v1"),
		Syntax:  strp("proto3"),
		Dependency: []string{
			"gogoproto/gogo.proto",
			"google/api/annotations.proto",
			"cosmos/base/query/v1beta1/pagination.proto",
			"mychain/taskreg/v1/types.proto",
		},
		Options: &descriptorpb.FileOptions{
			GoPackage: strp("poc-cometbft/mychain/x/taskreg/types"),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strp("Query"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: strp("Task"), InputType: strp(".mychain.taskreg.v1.QueryTaskRequest"), OutputType: strp(".mychain.taskreg.v1.QueryTaskResponse")},
					{Name: strp("ListTasks"), InputType: strp(".mychain.taskreg.v1.QueryListTasksRequest"), OutputType: strp(".mychain.taskreg.v1.QueryListTasksResponse")},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strp("QueryTaskRequest"), Field: []*descriptorpb.FieldDescriptorProto{uint64Field("id", 1)}},
			{Name: strp("QueryTaskResponse"), Field: []*descriptorpb.FieldDescriptorProto{msgField("task", 1, ".mychain.taskreg.v1.Task")}},
			{Name: strp("QueryListTasksRequest"), Field: []*descriptorpb.FieldDescriptorProto{msgField("pagination", 1, ".cosmos.base.query.v1beta1.PageRequest")}},
			{Name: strp("QueryListTasksResponse"), Field: []*descriptorpb.FieldDescriptorProto{repeatedMsgField("tasks", 1, ".mychain.taskreg.v1.Task"), msgField("pagination", 2, ".cosmos.base.query.v1beta1.PageResponse")}},
		},
	}
	return registerProtoFile(fd)
}

// genesis.proto: MessageType order: GenesisState(0)
func registerGenesisFileDescriptor() []byte {
	fd := &descriptorpb.FileDescriptorProto{
		Name:       strp("mychain/taskreg/v1/genesis.proto"),
		Package:    strp("mychain.taskreg.v1"),
		Syntax:     strp("proto3"),
		Dependency: []string{"gogoproto/gogo.proto", "mychain/taskreg/v1/types.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: strp("poc-cometbft/mychain/x/taskreg/types"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strp("GenesisState"),
				Field: []*descriptorpb.FieldDescriptorProto{
					repeatedMsgField("tasks", 1, ".mychain.taskreg.v1.Task"),
					uint64Field("next_task_id", 2),
				},
			},
		},
	}
	return registerProtoFile(fd)
}
