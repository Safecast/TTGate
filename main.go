// Teletype Gateway
package main

import (
    "fmt"
    "net/http"
    "bytes"
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

func cmdPingTeletypeService() {
    UploadURL := "http://api.teletype.io:8080/send"
	data := []byte("Hello.")
    req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(data))
    req.Header.Set("User-Agent", "TTGATE")
    req.Header.Set("Content-Type", "application/json")
    httpclient := &http.Client{}
    resp, err := httpclient.Do(req)
    if err == nil {
		resp.Body.Close()
    }

}

