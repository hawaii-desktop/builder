// Code generated by protoc-gen-go.
// source: protocol/builder.proto
// DO NOT EDIT!

/*
Package protocol is a generated protocol buffer package.

It is generated from these files:
	protocol/builder.proto

It has these top-level messages:
	BooleanMessage
	StringMessage
	StringListMessage
	SubscribeRequest
	SubscribeResponse
	UnsubscribeRequest
	UnsubscribeResponse
	JobDispatchRequest
	JobUpdateRequest
	StepResponse
	InputMessage
	OutputMessage
	CollectJobRequest
	CollectJobResponse
	VcsInfo
	PackageInfo
	ImageInfo
*/
package protocol

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Job status.
type EnumJobStatus int32

const (
	EnumJobStatus_JOB_STATUS_JUST_CREATED EnumJobStatus = 0
	EnumJobStatus_JOB_STATUS_WAITING      EnumJobStatus = 1
	EnumJobStatus_JOB_STATUS_PROCESSING   EnumJobStatus = 2
	EnumJobStatus_JOB_STATUS_SUCCESSFUL   EnumJobStatus = 3
	EnumJobStatus_JOB_STATUS_FAILED       EnumJobStatus = 4
	EnumJobStatus_JOB_STATUS_CRASHED      EnumJobStatus = 5
)

var EnumJobStatus_name = map[int32]string{
	0: "JOB_STATUS_JUST_CREATED",
	1: "JOB_STATUS_WAITING",
	2: "JOB_STATUS_PROCESSING",
	3: "JOB_STATUS_SUCCESSFUL",
	4: "JOB_STATUS_FAILED",
	5: "JOB_STATUS_CRASHED",
}
var EnumJobStatus_value = map[string]int32{
	"JOB_STATUS_JUST_CREATED": 0,
	"JOB_STATUS_WAITING":      1,
	"JOB_STATUS_PROCESSING":   2,
	"JOB_STATUS_SUCCESSFUL":   3,
	"JOB_STATUS_FAILED":       4,
	"JOB_STATUS_CRASHED":      5,
}

func (x EnumJobStatus) String() string {
	return proto.EnumName(EnumJobStatus_name, int32(x))
}

// Build target.
type EnumTargetType int32

const (
	EnumTargetType_PACKAGE EnumTargetType = 0
	EnumTargetType_IMAGE   EnumTargetType = 1
)

var EnumTargetType_name = map[int32]string{
	0: "PACKAGE",
	1: "IMAGE",
}
var EnumTargetType_value = map[string]int32{
	"PACKAGE": 0,
	"IMAGE":   1,
}

func (x EnumTargetType) String() string {
	return proto.EnumName(EnumTargetType_name, int32(x))
}

// Generic boolean message.
type BooleanMessage struct {
	// Result.
	Result bool `protobuf:"varint,1,opt,name=result" json:"result,omitempty"`
}

func (m *BooleanMessage) Reset()         { *m = BooleanMessage{} }
func (m *BooleanMessage) String() string { return proto.CompactTextString(m) }
func (*BooleanMessage) ProtoMessage()    {}

// Generic string message.
type StringMessage struct {
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
}

func (m *StringMessage) Reset()         { *m = StringMessage{} }
func (m *StringMessage) String() string { return proto.CompactTextString(m) }
func (*StringMessage) ProtoMessage()    {}

// Generic string list message.
type StringListMessage struct {
	List []string `protobuf:"bytes,1,rep,name=list" json:"list,omitempty"`
}

func (m *StringListMessage) Reset()         { *m = StringListMessage{} }
func (m *StringListMessage) String() string { return proto.CompactTextString(m) }
func (*StringListMessage) ProtoMessage()    {}

// Subscription request.
type SubscribeRequest struct {
	// Name.
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Topics.
	Types []string `protobuf:"bytes,2,rep,name=types" json:"types,omitempty"`
	// Architectures.
	Architectures []string `protobuf:"bytes,3,rep,name=architectures" json:"architectures,omitempty"`
}

