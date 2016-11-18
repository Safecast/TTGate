// Telecast message handling
package main

import (
    "bytes"
	"encoding/hex"
    "encoding/json"
    "fmt"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
    "io/ioutil"
    "net"
    "net/http"
    "os"
    "strconv"
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
                fmt.Printf("marshaling error: ", err)
            }
            // Importantly, sleep for a couple seconds to give the (slow) receiver a chance to get into receive mode
            time.Sleep(2 * time.Second)
            cmdEnqueueOutbound(data)
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

    // The first time through here, let's fetch info about our IP.
    // We embrace the ip-api.com data definitions as our native format.
    if ipinfo == "" {
        response, err := http.Get("http://ip-api.com/json/")
        if err == nil {
            defer response.Body.Close()
            contents, err := ioutil.ReadAll(response.Body)
            if err == nil {
                ipinfo = string(contents)
            }
        }
    }

    // Pack the data into the same data structure as TTN, because we're simulating TTN inbound
    msg := &DataUpAppReq{}
    msg.Metadata = make([]AppMetadata, 1)
    msg.Payload = pb

    // Some devices don't have LAT/LON, and in this case the gateway will supply it (if configured)
    Latitude := os.Getenv("LAT")
    if Latitude != "" {
        f64, err := strconv.ParseFloat(Latitude, 64)
        if err == nil {
            msg.Metadata[0].Latitude = float32(f64)
        }
    }
    Longitude := os.Getenv("LON")
    if Longitude != "" {
        f64, err := strconv.ParseFloat(Longitude, 64)
        if err == nil {
            msg.Metadata[0].Longitude = float32(f64)
        }
    }
    Altitude := os.Getenv("ALT")
    if Altitude != "" {
        i64, err := strconv.ParseInt(Altitude, 10, 64)
        if err == nil {
            msg.Metadata[0].Altitude = int32(i64)
        }
    }

    // The service might find it handy to see the SNR of the last message received from the gateway
    if snr != invalidSNR {
        msg.Metadata[0].Lsnr = float32(snr)
    }

    // Augment the outbound metadata with ip info, overloading the
    // GatewayEUI data structure for this purpose
    msg.Metadata[0].GatewayEUI = ipinfo

    // Send it to the teletype service via HTTP
    UploadURL := "http://api.teletype.io:8080/send"
    msgJSON, _ := json.Marshal(msg)
    req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(msgJSON))
    req.Header.Set("User-Agent", "TTGATE")
    req.Header.Set("Content-Type", "application/json")
    httpclient := &http.Client{}
    resp, err := httpclient.Do(req)
    if err != nil {
		setTeletypeServiceReachability(false)
        fmt.Printf("*** Error uploading to TTSERVE %s\n\n", err)
    } else {
		setTeletypeServiceReachability(true)
		contents, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			payloadstr := string(contents)
			if payloadstr != "" {
				payload, err := hex.DecodeString(payloadstr)
				if err == nil {
					cmdEnqueueOutbound(payload)
					fmt.Printf("Sent reply: %s\n", payloadstr)
				} else {
					fmt.Printf("Error %v: %s\n", err, payloadstr)
				}
			}
		}
		resp.Body.Close()
    }

    // When testing, also send a duplicate of the message via UDP
    testUDP := false
    if testUDP {

        ServerAddr, err := net.ResolveUDPAddr("udp", "api.teletype.io:8081")
        if err != nil {
            fmt.Printf("*** Error resolving UDP address: %v\n", err)
        } else {

            Conn, err := net.DialUDP("udp", nil, ServerAddr)
            if err != nil {
                fmt.Printf("*** Error dialing UDP: %v\n", err)
            } else {

                _, err := Conn.Write(pb)
                if err != nil {
                    fmt.Printf("*** Error writing UDP: %v\n", err)
                }

                Conn.Close()

            }
        }

    }

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
