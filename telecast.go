// Copyright 2017 Inca Roads LLC.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

// Telecast message handling
package main

import (
    "bytes"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net"
    "net/http"
    "os"
    "strconv"
    "time"
    "github.com/golang/protobuf/proto"
    "github.com/safecast/ttproto/golang"
)

// Service
var TTUploadAddress string = "tt.safecast.org"
var TTUploadURLPattern string = "http://%s/send"
var TTUploadIP string
var TTStatsURL string = "http://tt.safecast.org/gateway"

// Statics
var ipInfoString string = ""
var ipInfoData IPInfoData
var serviceReachable = true
var serviceFirstUnreachableAt time.Time
var FetchedIPInfo bool = false
var FetchedLatLon bool = false
var Latitude = ""
var Longitude = ""
var Altitude = ""

// Load localization information
func UpdateTargetIP() {

    // Look up the two IP addresses that we KNOW have only a single A record,
    // and determine if WE are the server for those protocols
    addrs, err := net.LookupHost(TTUploadAddress)
    if err != nil {
        go fmt.Printf("Can't resolve %s: %v\n", TTUploadAddress, err);
        TTUploadIP = TTUploadAddress
        return
    }
    if len(addrs) < 1 {
        go fmt.Printf("Can't resolve %s: %v\n", TTUploadAddress, err);
        TTUploadIP = TTUploadAddress
        return
    }
    TTUploadIP = addrs[0]

}

// Process a received Telecast message, forwarding if appropriate
func cmdProcessReceivedTelecastMessage(msg ttproto.Telecast, pb []byte, snr float32,  replyAllowed bool) {

    // Do various things baed upon the message type
    if msg.DeviceType == nil {

		// Solarcast
        cmdForwardMessageToTeletypeService(pb, snr, replyAllowed)
        go cmdLocallyDisplaySafecastMessage(msg, snr)

    } else {

        switch msg.GetDeviceType() {

            // Is this a simplecast message?
        case ttproto.Telecast_UNKNOWN_DEVICE_TYPE:
			fallthrough
        case ttproto.Telecast_SOLARCAST:
            cmdForwardMessageToTeletypeService(pb, snr, replyAllowed)
            go cmdLocallyDisplaySafecastMessage(msg, snr)

            // Are we simply forwarding a message originating from a nano?
        case ttproto.Telecast_BGEIGIE_NANO:
            cmdForwardMessageToTeletypeService(pb, snr, replyAllowed)
            go cmdLocallyDisplaySafecastMessage(msg, snr)

            // If this is a ping request (indicated by null Message), then send that device back the same thing we received,
            // but WITH a message (so that we don't cause a ping storm among multiple ttgates with visibility to each other)
        case ttproto.Telecast_TTGATE:
            // If we're offline, short circuit this because we don't want to mislead.
            // We'd rather that they use cellular.
            if !isTeletypeServiceReachable() {
                return
            }
            // Process it
            if msg.Message == nil {
                msg.Message = proto.String("ping")
                data, err := proto.Marshal(&msg)
                if err != nil {
                    go fmt.Printf("marshaling error: ", err)
                }
                // Importantly, sleep for a couple seconds to give the (slow) receiver a chance to get into receive mode.
				// We randomize it in case there are several ttgate's alive within listening range, so we minimize the chance
				// that we will step on each others' transmissions.
                time.Sleep(time.Duration(random(1,3)) * time.Second)
                cmdEnqueueOutboundPb(data)
                go fmt.Printf("Sent pingback to device %d\n", msg.GetDeviceId())
                return
            }

            // Forward the message to the service
            cmdForwardMessageToTeletypeService(pb, snr, replyAllowed)

            // If it's a non-Safecast device, just display what we received
        default:
            if msg.DeviceId != nil {
                go fmt.Printf("Received Msg from Device %d: '%s'\n", msg.GetDeviceId(), msg.GetMessage())
            }

        }
    }
}

