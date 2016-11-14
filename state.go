// State management for processing of commands
package main

import (
    "bytes"
    "fmt"
    "strconv"
    "time"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
)

// Command processing states
const (
    CMD_STATE_IDLE = iota
    CMD_STATE_LPWAN_RESETREQ
    CMD_STATE_LPWAN_RESETRPL
    CMD_STATE_LPWAN_GETVERRPL
    CMD_STATE_LPWAN_MACPAUSERPL
    CMD_STATE_LPWAN_SETWDTRPL
    CMD_STATE_LPWAN_RCVRPL
    CMD_STATE_LPWAN_TXRPL1
    CMD_STATE_LPWAN_TXRPL2
    CMD_STATE_LPWAN_SNRRPL
)

// Constants
const invalidSNR float32 = 123.456

// Statics
var receivedMessage []byte
var currentState uint16

// Set the current state of the state machine
func cmdSetState(newState uint16) {
    currentState = newState
    cmdStateChangeWatchdogReset()
}

// Set into a Receive state, and await reply
func RestartReceive() {
    ioSendCommandString("radio rx 0")
    cmdBusyReset()
    cmdSetState(CMD_STATE_LPWAN_RCVRPL)
}

// Process an inbound message received from the LPWAN
func cmdProcess(cmd []byte) {
    cmdstr := string(cmd)

    // Handle initialization cases
    if cmd == nil {
        // Special case of the very first init, which is called from outside the inbound task
        cmd = []byte("")
    } else {
        // This is a special, necessary delay because we DO get called here from
        // the inbound task even in the middle of initialization, and we're simply
        // not prepared to deal with it yet.
        for !cmdInitialized || inReinit {
            time.Sleep(1 * time.Second)
        }
    }

    // State dispatcher
    fmt.Printf("recv(%s)\n", cmdstr)
    switch currentState {

		////
		// Initialization states
		////

    case CMD_STATE_LPWAN_RESETREQ:
        time.Sleep(4 * time.Second)
        ioSendCommandString("sys get ver")
        cmdSetState(CMD_STATE_LPWAN_GETVERRPL)

    case CMD_STATE_LPWAN_GETVERRPL:
        time.Sleep(4 * time.Second)
        if (!bytes.HasPrefix(cmd, []byte("RN2483"))) && (!bytes.HasPrefix(cmd, []byte("RN2903"))) {
            ioSendCommandString("sys get ver")
            cmdSetState(CMD_STATE_LPWAN_GETVERRPL)
        } else {
            ioSendCommandString("sys reset")
            cmdSetState(CMD_STATE_LPWAN_RESETRPL)
        }

    case CMD_STATE_LPWAN_RESETRPL:
        time.Sleep(4 * time.Second)
        ioSendCommandString("mac pause")
        cmdSetState(CMD_STATE_LPWAN_MACPAUSERPL)

    case CMD_STATE_LPWAN_MACPAUSERPL:
        time.Sleep(4 * time.Second)
        // If we're still getting these responses, it's because we're still
        // flushing the buffer of incoming sys get ver's or sys resets from
        // previous commands.  In this case, do NOT issue new commands
        // because we'll just aggravate the situation.  Just flush,
        // and keep waiting for the expected command.
        if (bytes.HasPrefix(cmd, []byte("RN2483"))) || (bytes.HasPrefix(cmd, []byte("RN2903"))) {
            cmdSetState(CMD_STATE_LPWAN_MACPAUSERPL)
        } else {
            i64, err := strconv.ParseInt(cmdstr, 10, 64)
            if err != nil || i64 < 100000 {
                fmt.Printf("Bad response from mac pause: %s\n", cmdstr)
            } else {
                ioSendCommandString("radio set wdt 60000")
                cmdSetState(CMD_STATE_LPWAN_SETWDTRPL)
            }
        }

    case CMD_STATE_LPWAN_SETWDTRPL:
        // Allow the LPWAN to settle after init
        time.Sleep(4 * time.Second)
        // The init sequence is over, so begin a receive
        RestartReceive()

		////
		// Steady-state receive handling states
		////
		
    case CMD_STATE_LPWAN_RCVRPL:
        if bytes.HasPrefix(cmd, []byte("ok")) {
            // this is expected response from initiating the rcv,
            // so just ignore it and keep waiting for a message to come in
        } else if bytes.HasPrefix(cmd, []byte("radio_err")) {
            // Expected from receive timeout of WDT seconds.
            // if there's a pending outbound, transmit it (which will change state)
            // else restart the receive
            if !SentPendingOutbound() {
                RestartReceive()
            }
        } else if bytes.HasPrefix(cmd, []byte("busy")) {
            // This is not at all expected, but it means that we're
            // moving too quickly and we should try again.
            time.Sleep(5 * time.Second)
            RestartReceive()
            // reset the world if too many consecutive busy errors
            cmdBusy()
        } else if bytes.HasPrefix(cmd, []byte("radio_rx")) {
            // skip whitespace, then remember the message that we received,
            // because we'll need it after we get the SNR of the transmission
            var hexstarts int
            for hexstarts = len("radio_rx"); hexstarts < len(cmd); hexstarts++ {
                if cmd[hexstarts] > ' ' {
                    break
                }
            }
            receivedMessage = cmd[hexstarts:]
            // Get the SNR of the last message received
            ioSendCommandString("radio get snr")
            cmdSetState(CMD_STATE_LPWAN_SNRRPL)
        } else {
            // Totally unknown error, but since we cannot just
            // leave things in a state without a pending receive,
            // we need to just restart the world.
            fmt.Printf("LPWAN rcv error\n")
            cmdReinit()
        }

    case CMD_STATE_LPWAN_SNRRPL:
        {
            // Get the number in the commanbd buffer
            snr64, err := strconv.ParseFloat(cmdstr, 64)
            if err != nil {
                snr64 = float64(invalidSNR)
            }
            // Parse and process the received message
            cmdProcessReceived(receivedMessage, float32(snr64))
            // If there's a pending outbound, transmit it (which will change state)
            // else restart the receive
            if !SentPendingOutbound() {
                RestartReceive()
            }
        }

		////
		// Post-cmdEnqueueOutbound transmit-handling states
		////

    case CMD_STATE_LPWAN_TXRPL1:
        if bytes.HasPrefix(cmd, []byte("ok")) {
            cmdSetState(CMD_STATE_LPWAN_TXRPL2)
        } else if bytes.HasPrefix(cmd, []byte("busy")) {
            // This is not at all expected, but it means that we're
            // moving too quickly and we should try again.
            time.Sleep(5 * time.Second)
            RestartReceive()
            // reset the world if too many consecutive busy errors
            cmdBusy()
        } else {
            fmt.Printf("LPWAN xmt1 error\n")
            RestartReceive()
        }

    case CMD_STATE_LPWAN_TXRPL2:
        if bytes.HasPrefix(cmd, []byte("radio_tx_ok")) {
            // if there's another pending outbound, transmit it, else restart the receive
            if !SentPendingOutbound() {
                RestartReceive()
            }
        } else {
            fmt.Printf("LPWAN xmt2 error\n")
            RestartReceive()
        }

    }

}