func (m *SubscribeRequest) Reset()         { *m = SubscribeRequest{} }
func (m *SubscribeRequest) String() string { return proto.CompactTextString(m) }
func (*SubscribeRequest) ProtoMessage()    {}

// Subscription response.
type SubscribeResponse struct {
	// Slave identifier.
	Id uint64 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
}

func (m *SubscribeResponse) Reset()         { *m = SubscribeResponse{} }
func (m *SubscribeResponse) String() string { return proto.CompactTextString(m) }
func (*SubscribeResponse) ProtoMessage()    {}

// Unsubscription request.
type UnsubscribeRequest struct {
	// Slave identifier.
	Id uint64 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
}

func (m *UnsubscribeRequest) Reset()         { *m = UnsubscribeRequest{} }
func (m *UnsubscribeRequest) String() string { return proto.CompactTextString(m) }
func (*UnsubscribeRequest) ProtoMessage()    {}

// Unsubscription response.
type UnsubscribeResponse struct {
	// Result.
	Result bool `protobuf:"varint,1,opt,name=result" json:"result,omitempty"`
}

func (m *UnsubscribeResponse) Reset()         { *m = UnsubscribeResponse{} }
func (m *UnsubscribeResponse) String() string { return proto.CompactTextString(m) }
func (*UnsubscribeResponse) ProtoMessage()    {}

// Contains information on the job that has to be processed by
// the slave receiving this.
type JobDispatchRequest struct {
	// Identifier.
	Id uint64 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	// Payload.
	//
	// Types that are valid to be assigned to Payload:
	//	*JobDispatchRequest_Package
	//	*JobDispatchRequest_Image
	Payload isJobDispatchRequest_Payload `protobuf_oneof:"payload"`
}

func (m *JobDispatchRequest) Reset()         { *m = JobDispatchRequest{} }
func (m *JobDispatchRequest) String() string { return proto.CompactTextString(m) }
func (*JobDispatchRequest) ProtoMessage()    {}

type isJobDispatchRequest_Payload interface {
	isJobDispatchRequest_Payload()
}

type JobDispatchRequest_Package struct {
	Package *PackageInfo `protobuf:"bytes,2,opt,name=package,oneof"`
}
type JobDispatchRequest_Image struct {
	Image *ImageInfo `protobuf:"bytes,3,opt,name=image,oneof"`
}

func (*JobDispatchRequest_Package) isJobDispatchRequest_Payload() {}
func (*JobDispatchRequest_Image) isJobDispatchRequest_Payload()   {}

func (m *JobDispatchRequest) GetPayload() isJobDispatchRequest_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *JobDispatchRequest) GetPackage() *PackageInfo {
	if x, ok := m.GetPayload().(*JobDispatchRequest_Package); ok {
		return x.Package
	}
	return nil
}

func (m *JobDispatchRequest) GetImage() *ImageInfo {
	if x, ok := m.GetPayload().(*JobDispatchRequest_Image); ok {
		return x.Image
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*JobDispatchRequest) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), []interface{}) {
	return _JobDispatchRequest_OneofMarshaler, _JobDispatchRequest_OneofUnmarshaler, []interface{}{
		(*JobDispatchRequest_Package)(nil),
		(*JobDispatchRequest_Image)(nil),
	}
}

func _JobDispatchRequest_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*JobDispatchRequest)
	// payload
	switch x := m.Payload.(type) {
	case *JobDispatchRequest_Package:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Package); err != nil {
			return err
		}
	case *JobDispatchRequest_Image:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Image); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("JobDispatchRequest.Payload has unexpected type %T", x)
	}
	return nil
}

func _JobDispatchRequest_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*JobDispatchRequest)
	switch tag {
	case 2: // payload.package
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(PackageInfo)
		err := b.DecodeMessage(msg)
		m.Payload = &JobDispatchRequest_Package{msg}
		return true, err
	case 3: // payload.image
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(ImageInfo)
		err := b.DecodeMessage(msg)
		m.Payload = &JobDispatchRequest_Image{msg}
		return true, err
	default:
		return false, nil
	}
}

