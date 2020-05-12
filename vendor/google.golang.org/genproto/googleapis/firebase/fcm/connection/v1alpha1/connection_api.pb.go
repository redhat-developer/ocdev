// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/firebase/fcm/connection/v1alpha1/connection_api.proto

package connection // import "google.golang.org/genproto/googleapis/firebase/fcm/connection/v1alpha1"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import timestamp "github.com/golang/protobuf/ptypes/timestamp"
import _ "google.golang.org/genproto/googleapis/api/annotations"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Request sent to FCM from the connected client.
type UpstreamRequest struct {
	// The type of request the client is making to FCM.
	//
	// Types that are valid to be assigned to RequestType:
	//	*UpstreamRequest_Ack
	RequestType          isUpstreamRequest_RequestType `protobuf_oneof:"request_type"`
	XXX_NoUnkeyedLiteral struct{}                      `json:"-"`
	XXX_unrecognized     []byte                        `json:"-"`
	XXX_sizecache        int32                         `json:"-"`
}

func (m *UpstreamRequest) Reset()         { *m = UpstreamRequest{} }
func (m *UpstreamRequest) String() string { return proto.CompactTextString(m) }
func (*UpstreamRequest) ProtoMessage()    {}
func (*UpstreamRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_connection_api_e2e68185fc0238ce, []int{0}
}
func (m *UpstreamRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UpstreamRequest.Unmarshal(m, b)
}
func (m *UpstreamRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UpstreamRequest.Marshal(b, m, deterministic)
}
func (dst *UpstreamRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UpstreamRequest.Merge(dst, src)
}
func (m *UpstreamRequest) XXX_Size() int {
	return xxx_messageInfo_UpstreamRequest.Size(m)
}
func (m *UpstreamRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_UpstreamRequest.DiscardUnknown(m)
}

var xxx_messageInfo_UpstreamRequest proto.InternalMessageInfo

type isUpstreamRequest_RequestType interface {
	isUpstreamRequest_RequestType()
}

type UpstreamRequest_Ack struct {
	Ack *Ack `protobuf:"bytes,1,opt,name=ack,proto3,oneof"`
}

func (*UpstreamRequest_Ack) isUpstreamRequest_RequestType() {}

func (m *UpstreamRequest) GetRequestType() isUpstreamRequest_RequestType {
	if m != nil {
		return m.RequestType
	}
	return nil
}

func (m *UpstreamRequest) GetAck() *Ack {
	if x, ok := m.GetRequestType().(*UpstreamRequest_Ack); ok {
		return x.Ack
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*UpstreamRequest) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _UpstreamRequest_OneofMarshaler, _UpstreamRequest_OneofUnmarshaler, _UpstreamRequest_OneofSizer, []interface{}{
		(*UpstreamRequest_Ack)(nil),
	}
}

func _UpstreamRequest_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*UpstreamRequest)
	// request_type
	switch x := m.RequestType.(type) {
	case *UpstreamRequest_Ack:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Ack); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("UpstreamRequest.RequestType has unexpected type %T", x)
	}
	return nil
}

func _UpstreamRequest_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*UpstreamRequest)
	switch tag {
	case 1: // request_type.ack
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(Ack)
		err := b.DecodeMessage(msg)
		m.RequestType = &UpstreamRequest_Ack{msg}
		return true, err
	default:
		return false, nil
	}
}