// Refresh ipinfo as a string
func GetIPInfo() (bool, string, IPInfoData) {

    // If already avail, return it
    if ipInfoString != "" {
        return true, ipInfoString, ipInfoData
    }

    // The first time through here, let's fetch info about our IP.
    // We embrace the ip-api.com data definitions as our native format.
    if !FetchedIPInfo {
        FetchedIPInfo = true

        response, err := http.Get("http://ip-api.com/json/")
        if err == nil {
            defer response.Body.Close()
            contents, err := ioutil.ReadAll(response.Body)
            if err == nil {
                ipInfoString = string(contents)
                err = json.Unmarshal(contents, &ipInfoData)
                if err != nil {
                    ipInfoData = IPInfoData{}
                }
                return true, ipInfoString, ipInfoData
            }
        }

        go fmt.Printf("IPInfo failure: %s\n", err)

    }

    // Failure
    return false, "", IPInfoData{}

}

// Forward this message to the teletype service via HTTP
func cmdForwardMessageToTeletypeService(pb []byte, snr float32, replyAllowed bool) {

	// Note that if a reply is allowed, we MUST do this synchronously, because failing
	// to do so will cause the state.go state machine to immediately go into a recv()
	// which will prevent our send() from occurring within the waiting device's allowed
	// time window.
	if replyAllowed {
		forwardMessageToTeletypeService(pb, snr)
	} else {
		go forwardMessageToTeletypeService(pb, snr)
	}

}
	
// Forward this message to the teletype service via HTTP
func forwardMessageToTeletypeService(pb []byte, snr float32) {

    _, ipinfo, _ := GetIPInfo()

    // Pack the data into the same data structure as TTN, because we're simulating TTN inbound
    msg := &TTGateReq{}
    msg.ReceivedAt = nowInUTC()
    msg.Payload = pb

    // Pass along the gateway EUI
    msg.GatewayId, _ = cmdGetGatewayInfo()

    // Some devices don't have LAT/LON, and in this case the gateway will supply it (if configured)
    if !FetchedLatLon {
        FetchedLatLon = true
        Latitude = os.Getenv("LAT")
        Longitude = os.Getenv("LON")
        Altitude = os.Getenv("ALT")
    }

    if Latitude != "" {
        f64, err := strconv.ParseFloat(Latitude, 64)
        if err == nil {
            msg.Latitude = float32(f64)
        }
    }
    if Longitude != "" {
        f64, err := strconv.ParseFloat(Longitude, 64)
        if err == nil {
            msg.Longitude = float32(f64)
        }
    }
    if Altitude != "" {
        i64, err := strconv.ParseInt(Altitude, 10, 64)
        if err == nil {
            msg.Altitude = int32(i64)
        }
    }

    // The service might find it handy to see the SNR of the last message received from the gateway
    if snr != invalidSNR {
        msg.Snr = float32(snr)
    }

    // Augment the outbound metadata with ip info
    msg.Location = ipinfo

    // Send it to the teletype service via HTTP
    msgJSON, _ := json.Marshal(msg)
    UploadURL := fmt.Sprintf(TTUploadURLPattern, TTUploadIP)
    req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(msgJSON))
    req.Header.Set("User-Agent", "TTGATE")
    req.Header.Set("Content-Type", "application/json")
    httpclient := &http.Client{
        Timeout: time.Second * 15,
    }
    transaction_start := time.Now()
    resp, err := httpclient.Do(req)
    if err != nil {
        setTeletypeServiceReachability(false)
        go fmt.Printf("*** Error uploading to %s %s\n\n", UploadURL, err)
    } else {
        transaction_seconds := int64(time.Now().Sub(transaction_start) / time.Second)
        go fmt.Printf("Upload to %s took %ds\n", UploadURL, transaction_seconds)
        setTeletypeServiceReachability(true)
        contents, err := ioutil.ReadAll(resp.Body)
        if err == nil {
            payloadstr := string(contents)
            if payloadstr != "" {
                payload, err := hex.DecodeString(payloadstr)
                if err == nil {
                    cmdEnqueueOutboundPayload(payload)
                    go fmt.Printf("Sent reply: %s\n", payloadstr)
                } else {
                    go fmt.Printf("Error %v: %s\n", err, payloadstr)
                }
            }
        }
        resp.Body.Close()
    }

    // For testing purposes only, Also send the message via UDP
    testUDP := false
    if testUDP {

        ServerAddr, err := net.ResolveUDPAddr("udp", "tt.safecast.org:8081")
        if err != nil {
            go fmt.Printf("*** Error resolving UDP address: %v\n", err)
        } else {

            Conn, err := net.DialUDP("udp", nil, ServerAddr)
            if err != nil {
                go fmt.Printf("*** Error dialing UDP: %v\n", err)
            } else {

                _, err := Conn.Write(pb)
                if err != nil {
                    go fmt.Printf("*** Error writing UDP: %v\n", err)
                }

                Conn.Close()

            }
        }

    }

}

