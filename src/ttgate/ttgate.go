/*
 * Teletype Gateway
 *
 */

package main

import (
    "time"
)

func main() {

    // Initialize serial I/O.  We can't very well proceed without
    // a serial port, and yet it's senseless to exit within
    // the resin environment.

	fmt.Printf("Teletype Gateway\n")
	
    for !ioInit() {
        time.Sleep(5 * time.Second)
    }

	fmt.Printf("Serial I/O Initialized\n")

    // In our idle loop, transmit a beacon once per minute.
    // This is to simulate stuff coming in from the cloud service

    for {

        time.Sleep(60 * time.Second)

        cmdEnqueueOutbound([]byte("Heartbeat"))
    }

}

// eof