// Contains updated information on a job being processed.
type JobUpdateRequest struct {
	// Slave identifier.
	SlaveId uint64 `protobuf:"varint,1,opt,name=slave_id" json:"slave_id,omitempty"`
	// Identifier.
	Id uint64 `protobuf:"varint,2,opt,name=id" json:"id,omitempty"`
	// Current status of the job.
	Status EnumJobStatus `protobuf:"varint,3,opt,name=status,enum=protocol.EnumJobStatus" json:"status,omitempty"`
}

func (m *JobUpdateRequest) Reset()         { *m = JobUpdateRequest{} }
func (m *JobUpdateRequest) String() string { return proto.CompactTextString(m) }
func (*JobUpdateRequest) ProtoMessage()    {}

// Contains updated information on a build step being executed.
type StepResponse struct {
	// Job identifier.
	JobId uint64 `protobuf:"varint,1,opt,name=job_id" json:"job_id,omitempty"`
	// Name.
	Name string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	// Whether it's still running.
	Running bool `protobuf:"varint,3,opt,name=running" json:"running,omitempty"`
	// When it has started (nanoseconds since Epoch).
	Started int64 `protobuf:"varint,4,opt,name=started" json:"started,omitempty"`
	// When it has finished (nanoseconds since Epoch).
	Finished int64 `protobuf:"varint,5,opt,name=finished" json:"finished,omitempty"`
	// Optional summary of this step.
	Summary map[string]string `protobuf:"bytes,6,rep,name=summary" json:"summary,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Other optional logs.
	Logs map[string][]byte `protobuf:"bytes,7,rep,name=logs" json:"logs,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (m *StepResponse) Reset()         { *m = StepResponse{} }
func (m *StepResponse) String() string { return proto.CompactTextString(m) }
func (*StepResponse) ProtoMessage()    {}

func (m *StepResponse) GetSummary() map[string]string {
	if m != nil {
		return m.Summary
	}
	return nil
}

func (m *StepResponse) GetLogs() map[string][]byte {
	if m != nil {
		return m.Logs
	}
	return nil
}

// Communication from slave to master.
type InputMessage struct {
	// Types that are valid to be assigned to Payload:
	//	*InputMessage_Subscription
	//	*InputMessage_JobUpdate
	//	*InputMessage_StepUpdate
	Payload isInputMessage_Payload `protobuf_oneof:"payload"`
}

func (m *InputMessage) Reset()         { *m = InputMessage{} }
func (m *InputMessage) String() string { return proto.CompactTextString(m) }
func (*InputMessage) ProtoMessage()    {}

type isInputMessage_Payload interface {
	isInputMessage_Payload()
}

type InputMessage_Subscription struct {
	Subscription *SubscribeRequest `protobuf:"bytes,1,opt,name=subscription,oneof"`
}
type InputMessage_JobUpdate struct {
	JobUpdate *JobUpdateRequest `protobuf:"bytes,2,opt,name=job_update,oneof"`
}
type InputMessage_StepUpdate struct {
	StepUpdate *StepResponse `protobuf:"bytes,3,opt,name=step_update,oneof"`
}

func (*InputMessage_Subscription) isInputMessage_Payload() {}
func (*InputMessage_JobUpdate) isInputMessage_Payload()    {}
func (*InputMessage_StepUpdate) isInputMessage_Payload()   {}

func (m *InputMessage) GetPayload() isInputMessage_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *InputMessage) GetSubscription() *SubscribeRequest {
	if x, ok := m.GetPayload().(*InputMessage_Subscription); ok {
		return x.Subscription
	}
	return nil
}

func (m *InputMessage) GetJobUpdate() *JobUpdateRequest {
	if x, ok := m.GetPayload().(*InputMessage_JobUpdate); ok {
		return x.JobUpdate
	}
	return nil
}

