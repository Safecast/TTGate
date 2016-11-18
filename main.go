// Teletype Gateway
package main

import (
    "fmt"
    "net/http"
    "os"
    "time"
    "bytes"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
)


// Main entry point when launched by run.sh
func main() {

    // Spawn our localhost web server, used to update the HDMI status display
    go webServer()

	// Wait for quite a while, and then exit, which will cause our
	// shell script to restart the container.  This is a failsafe 
	// to ensure that any Linux-level process usage (such as bugs in
	// the golang runtime or Midori) will be reset, and we will
	// occasionally start completely fresh and clean.
    time.Sleep(7 * 24 * time.Hour)
    os.Exit(0)
	
}

// The localhost server used exclusively to update the local HDMI display
func webServer() {
    http.Handle("/", http.FileServer(http.Dir("./web")))
    http.ListenAndServe(":8080", nil)
}

// Process a received Telecast message, forwarding if appropriate
func cmdProcessReceivedTelecastMessage(msg *teletype.Telecast, pb []byte, snr float32) {

    // Do various things baed upon the message type
    switch msg.GetDeviceType() {

        // If this is a ping request (indicated by null Message), then send that device back the same thing we received,
        // but WITH a message (so that we don't cause a ping storm among multiple ttgates with visibility to each other)
    case teletype.Telecast_TTGATE:
        if msg.Message == nil {
            msg.Message = proto.String("ping")
            data, err := proto.Marshal(msg)
            if err != nil {
                fmt.Printf("marshaling error: ", err, data)
            }
            // Importantly, sleep for a couple seconds to give the (slow) receiver a chance to get into receive mode
            time.Sleep(2 * time.Second)
            fmt.Printf("Sent pingback to device %d\n", msg.GetDeviceIDNumber())
            return
        }

        // If it's a non-Safecast device, display what we received
    default:
        if msg.DeviceIDString != nil {
            fmt.Printf("Received Msg from Device %s: '%s'\n", msg.GetDeviceIDString(), msg.GetMessage())
        } else if msg.DeviceIDNumber != nil {
            fmt.Printf("Received Msg from Device %d: '%s'\n", msg.GetDeviceIDNumber(), msg.GetMessage())
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