func _UpstreamRequest_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*UpstreamRequest)
	// request_type
	switch x := m.RequestType.(type) {
	case *UpstreamRequest_Ack:
		s := proto.Size(x.Ack)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

// Response sent to the connected client from FCM.
type DownstreamResponse struct {
	// The type of response FCM is sending to the client.
	//
	// Types that are valid to be assigned to ResponseType:
	//	*DownstreamResponse_Message
	ResponseType         isDownstreamResponse_ResponseType `protobuf_oneof:"response_type"`
	XXX_NoUnkeyedLiteral struct{}                          `json:"-"`
	XXX_unrecognized     []byte                            `json:"-"`
	XXX_sizecache        int32                             `json:"-"`
}

func (m *DownstreamResponse) Reset()         { *m = DownstreamResponse{} }
func (m *DownstreamResponse) String() string { return proto.CompactTextString(m) }
func (*DownstreamResponse) ProtoMessage()    {}
func (*DownstreamResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_connection_api_e2e68185fc0238ce, []int{1}
}
func (m *DownstreamResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DownstreamResponse.Unmarshal(m, b)
}
func (m *DownstreamResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DownstreamResponse.Marshal(b, m, deterministic)
}
func (dst *DownstreamResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DownstreamResponse.Merge(dst, src)
}
func (m *DownstreamResponse) XXX_Size() int {
	return xxx_messageInfo_DownstreamResponse.Size(m)
}
func (m *DownstreamResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DownstreamResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DownstreamResponse proto.InternalMessageInfo

type isDownstreamResponse_ResponseType interface {
	isDownstreamResponse_ResponseType()
}

type DownstreamResponse_Message struct {
	Message *Message `protobuf:"bytes,1,opt,name=message,proto3,oneof"`
}

func (*DownstreamResponse_Message) isDownstreamResponse_ResponseType() {}

func (m *DownstreamResponse) GetResponseType() isDownstreamResponse_ResponseType {
	if m != nil {
		return m.ResponseType
	}
	return nil
}

func (m *DownstreamResponse) GetMessage() *Message {
	if x, ok := m.GetResponseType().(*DownstreamResponse_Message); ok {
		return x.Message
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*DownstreamResponse) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _DownstreamResponse_OneofMarshaler, _DownstreamResponse_OneofUnmarshaler, _DownstreamResponse_OneofSizer, []interface{}{
		(*DownstreamResponse_Message)(nil),
	}
}

func _DownstreamResponse_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*DownstreamResponse)
	// response_type
	switch x := m.ResponseType.(type) {
	case *DownstreamResponse_Message:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Message); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("DownstreamResponse.ResponseType has unexpected type %T", x)
	}
	return nil
}

func _DownstreamResponse_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*DownstreamResponse)
	switch tag {
	case 1: // response_type.message
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(Message)
		err := b.DecodeMessage(msg)
		m.ResponseType = &DownstreamResponse_Message{msg}
		return true, err
	default:
		return false, nil
	}
}