func (m *InputMessage) GetStepUpdate() *StepResponse {
	if x, ok := m.GetPayload().(*InputMessage_StepUpdate); ok {
		return x.StepUpdate
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*InputMessage) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), []interface{}) {
	return _InputMessage_OneofMarshaler, _InputMessage_OneofUnmarshaler, []interface{}{
		(*InputMessage_Subscription)(nil),
		(*InputMessage_JobUpdate)(nil),
		(*InputMessage_StepUpdate)(nil),
	}
}

func _InputMessage_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*InputMessage)
	// payload
	switch x := m.Payload.(type) {
	case *InputMessage_Subscription:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Subscription); err != nil {
			return err
		}
	case *InputMessage_JobUpdate:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.JobUpdate); err != nil {
			return err
		}
	case *InputMessage_StepUpdate:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.StepUpdate); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("InputMessage.Payload has unexpected type %T", x)
	}
	return nil
}

func _InputMessage_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*InputMessage)
	switch tag {
	case 1: // payload.subscription
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(SubscribeRequest)
		err := b.DecodeMessage(msg)
		m.Payload = &InputMessage_Subscription{msg}
		return true, err
	case 2: // payload.job_update
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(JobUpdateRequest)
		err := b.DecodeMessage(msg)
		m.Payload = &InputMessage_JobUpdate{msg}
		return true, err
	case 3: // payload.step_update
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(StepResponse)
		err := b.DecodeMessage(msg)
		m.Payload = &InputMessage_StepUpdate{msg}
		return true, err
	default:
		return false, nil
	}
}

// Communication from master to slave.
type OutputMessage struct {
	// Types that are valid to be assigned to Payload:
	//	*OutputMessage_Subscription
	//	*OutputMessage_JobDispatch
	Payload isOutputMessage_Payload `protobuf_oneof:"payload"`
}

func (m *OutputMessage) Reset()         { *m = OutputMessage{} }
func (m *OutputMessage) String() string { return proto.CompactTextString(m) }
func (*OutputMessage) ProtoMessage()    {}

type isOutputMessage_Payload interface {
	isOutputMessage_Payload()
}

type OutputMessage_Subscription struct {
	Subscription *SubscribeResponse `protobuf:"bytes,1,opt,name=subscription,oneof"`
}
type OutputMessage_JobDispatch struct {
	JobDispatch *JobDispatchRequest `protobuf:"bytes,2,opt,name=job_dispatch,oneof"`
}

func (*OutputMessage_Subscription) isOutputMessage_Payload() {}
func (*OutputMessage_JobDispatch) isOutputMessage_Payload()  {}

func (m *OutputMessage) GetPayload() isOutputMessage_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *OutputMessage) GetSubscription() *SubscribeResponse {
	if x, ok := m.GetPayload().(*OutputMessage_Subscription); ok {
		return x.Subscription
	}
	return nil
}

func (m *OutputMessage) GetJobDispatch() *JobDispatchRequest {
	if x, ok := m.GetPayload().(*OutputMessage_JobDispatch); ok {
		return x.JobDispatch
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*OutputMessage) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), []interface{}) {
	return _OutputMessage_OneofMarshaler, _OutputMessage_OneofUnmarshaler, []interface{}{
		(*OutputMessage_Subscription)(nil),
		(*OutputMessage_JobDispatch)(nil),
	}
}

func _OutputMessage_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*OutputMessage)
	// payload
	switch x := m.Payload.(type) {
	case *OutputMessage_Subscription:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Subscription); err != nil {
			return err
		}
	case *OutputMessage_JobDispatch:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.JobDispatch); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("OutputMessage.Payload has unexpected type %T", x)
	}
	return nil
}

