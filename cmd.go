// Command processing for interaction with the LPWAN chip
package main

import (
	"fmt"
)

// Outbound command queue structure
type OutboundCommand struct {
	Command []byte
}

// Statics
var outboundQueue chan OutboundCommand
var cmdInitialized bool = false
var inReinit bool = false
var totalMessagesReceived int = 0
var busyCount = 0
var watchdog1mCount = 0

// First time initialization of the command processing subsystem
func cmdInit() {

	// Initialize the outbound queue
	outboundQueue = make(chan OutboundCommand, 100) // Don't exhibit backpressure for a long time

	// Init state machine, etc.
	cmdReinit()

	// We're now fully initialized
	cmdInitialized = true

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
	cmdSetState(CMD_STATE_LPWAN_RESETREQ)
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
		fmt.Printf("*** cmdStateChangeWatchdog: Warning!\n")
	case 3:
		fmt.Printf("*** cmdStateChangeWatchdog: Reinitializing!\n")
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
func cmdGetStats() (received int) {
	return totalMessagesReceived
}
