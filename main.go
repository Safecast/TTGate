// Teletype Gateway
package main

import (
    "fmt"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
)


func main() {
}

func cmdProcessReceivedTelecastMessage(msg *teletype.Telecast, pb []byte, snr float32) {

    switch msg.GetDeviceType() {

    case teletype.Telecast_TTGATE:
        if msg.Message == nil {
            msg.Message = proto.String("ping")
            data, err := proto.Marshal(msg)
            if err != nil {
                fmt.Printf("marshaling error: ", err, data)
            }
        }

    }


}
