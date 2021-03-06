// Copyright 2017 Inca Roads LLC.  All rights reserved. 
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file. 

// Command processing for interaction with the LPWAN chip
package main

import (
	"fmt"
)

// Outbound command queue structure
type outboundCommand struct {
	Command []byte
}

// Statics
var outboundQueue chan outboundCommand
var cmdInitialized bool
var inReinit bool
var totalMessagesReceived uint32
var busyCount int
var watchdog1mCount int

// First time initialization of the command processing subsystem
func cmdInit() {

	// Initialize the outbound queue
	outboundQueue = make(chan outboundCommand, 100) // Don't exhibit backpressure for a long time

	// Init state machine, etc.
	cmdReinit()

	// We're now fully initialized
	cmdInitialized = true

}

// Enqueue an outbound message that already has a PB_ARRAY header
func cmdEnqueueOutboundPayload(cmd []byte) {
    var ocmd outboundCommand
    ocmd.Command = cmd
    outboundQueue <- ocmd
}

// Reinitialize the world upon failure conditions
func cmdReinit() {

	// Prevent recursion because we call this from multiple goroutines
	if inReinit {
		return
	}
	inReinit = true

	// Reinitialize the Microchip in case it's wedged.
	ioInitMicrochip()

	// Initialize the state machine and kick off a device reset
	cmdSetResetState()
	cmdProcess(nil)

	// Done
	inReinit = false

}

// Watchdog, in order to handle LPWAN chip resets
func cmd1mWatchdog() {

	// Exit if we're not yet initialized
	if !cmdInitialized || inReinit {
		return
	}

	// Ignore the first increments, but then reset the world
	watchdog1mCount = watchdog1mCount + 1
	switch watchdog1mCount {
	case 1:
	case 2:
		go fmt.Printf("*** cmdStateChangeWatchdog: Warning!\n")
	case 3:
		go fmt.Printf("*** cmdStateChangeWatchdog: Reinitializing!\n")
		cmdReinit()
	}

}

// Handle the case where the chip gets into a locked state
// in which it is permanently returning "busy" as a reply
func cmdBusy() {

	// Ignore the first increments, but then reset the world
	busyCount = busyCount + 1
	if busyCount > 10 {
		cmdReinit()
	}

}

// Reset the cmd watchdog
func cmdStateChangeWatchdogReset() {
	watchdog1mCount = 0
}

// Reset the "busy reply" watchdog
func cmdBusyReset() {
	busyCount = 0
}

// Get stats
func cmdGetStats() (received uint32) {
	return totalMessagesReceived
}
