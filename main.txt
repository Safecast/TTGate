package main
import proto "github.com/golang/protobuf/proto"
type Telecast struct {}
func (m *Telecast) Reset()                    { *m = Telecast{} }
func (*Telecast) ProtoMessage()               {}
func (m *Telecast) String() string            { return proto.CompactTextString(m) }
func main() {}
