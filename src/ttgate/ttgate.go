/*
 * Teletype Gateway
 *
 */

package main

import (
    "time"
	"fmt"
	"os"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
)

var debug bool = false

func main() {
	var s string

	fmt.Printf("Teletype Gateway\n")

	s = os.Getenv("HALT")		// Resin debugging via terminal requires quitting the main instance
	if (s != "") {
		fmt.Printf("HALT environment variable detected\n");
		fmt.Printf("Exiting.\n");
		os.Exit(0);
	}

	s = os.Getenv("DEBUG")		// For verbose debugging info
	debug = s != ""

    // Initialize serial I/O.  We can't very well proceed without
    // a serial port, and yet it's senseless to exit within
    // the resin environment.
	
    for !ioInit() {
        time.Sleep(5 * time.Second)
    }

	fmt.Printf("Serial I/O Initialized\n")

    // In our idle loop, transmit a beacon once per minute.
    // This is to simulate stuff coming in from the cloud service

    for {
		totalReceived, totalSent := cmdGetStats()
		message := fmt.Sprintf("#gateway received %lu sent %lu", totalReceived, totalSent)

		// Broadcast a test message

		deviceType := teletype.Telecast_TTGATE
		msg := &teletype.Telecast {}
		msg.DeviceType = &deviceType
		msg.DeviceID = proto.String(getDeviceID())
		msg.Message = proto.String(message)
		data, err := proto.Marshal(msg)
		if err != nil {
			fmt.Printf("marshaling error: ", err)
		}
        cmdEnqueueOutbound(data)

		// Sleep for a while
		
        time.Sleep(60 * time.Second)
    }

}

// eof