func _DownstreamResponse_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*DownstreamResponse)
	// response_type
	switch x := m.ResponseType.(type) {
	case *DownstreamResponse_Message:
		s := proto.Size(x.Message)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

// Acknowledgement to indicate a client successfully received an FCM message.
//
// If a message is not acked, FCM will continously resend the message until
// it expires. Duplicate delivery in this case is working as intended.
type Ack struct {
	// Id of message being acknowledged
	MessageId            string   `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Ack) Reset()         { *m = Ack{} }
func (m *Ack) String() string { return proto.CompactTextString(m) }
func (*Ack) ProtoMessage()    {}
func (*Ack) Descriptor() ([]byte, []int) {
	return fileDescriptor_connection_api_e2e68185fc0238ce, []int{2}
}
func (m *Ack) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Ack.Unmarshal(m, b)
}
func (m *Ack) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Ack.Marshal(b, m, deterministic)
}
func (dst *Ack) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Ack.Merge(dst, src)
}
func (m *Ack) XXX_Size() int {
	return xxx_messageInfo_Ack.Size(m)
}
func (m *Ack) XXX_DiscardUnknown() {
	xxx_messageInfo_Ack.DiscardUnknown(m)
}

var xxx_messageInfo_Ack proto.InternalMessageInfo

func (m *Ack) GetMessageId() string {
	if m != nil {
		return m.MessageId
	}
	return ""
}

// Message created through the [Send
// API](https://firebase.google.com/docs/reference/fcm/rest/v1/projects.messages#resource-message).
type Message struct {
	// The identifier of the message. Used to ack the message.
	MessageId string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	// Time the message was received in FCM.
	CreateTime *timestamp.Timestamp `protobuf:"bytes,2,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	// Expiry time of the message. Currently it is always 4 weeks.
	ExpireTime *timestamp.Timestamp `protobuf:"bytes,3,opt,name=expire_time,json=expireTime,proto3" json:"expire_time,omitempty"`
	// The arbitrary payload set in the [Send
	// API](https://firebase.google.com/docs/reference/fcm/rest/v1/projects.messages#resource-message).
	Data                 map[string]string `protobuf:"bytes,4,rep,name=data,proto3" json:"data,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_connection_api_e2e68185fc0238ce, []int{3}
}
func (m *Message) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Message.Unmarshal(m, b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Message.Marshal(b, m, deterministic)
}
func (dst *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(dst, src)
}
func (m *Message) XXX_Size() int {
	return xxx_messageInfo_Message.Size(m)
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetMessageId() string {
	if m != nil {
		return m.MessageId
	}
	return ""
}

func (m *Message) GetCreateTime() *timestamp.Timestamp {
	if m != nil {
		return m.CreateTime
	}
	return nil
}

func (m *Message) GetExpireTime() *timestamp.Timestamp {
	if m != nil {
		return m.ExpireTime
	}
	return nil
}

func (m *Message) GetData() map[string]string {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
	proto.RegisterType((*UpstreamRequest)(nil), "google.firebase.fcm.connection.v1alpha1.UpstreamRequest")
	proto.RegisterType((*DownstreamResponse)(nil), "google.firebase.fcm.connection.v1alpha1.DownstreamResponse")
	proto.RegisterType((*Ack)(nil), "google.firebase.fcm.connection.v1alpha1.Ack")
	proto.RegisterType((*Message)(nil), "google.firebase.fcm.connection.v1alpha1.Message")
	proto.RegisterMapType((map[string]string)(nil), "google.firebase.fcm.connection.v1alpha1.Message.DataEntry")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ConnectionApiClient is the client API for ConnectionApi service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ConnectionApiClient interface {
	// Creates a streaming connection with FCM to send messages and their
	// respective ACKs.
	//
	// The client credentials need to be passed in the [gRPC
	// Metadata](https://grpc.io/docs/guides/concepts.html#metadata). The Format
	// of the header is:
	//   Key: "authorization"
	//   Value: "Checkin [client_id:secret]"
	//
	//
	// The project's API key also needs to be sent to authorize the project.
	// That can be set in the X-Goog-Api-Key Metadata header.
	Connect(ctx context.Context, opts ...grpc.CallOption) (ConnectionApi_ConnectClient, error)
}

type connectionApiClient struct {
	cc *grpc.ClientConn
}

func NewConnectionApiClient(cc *grpc.ClientConn) ConnectionApiClient {
	return &connectionApiClient{cc}
}

func (c *connectionApiClient) Connect(ctx context.Context, opts ...grpc.CallOption) (ConnectionApi_ConnectClient, error) {
	stream, err := c.cc.NewStream(ctx, &_ConnectionApi_serviceDesc.Streams[0], "/google.firebase.fcm.connection.v1alpha1.ConnectionApi/Connect", opts...)
	if err != nil {
		return nil, err
	}
	x := &connectionApiConnectClient{stream}
	return x, nil
}

type ConnectionApi_ConnectClient interface {
	Send(*UpstreamRequest) error
	Recv() (*DownstreamResponse, error)
	grpc.ClientStream
}

type connectionApiConnectClient struct {
	grpc.ClientStream
}

func (x *connectionApiConnectClient) Send(m *UpstreamRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *connectionApiConnectClient) Recv() (*DownstreamResponse, error) {
	m := new(DownstreamResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ConnectionApiServer is the server API for ConnectionApi service.
type ConnectionApiServer interface {
	// Creates a streaming connection with FCM to send messages and their
	// respective ACKs.
	//
	// The client credentials need to be passed in the [gRPC
	// Metadata](https://grpc.io/docs/guides/concepts.html#metadata). The Format
	// of the header is:
	//   Key: "authorization"
	//   Value: "Checkin [client_id:secret]"
	//
	//
	// The project's API key also needs to be sent to authorize the project.
	// That can be set in the X-Goog-Api-Key Metadata header.
	Connect(ConnectionApi_ConnectServer) error
}

func RegisterConnectionApiServer(s *grpc.Server, srv ConnectionApiServer) {
	s.RegisterService(&_ConnectionApi_serviceDesc, srv)
}

func _ConnectionApi_Connect_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ConnectionApiServer).Connect(&connectionApiConnectServer{stream})
}

type ConnectionApi_ConnectServer interface {
	Send(*DownstreamResponse) error
	Recv() (*UpstreamRequest, error)
	grpc.ServerStream
}

type connectionApiConnectServer struct {
	grpc.ServerStream
}

func (x *connectionApiConnectServer) Send(m *DownstreamResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *connectionApiConnectServer) Recv() (*UpstreamRequest, error) {
	m := new(UpstreamRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _ConnectionApi_serviceDesc = grpc.ServiceDesc{
	ServiceName: "google.firebase.fcm.connection.v1alpha1.ConnectionApi",
	HandlerType: (*ConnectionApiServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Connect",
			Handler:       _ConnectionApi_Connect_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "google/firebase/fcm/connection/v1alpha1/connection_api.proto",
}

func init() {
	proto.RegisterFile("google/firebase/fcm/connection/v1alpha1/connection_api.proto", fileDescriptor_connection_api_e2e68185fc0238ce)
}

var fileDescriptor_connection_api_e2e68185fc0238ce = []byte{
	// 453 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x93, 0xc1, 0x6e, 0x13, 0x31,
	0x10, 0x86, 0xb3, 0xd9, 0x42, 0x94, 0x09, 0xa5, 0xc8, 0xe2, 0x10, 0xad, 0x40, 0x54, 0x11, 0x12,
	0x91, 0x40, 0xde, 0x36, 0x1c, 0xa8, 0x1a, 0x0e, 0x24, 0x14, 0xa9, 0x48, 0x80, 0x60, 0x05, 0x17,
	0x2e, 0xd1, 0xc4, 0x71, 0x16, 0x2b, 0x59, 0xdb, 0xd8, 0x4e, 0x21, 0x57, 0x0e, 0x3c, 0x03, 0xef,
	0xc0, 0x4b, 0xa2, 0x5d, 0x7b, 0x5b, 0x04, 0x87, 0x6c, 0x6f, 0xf1, 0xcc, 0xff, 0xfd, 0xff, 0x78,
	0xe2, 0x85, 0xe7, 0xb9, 0x52, 0xf9, 0x9a, 0xa7, 0x4b, 0x61, 0xf8, 0x1c, 0x2d, 0x4f, 0x97, 0xac,
	0x48, 0x99, 0x92, 0x92, 0x33, 0x27, 0x94, 0x4c, 0x2f, 0x8e, 0x71, 0xad, 0xbf, 0xe0, 0xf1, 0x5f,
	0xb5, 0x19, 0x6a, 0x41, 0xb5, 0x51, 0x4e, 0x91, 0x47, 0x9e, 0xa6, 0x35, 0x4d, 0x97, 0xac, 0xa0,
	0x57, 0x4a, 0x5a, 0xd3, 0xc9, 0xbd, 0x10, 0x83, 0x5a, 0xa4, 0x28, 0xa5, 0x72, 0x58, 0xf6, 0xad,
	0xb7, 0x49, 0x1e, 0x84, 0x6e, 0x75, 0x9a, 0x6f, 0x96, 0xa9, 0x13, 0x05, 0xb7, 0x0e, 0x0b, 0xed,
	0x05, 0x03, 0x06, 0x07, 0x9f, 0xb4, 0x75, 0x86, 0x63, 0x91, 0xf1, 0xaf, 0x1b, 0x6e, 0x1d, 0x79,
	0x01, 0x31, 0xb2, 0x55, 0x3f, 0x3a, 0x8c, 0x86, 0xbd, 0xd1, 0x13, 0xda, 0x70, 0x10, 0x3a, 0x61,
	0xab, 0xf3, 0x56, 0x56, 0xa2, 0xd3, 0xdb, 0x70, 0xcb, 0x78, 0xb3, 0x99, 0xdb, 0x6a, 0x3e, 0xb0,
	0x40, 0xce, 0xd4, 0x37, 0x59, 0xc7, 0x58, 0xad, 0xa4, 0xe5, 0xe4, 0x0d, 0x74, 0x0a, 0x6e, 0x2d,
	0xe6, 0x3c, 0x64, 0x1d, 0x35, 0xce, 0x7a, 0xeb, 0xb9, 0xf3, 0x56, 0x56, 0x5b, 0x4c, 0x0f, 0x60,
	0xdf, 0x04, 0x67, 0x1f, 0xfa, 0x10, 0xe2, 0x09, 0x5b, 0x91, 0xfb, 0x00, 0x41, 0x32, 0x13, 0x8b,
	0x2a, 0xa8, 0x9b, 0x75, 0x43, 0xe5, 0xf5, 0x62, 0xf0, 0xbb, 0x0d, 0x9d, 0xe0, 0xb6, 0x43, 0x4a,
	0xc6, 0xd0, 0x63, 0x86, 0xa3, 0xe3, 0xb3, 0x72, 0x89, 0xfd, 0x76, 0x35, 0x73, 0x52, 0xcf, 0x5c,
	0x6f, 0x98, 0x7e, 0xac, 0x37, 0x9c, 0x81, 0x97, 0x97, 0x85, 0x12, 0xe6, 0xdf, 0xb5, 0x30, 0x01,
	0x8e, 0x77, 0xc3, 0x5e, 0x5e, 0xc1, 0xef, 0x60, 0x6f, 0x81, 0x0e, 0xfb, 0x7b, 0x87, 0xf1, 0xb0,
	0x37, 0x3a, 0xbd, 0xee, 0x9a, 0xe8, 0x19, 0x3a, 0x7c, 0x25, 0x9d, 0xd9, 0x66, 0x95, 0x4f, 0xf2,
	0x0c, 0xba, 0x97, 0x25, 0x72, 0x07, 0xe2, 0x15, 0xdf, 0x86, 0xeb, 0x96, 0x3f, 0xc9, 0x5d, 0xb8,
	0x71, 0x81, 0xeb, 0x8d, 0xbf, 0x62, 0x37, 0xf3, 0x87, 0xd3, 0xf6, 0x49, 0x34, 0xfa, 0x15, 0xc1,
	0xfe, 0xcb, 0xcb, 0xa0, 0x89, 0x16, 0xe4, 0x67, 0x04, 0x9d, 0x50, 0x21, 0x27, 0x8d, 0x07, 0xfb,
	0xe7, 0xc9, 0x25, 0xe3, 0xc6, 0xe4, 0xff, 0xef, 0x68, 0xd0, 0x1a, 0x46, 0x47, 0xd1, 0xf4, 0x47,
	0x04, 0x8f, 0x99, 0x2a, 0x9a, 0x1a, 0xbd, 0x8f, 0x3e, 0x7f, 0x08, 0xd2, 0x5c, 0xad, 0x51, 0xe6,
	0x54, 0x99, 0x3c, 0xcd, 0xb9, 0xac, 0xfe, 0x8a, 0xd4, 0xb7, 0x50, 0x0b, 0xbb, 0xf3, 0xfb, 0x1d,
	0x5f, 0xd5, 0xe6, 0x37, 0x2b, 0xfa, 0xe9, 0x9f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xe4, 0x30, 0x40,
	0x1a, 0xfc, 0x03, 0x00, 0x00,
}
