// Code generated by protoc-gen-go. DO NOT EDIT.
// source: githubcard.proto

/*
Package githubcard is a generated protocol buffer package.

It is generated from these files:
	githubcard.proto

It has these top-level messages:
	Token
*/
package githubcard

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Token struct {
	Token string `protobuf:"bytes,1,opt,name=token" json:"token,omitempty"`
}

func (m *Token) Reset()                    { *m = Token{} }
func (m *Token) String() string            { return proto.CompactTextString(m) }
func (*Token) ProtoMessage()               {}
func (*Token) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Token) GetToken() string {
	if m != nil {
		return m.Token
	}
	return ""
}

func init() {
	proto.RegisterType((*Token)(nil), "githubcard.Token")
}

func init() { proto.RegisterFile("githubcard.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 75 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x48, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0x4a, 0x4e, 0x2c, 0x4a, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x42, 0x88,
	0x28, 0xc9, 0x72, 0xb1, 0x86, 0xe4, 0x67, 0xa7, 0xe6, 0x09, 0x89, 0x70, 0xb1, 0x96, 0x80, 0x18,
	0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x10, 0x4e, 0x12, 0x1b, 0x58, 0x87, 0x31, 0x20, 0x00,
	0x00, 0xff, 0xff, 0xc7, 0x5a, 0xfa, 0x37, 0x45, 0x00, 0x00, 0x00,
}