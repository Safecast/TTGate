// Telecast message handling
package main

import (
    "bytes"
    "fmt"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
    "net/http"
    "time"
)

// Statics
var ipinfo string = ""
var serviceReachable = true
var serviceFirstUnreachableAt time.Time

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

    // Forward the message to the service [and delete the stuff from processreceivedsafecastmessage!]
    cmdForwardMessageToTeletypeService(pb, snr)

}

// Forward this message to the teletype service via HTTP
func cmdForwardMessageToTeletypeService(pb []byte, snr float32) {

}

// Ping the teletype service via HTTP, just to determine its reachability
func cmdPingTeletypeService() {

    UploadURL := "http://api.teletype.io:8080/send"
	data := []byte("Hello.")
    req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(data))
    req.Header.Set("User-Agent", "TTGATE")
    req.Header.Set("Content-Type", "application/json")
    httpclient := &http.Client{}
    resp, err := httpclient.Do(req)
    if err != nil {
		setTeletypeServiceReachability(false)
    } else {
		setTeletypeServiceReachability(true)
		resp.Body.Close()
    }

}

// Set the teletype service as known-reachable or known-unreachable
func setTeletypeServiceReachability(isReachable bool) {
	if (!serviceReachable && isReachable) {
	    fmt.Printf("*** TTSERVE is now reachable\n");
	} else if (serviceReachable && !isReachable) {
	    fmt.Printf("*** TTSERVE is now unreachable\n");
	    serviceFirstUnreachableAt = time.Now()
	} else if (!serviceReachable && !isReachable) {
	    t := time.Now()
		unreachableForMinutes := int64(t.Sub(serviceFirstUnreachableAt) / time.Minute)
	    fmt.Printf("*** TTSERVE has been unreachable for %d minutes\n", unreachableForMinutes);
	}
	serviceReachable = isReachable
}

// Set the teletype service as known-reachable or known-unreachable, with debouncing so that
// it ONLY says that it's unreachable if it has been down for a very long time.
// We use a significant amount of debounce time because this will cause devices to
// resort to using Cellular until their next reboot cycle.
func isTeletypeServiceReachable() bool {
	// Exit immediately if the service is known to be reachable
	if serviceReachable {
		return true
	}
	// Return unreachable immediately when testing
	testing := true
	if testing {
		return false
	}
	// Suppress the notion of "unreachable" until we have been offline for quite some time
	unreachableMinutes := int64(time.Now().Sub(serviceFirstUnreachableAt) / time.Minute)
	return unreachableMinutes < 60
}
