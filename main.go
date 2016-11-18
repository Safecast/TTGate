package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
)

func main() {
	fmt.Println(proto.GetStats())
}