// Set the teletype service as known-reachable or known-unreachable
func setTeletypeServiceReachability(isReachable bool) {
    if (!serviceReachable && isReachable) {
        go fmt.Printf("*** TTSERVE is now reachable\n");
    } else if (serviceReachable && !isReachable) {
        go fmt.Printf("*** TTSERVE is now unreachable\n");
        serviceFirstUnreachableAt = time.Now()
    } else if (!serviceReachable && !isReachable) {
        t := time.Now()
        unreachableForMinutes := int64(t.Sub(serviceFirstUnreachableAt) / time.Minute)
        go fmt.Printf("*** TTSERVE has been unreachable for %d minutes\n", unreachableForMinutes);
    }
    serviceReachable = isReachable
}

// Set the teletype service as known-reachable or known-unreachable, with debouncing so that
// it ONLY says that it's unreachable if it has been down for a very long time.
// We use a significant amount of debounce time because this will cause devices to
// resort to using Cellular until their next reboot cycle.
func isTeletypeServiceReachable() bool {
    // Useful (saves an hour) when debugging ttrelay behavior upon receiving "down" message
    if DebugFailover {
        return false
    }
    // Exit immediately if the service is known to be reachable
    if serviceReachable {
        return true
    }
    // Suppress the notion of "unreachable" until we have been offline for quite some time
    unreachableMinutes := int64(time.Now().Sub(serviceFirstUnreachableAt) / time.Minute)
    return unreachableMinutes < 60
}

// Determine whether or not the service has been unreachable for a VERY long time,
// in which case we should assume that the device is in a really bad state.  If this
// is the case, we reboot.
func isOfflineForExtendedPeriod() bool {
    // Exit immediately if the service is known to be reachable
    if serviceReachable {
        return false
    }
    // Suppress the notion of "unreachable" until we have been offline for quite some time
    unreachableMinutes := int64(time.Now().Sub(serviceFirstUnreachableAt) / time.Minute)
    return unreachableMinutes > (60 * 6)
}

// Send stats to the service
func cmdSendStatsToTeletypeService() {

    // Construct an outbound message
    msg := &TTGateReq{}
    msg.ReceivedAt = nowInUTC()

    // Gateway name
    msg.GatewayId, msg.GatewayRegion = cmdGetGatewayInfo()
    msg.GatewayName = os.Getenv("RESIN_DEVICE_NAME_AT_INIT")

    // If we're executing prior to the fetching of the
    // gateway ID from the Lora chip, exit
    if msg.GatewayId == "" {
        return
    }

    // IPInfo
    _, _, msg.IPInfo = GetIPInfo()
	
    // Stats
    msg.MessagesReceived = cmdGetStats()
    msg.DevicesSeen = GetSafecastDevicesString()

	// Debug
	go fmt.Printf("Sending stats update to service\n")

    // Send it
    msgJSON, _ := json.Marshal(msg)
    req, err := http.NewRequest("POST", TTStatsURL, bytes.NewBuffer(msgJSON))
    req.Header.Set("User-Agent", "TTGATE")
    req.Header.Set("Content-Type", "application/json")
    httpclient := &http.Client{
        Timeout: time.Second * 15,
    }
    resp, err := httpclient.Do(req)
    if err != nil {
		fmt.Printf("*** Cannot reach service: %s\n", err)
        setTeletypeServiceReachability(false)
    } else {
        setTeletypeServiceReachability(true)
        resp.Body.Close()
    }

}

// Get the current time in UTC as a string
func nowInUTC() string {
    return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}
