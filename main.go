package main

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

type Telecast struct {
}

func (m *Telecast) Reset()                    { *m = Telecast{} }
func (m *Telecast) String() string            { return proto.CompactTextString(m) }
func (*Telecast) ProtoMessage()               {}
func (*Telecast) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func init() {
	proto.RegisterType((*Telecast)(nil), "Telecast")
}

var fileDescriptor0 = []byte{
}

func main() {}
func test(msg *Telecast) {}