func _OutputMessage_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*OutputMessage)
	switch tag {
	case 1: // payload.subscription
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(SubscribeResponse)
		err := b.DecodeMessage(msg)
		m.Payload = &OutputMessage_Subscription{msg}
		return true, err
	case 2: // payload.job_dispatch
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(JobDispatchRequest)
		err := b.DecodeMessage(msg)
		m.Payload = &OutputMessage_JobDispatch{msg}
		return true, err
	default:
		return false, nil
	}
}

// CollectJob request.
type CollectJobRequest struct {
	// Target name
	Target string `protobuf:"bytes,1,opt,name=target" json:"target,omitempty"`
	// Target architecture.
	Architecture string `protobuf:"bytes,2,opt,name=architecture" json:"architecture,omitempty"`
	// Target type.
	Type EnumTargetType `protobuf:"varint,3,opt,name=type,enum=protocol.EnumTargetType" json:"type,omitempty"`
}

func (m *CollectJobRequest) Reset()         { *m = CollectJobRequest{} }
func (m *CollectJobRequest) String() string { return proto.CompactTextString(m) }
func (*CollectJobRequest) ProtoMessage()    {}

// CollectJob response.
type CollectJobResponse struct {
	// Result.
	Result bool `protobuf:"varint,1,opt,name=result" json:"result,omitempty"`
	// Identifier.
	Id uint64 `protobuf:"varint,2,opt,name=id" json:"id,omitempty"`
}

func (m *CollectJobResponse) Reset()         { *m = CollectJobResponse{} }
func (m *CollectJobResponse) String() string { return proto.CompactTextString(m) }
func (*CollectJobResponse) ProtoMessage()    {}

// VCS information.
type VcsInfo struct {
	Url    string `protobuf:"bytes,1,opt,name=url" json:"url,omitempty"`
	Branch string `protobuf:"bytes,2,opt,name=branch" json:"branch,omitempty"`
}

func (m *VcsInfo) Reset()         { *m = VcsInfo{} }
func (m *VcsInfo) String() string { return proto.CompactTextString(m) }
func (*VcsInfo) ProtoMessage()    {}

// Package information.
type PackageInfo struct {
	// Name.
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Architectures supported.
	Architectures []string `protobuf:"bytes,2,rep,name=architectures" json:"architectures,omitempty"`
	// Is it a CI package?
	Ci bool `protobuf:"varint,3,opt,name=ci" json:"ci,omitempty"`
	// VCS for packaging.
	Vcs *VcsInfo `protobuf:"bytes,4,opt,name=vcs" json:"vcs,omitempty"`
	// VCS for upstream (only for CI).
	UpstreamVcs *VcsInfo `protobuf:"bytes,5,opt,name=upstream_vcs" json:"upstream_vcs,omitempty"`
}

func (m *PackageInfo) Reset()         { *m = PackageInfo{} }
func (m *PackageInfo) String() string { return proto.CompactTextString(m) }
func (*PackageInfo) ProtoMessage()    {}

func (m *PackageInfo) GetVcs() *VcsInfo {
	if m != nil {
		return m.Vcs
	}
	return nil
}

func (m *PackageInfo) GetUpstreamVcs() *VcsInfo {
	if m != nil {
		return m.UpstreamVcs
	}
	return nil
}

// Image information.
type ImageInfo struct {
	// Name.
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Description.
	Description string `protobuf:"bytes,2,opt,name=description" json:"description,omitempty"`
	// Architectures supported.
	Architectures []string `protobuf:"bytes,3,rep,name=architectures" json:"architectures,omitempty"`
	// VCS with build scripts.
	Vcs *VcsInfo `protobuf:"bytes,4,opt,name=vcs" json:"vcs,omitempty"`
}

func (m *ImageInfo) Reset()         { *m = ImageInfo{} }
func (m *ImageInfo) String() string { return proto.CompactTextString(m) }
func (*ImageInfo) ProtoMessage()    {}

func (m *ImageInfo) GetVcs() *VcsInfo {
	if m != nil {
		return m.Vcs
	}
	return nil
}