// Enqueue an outbound message (from any goroutine)
func cmdEnqueueOutbound(cmd []byte) {
    var ocmd OutboundCommand
    ocmd.Command = cmd
    outboundQueue <- ocmd
}

// Send the pending outbound (from command processing goroutine)
func SentPendingOutbound() bool {
    hexchar := []byte("0123456789ABCDEF")

	// If there is no actual pending outbound, check to see
	// if we are offline.  If so, we should notify anyone who
	// is trying to transmit.
	if (len(outboundQueue) == 0 && !isTeletypeServiceReachable()) {
	    msg := &teletype.Telecast{}
        msg.Message = proto.String("down")
        data, err := proto.Marshal(msg)
        if err == nil {
            cmdEnqueueOutbound(data)
		}
	}

    // We test this because we can never afford to block here,
    // and we knkow that we're the only consumer of this queue
    if len(outboundQueue) != 0 {

        for ocmd := range outboundQueue {
            outbuf := []byte("radio tx ")
            for _, databyte := range ocmd.Command {
                loChar := hexchar[(databyte & 0x0f)]
                hiChar := hexchar[((databyte >> 4) & 0x0f)]
                outbuf = append(outbuf, hiChar)
                outbuf = append(outbuf, loChar)
            }
            ioSendCommand(outbuf)
            cmdBusyReset()
            cmdSetState(CMD_STATE_LPWAN_TXRPL1)
			// Returning true indicates that we set state
            return true
        }

    }
	
	// Returning false indicates that state is unchanged
    return false
}

// Process a received message, as hexadecimal text
func cmdProcessReceived(hex []byte, snr float32) {

    // Convert received message from hex to binary
    bin := make([]byte, len(hex)/2)
    for i := 0; i < len(hex)/2; i++ {

        var hinibble, lonibble byte
        hinibblechar := hex[2*i]
        lonibblechar := hex[(2*i)+1]

        if hinibblechar >= '0' && hinibblechar <= '9' {
            hinibble = hinibblechar - '0'
        } else if hinibblechar >= 'A' && hinibblechar <= 'F' {
            hinibble = (hinibblechar - 'A') + 10
        } else if hinibblechar >= 'a' && hinibblechar <= 'f' {
            hinibble = (hinibblechar - 'a') + 10
        } else {
            hinibble = 0
        }

        if lonibblechar >= '0' && lonibblechar <= '9' {
            lonibble = lonibblechar - '0'
        } else if lonibblechar >= 'A' && lonibblechar <= 'F' {
            lonibble = (lonibblechar - 'A') + 10
        } else if lonibblechar >= 'a' && lonibblechar <= 'f' {
            lonibble = (lonibblechar - 'a') + 10
        } else {
            lonibble = 0
        }

        bin[i] = (hinibble << 4) | lonibble

    }

    // Unpack the received message which is a protocol buffer
    msg := &teletype.Telecast{}
    err := proto.Unmarshal(bin, msg)
    if err != nil {
        fmt.Printf("cmdProcessReceivedProtobuf unmarshaling error: ", err)
        return
    }

	// Process it as a Telecast message
    cmdProcessReceivedTelecastMessage(msg, bin, snr)

}