func init() {
	proto.RegisterEnum("protocol.EnumJobStatus", EnumJobStatus_name, EnumJobStatus_value)
	proto.RegisterEnum("protocol.EnumTargetType", EnumTargetType_name, EnumTargetType_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Client API for Builder service

type BuilderClient interface {
	// Subscribe to the master.
	//
	// A slave calls this procedure to register itself on the master, which will
	// assign an identifier and reply with a Registration message.
	// Keep in mind that identifiers are valid until master is restarted.
	//
	// Once a slave has subscribed a full duplex communication is established
	// until the slave unsubscribe or the master quits, in that case the slave
	// detects that the connection is no longer valid and will resubscribe if
	// and when master comes up again.
	//
	// Master sends jobs to be processed through the stream as they are collected
	// and dispatched.  Jobs are dispatched to slaves whose capacity has not been
	// reached yet and whose architecture and channels match.
	Subscribe(ctx context.Context, opts ...grpc.CallOption) (Builder_SubscribeClient, error)
	// Unregister a slave.
	//
	// Slave class this procedure to unregister itself.
	// Master replies indicating whether the operation succeded or not.
	Unsubscribe(ctx context.Context, in *UnsubscribeRequest, opts ...grpc.CallOption) (*UnsubscribeResponse, error)
	// Send a job to the collector.
	//
	// Master will enqueue a new job and the dispatcher will find a suitable
	// slave and dispatch to it.
	CollectJob(ctx context.Context, in *CollectJobRequest, opts ...grpc.CallOption) (*CollectJobResponse, error)
	// Add or update a package.
	//
	// Store package information so that it can be referenced later when
	// scheduling a job.
	AddPackage(ctx context.Context, in *PackageInfo, opts ...grpc.CallOption) (*BooleanMessage, error)
	// Remove a package.
	//
	// Remove package information.
	RemovePackage(ctx context.Context, in *StringMessage, opts ...grpc.CallOption) (*BooleanMessage, error)
	// List packages.
	//
	// Return the list of packages and their information, matching the
	// regular expression passed as argument.
	// With an empty string the full list of packages will be retrieved.
	ListPackages(ctx context.Context, in *StringMessage, opts ...grpc.CallOption) (Builder_ListPackagesClient, error)
	// Add or update an image.
	//
	// Store image information so that it can be referenced later when
	// scheduling a job.
	AddImage(ctx context.Context, in *ImageInfo, opts ...grpc.CallOption) (*BooleanMessage, error)
	// Remove an image.
	//
	// Remove image information.
	RemoveImage(ctx context.Context, in *StringMessage, opts ...grpc.CallOption) (*BooleanMessage, error)
	// List images.
	//
	// Return the list of images and their information, matching the
	// regular expression passed as argument.
	// With an empty string the full list of images will be retrieved.
	ListImages(ctx context.Context, in *StringMessage, opts ...grpc.CallOption) (Builder_ListImagesClient, error)
}

type builderClient struct {
	cc *grpc.ClientConn
}

func NewBuilderClient(cc *grpc.ClientConn) BuilderClient {
	return &builderClient{cc}
}

func (c *builderClient) Subscribe(ctx context.Context, opts ...grpc.CallOption) (Builder_SubscribeClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Builder_serviceDesc.Streams[0], c.cc, "/protocol.Builder/Subscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &builderSubscribeClient{stream}
	return x, nil
}

type Builder_SubscribeClient interface {
	Send(*InputMessage) error
	Recv() (*OutputMessage, error)
	grpc.ClientStream
}

type builderSubscribeClient struct {
	grpc.ClientStream
}

func (x *builderSubscribeClient) Send(m *InputMessage) error {
	return x.ClientStream.SendMsg(m)
}

func (x *builderSubscribeClient) Recv() (*OutputMessage, error) {
	m := new(OutputMessage)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *builderClient) Unsubscribe(ctx context.Context, in *UnsubscribeRequest, opts ...grpc.CallOption) (*UnsubscribeResponse, error) {
	out := new(UnsubscribeResponse)
	err := grpc.Invoke(ctx, "/protocol.Builder/Unsubscribe", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *builderClient) CollectJob(ctx context.Context, in *CollectJobRequest, opts ...grpc.CallOption) (*CollectJobResponse, error) {
	out := new(CollectJobResponse)
	err := grpc.Invoke(ctx, "/protocol.Builder/CollectJob", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *builderClient) AddPackage(ctx context.Context, in *PackageInfo, opts ...grpc.CallOption) (*BooleanMessage, error) {
	out := new(BooleanMessage)
	err := grpc.Invoke(ctx, "/protocol.Builder/AddPackage", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *builderClient) RemovePackage(ctx context.Context, in *StringMessage, opts ...grpc.CallOption) (*BooleanMessage, error) {
	out := new(BooleanMessage)
	err := grpc.Invoke(ctx, "/protocol.Builder/RemovePackage", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *builderClient) ListPackages(ctx context.Context, in *StringMessage, opts ...grpc.CallOption) (Builder_ListPackagesClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Builder_serviceDesc.Streams[1], c.cc, "/protocol.Builder/ListPackages", opts...)
	if err != nil {
		return nil, err
	}
	x := &builderListPackagesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Builder_ListPackagesClient interface {
	Recv() (*PackageInfo, error)
	grpc.ClientStream
}

type builderListPackagesClient struct {
	grpc.ClientStream
}

func (x *builderListPackagesClient) Recv() (*PackageInfo, error) {
	m := new(PackageInfo)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *builderClient) AddImage(ctx context.Context, in *ImageInfo, opts ...grpc.CallOption) (*BooleanMessage, error) {
	out := new(BooleanMessage)
	err := grpc.Invoke(ctx, "/protocol.Builder/AddImage", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *builderClient) RemoveImage(ctx context.Context, in *StringMessage, opts ...grpc.CallOption) (*BooleanMessage, error) {
	out := new(BooleanMessage)
	err := grpc.Invoke(ctx, "/protocol.Builder/RemoveImage", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *builderClient) ListImages(ctx context.Context, in *StringMessage, opts ...grpc.CallOption) (Builder_ListImagesClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Builder_serviceDesc.Streams[2], c.cc, "/protocol.Builder/ListImages", opts...)
	if err != nil {
		return nil, err
	}
	x := &builderListImagesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Builder_ListImagesClient interface {
	Recv() (*ImageInfo, error)
	grpc.ClientStream
}

type builderListImagesClient struct {
	grpc.ClientStream
}

func (x *builderListImagesClient) Recv() (*ImageInfo, error) {
	m := new(ImageInfo)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server API for Builder service

type BuilderServer interface {
	// Subscribe to the master.
	//
	// A slave calls this procedure to register itself on the master, which will
	// assign an identifier and reply with a Registration message.
	// Keep in mind that identifiers are valid until master is restarted.
	//
	// Once a slave has subscribed a full duplex communication is established
	// until the slave unsubscribe or the master quits, in that case the slave
	// detects that the connection is no longer valid and will resubscribe if
	// and when master comes up again.
	//
	// Master sends jobs to be processed through the stream as they are collected
	// and dispatched.  Jobs are dispatched to slaves whose capacity has not been
	// reached yet and whose architecture and channels match.
	Subscribe(Builder_SubscribeServer) error
	// Unregister a slave.
	//
	// Slave class this procedure to unregister itself.
	// Master replies indicating whether the operation succeded or not.
	Unsubscribe(context.Context, *UnsubscribeRequest) (*UnsubscribeResponse, error)
	// Send a job to the collector.
	//
	// Master will enqueue a new job and the dispatcher will find a suitable
	// slave and dispatch to it.
	CollectJob(context.Context, *CollectJobRequest) (*CollectJobResponse, error)
	// Add or update a package.
	//
	// Store package information so that it can be referenced later when
	// scheduling a job.
	AddPackage(context.Context, *PackageInfo) (*BooleanMessage, error)
	// Remove a package.
	//
	// Remove package information.
	RemovePackage(context.Context, *StringMessage) (*BooleanMessage, error)
	// List packages.
	//
	// Return the list of packages and their information, matching the
	// regular expression passed as argument.
	// With an empty string the full list of packages will be retrieved.
	ListPackages(*StringMessage, Builder_ListPackagesServer) error
	// Add or update an image.
	//
	// Store image information so that it can be referenced later when
	// scheduling a job.
	AddImage(context.Context, *ImageInfo) (*BooleanMessage, error)
	// Remove an image.
	//
	// Remove image information.
	RemoveImage(context.Context, *StringMessage) (*BooleanMessage, error)
	// List images.
	//
	// Return the list of images and their information, matching the
	// regular expression passed as argument.
	// With an empty string the full list of images will be retrieved.
	ListImages(*StringMessage, Builder_ListImagesServer) error
}

func RegisterBuilderServer(s *grpc.Server, srv BuilderServer) {
	s.RegisterService(&_Builder_serviceDesc, srv)
}

func _Builder_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BuilderServer).Subscribe(&builderSubscribeServer{stream})
}

type Builder_SubscribeServer interface {
	Send(*OutputMessage) error
	Recv() (*InputMessage, error)
	grpc.ServerStream
}

type builderSubscribeServer struct {
	grpc.ServerStream
}

func (x *builderSubscribeServer) Send(m *OutputMessage) error {
	return x.ServerStream.SendMsg(m)
}

func (x *builderSubscribeServer) Recv() (*InputMessage, error) {
	m := new(InputMessage)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Builder_Unsubscribe_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(UnsubscribeRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(BuilderServer).Unsubscribe(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Builder_CollectJob_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(CollectJobRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(BuilderServer).CollectJob(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Builder_AddPackage_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(PackageInfo)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(BuilderServer).AddPackage(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Builder_RemovePackage_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(StringMessage)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(BuilderServer).RemovePackage(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Builder_ListPackages_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StringMessage)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BuilderServer).ListPackages(m, &builderListPackagesServer{stream})
}

type Builder_ListPackagesServer interface {
	Send(*PackageInfo) error
	grpc.ServerStream
}

type builderListPackagesServer struct {
	grpc.ServerStream
}

func (x *builderListPackagesServer) Send(m *PackageInfo) error {
	return x.ServerStream.SendMsg(m)
}

func _Builder_AddImage_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(ImageInfo)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(BuilderServer).AddImage(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Builder_RemoveImage_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(StringMessage)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(BuilderServer).RemoveImage(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Builder_ListImages_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StringMessage)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BuilderServer).ListImages(m, &builderListImagesServer{stream})
}

type Builder_ListImagesServer interface {
	Send(*ImageInfo) error
	grpc.ServerStream
}

type builderListImagesServer struct {
	grpc.ServerStream
}

func (x *builderListImagesServer) Send(m *ImageInfo) error {
	return x.ServerStream.SendMsg(m)
}

var _Builder_serviceDesc = grpc.ServiceDesc{
	ServiceName: "protocol.Builder",
	HandlerType: (*BuilderServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Unsubscribe",
			Handler:    _Builder_Unsubscribe_Handler,
		},
		{
			MethodName: "CollectJob",
			Handler:    _Builder_CollectJob_Handler,
		},
		{
			MethodName: "AddPackage",
			Handler:    _Builder_AddPackage_Handler,
		},
		{
			MethodName: "RemovePackage",
			Handler:    _Builder_RemovePackage_Handler,
		},
		{
			MethodName: "AddImage",
			Handler:    _Builder_AddImage_Handler,
		},
		{
			MethodName: "RemoveImage",
			Handler:    _Builder_RemoveImage_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Subscribe",
			Handler:       _Builder_Subscribe_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "ListPackages",
			Handler:       _Builder_ListPackages_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "ListImages",
			Handler:       _Builder_ListImages_Handler,
			ServerStreams: true,
		},
	},
}
