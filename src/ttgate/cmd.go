/*
 * Teletype Command Processing
 *
 * This module contains the command state machine.
 */

package main

import (
    "fmt"
    "bytes"
	"strings"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
)

// States

const (
    CMD_STATE_IDLE = iota
    CMD_STATE_LPWAN_RESETREQ
    CMD_STATE_LPWAN_GETVERRPL
    CMD_STATE_LPWAN_MACPAUSERPL
    CMD_STATE_LPWAN_SETWDTRPL
    CMD_STATE_LPWAN_RCVRPL
    CMD_STATE_LPWAN_TXRPL1
    CMD_STATE_LPWAN_TXRPL2
)

type OutboundCommand struct {
    Command []byte
}

var outboundQueue chan OutboundCommand
var currentState uint16

func cmdInit() {

    // Initialize the state machine and kick off a device reset

    cmdSetState(CMD_STATE_LPWAN_RESETREQ);
    cmdProcess([]byte(""))

    // Initialize the outbound queue

    outboundQueue = make(chan OutboundCommand, 100)         // Don't exhibit backpressure for a long time

}

func cmdEnqueueOutbound(cmd []byte) {
    var ocmd OutboundCommand
    ocmd.Command = cmd
    outboundQueue <- ocmd
}

func cmdSetState(newState uint16) {
    currentState = newState
}

func cmdProcess(cmd []byte) {

    fmt.Printf("cmdProcess(%s)\n", cmd)

    switch currentState {

    case CMD_STATE_LPWAN_RESETREQ:
        // This is important, because it is a harmless command
        // that we can use to get in sync with an unaligned
        // command stream.  This may fail, but that is the point.
        ioSendCommandString("sys get ver")
        cmdSetState(CMD_STATE_LPWAN_GETVERRPL)

    case CMD_STATE_LPWAN_GETVERRPL:
        ioSendCommandString("mac pause")
        cmdSetState(CMD_STATE_LPWAN_MACPAUSERPL)

    case CMD_STATE_LPWAN_MACPAUSERPL:
        ioSendCommandString("radio set wdt 10000")
        cmdSetState(CMD_STATE_LPWAN_SETWDTRPL)

    case CMD_STATE_LPWAN_SETWDTRPL:
        ioSendCommandString("radio rx 0")
        cmdSetState(CMD_STATE_LPWAN_RCVRPL)

    case CMD_STATE_LPWAN_RCVRPL:
        if bytes.HasPrefix(cmd, []byte("ok")) {
            // this is expected response from initiating the rcv,
            // so just ignore it and keep waiting for a message to come in
        } else if bytes.HasPrefix(cmd, []byte("radio_err")) {
            // Expected from receive timeout of WDT seconds.
            // if there's a pending outbound, transmit it (which will change state)
            // else restart the receive
            if (!SentPendingOutbound()) {
                RestartReceive()
            }
        } else if bytes.HasPrefix(cmd, []byte("busy")) {
            // This is not at all expected, but it means that we're
            // moving too quickly and we should try again.
            RestartReceive()
        } else if bytes.HasPrefix(cmd, []byte("radio_rx")) {
            // skip whitespace (there is more than one space)
            var hexstarts int
            for hexstarts = len("radio_rx"); hexstarts<len(cmd); hexstarts++ {
                if (cmd[hexstarts] > ' ') {
                    break
                }
            }
            // Parse and process the received message
            cmdProcessReceived(cmd[hexstarts:])
            // if there's a pending outbound, transmit it (which will change state)
            // else restart the receive
            if (!SentPendingOutbound()) {
                RestartReceive()
            }
        } else {
            // Totally unknown error, but since we cannot just
            // leave things in a state without a pending receive,
            // we need to just restart it.
            fmt.Printf("LPWAN rcv error\n")
            RestartReceive()
        }

    case CMD_STATE_LPWAN_TXRPL1:
        if bytes.HasPrefix(cmd, []byte("ok")) {
            cmdSetState(CMD_STATE_LPWAN_TXRPL2);
        } else {
            fmt.Printf("LPWAN xmt1 error\n")
            RestartReceive()
        }

    case CMD_STATE_LPWAN_TXRPL2:
        if bytes.HasPrefix(cmd, []byte("radio_tx_ok")) {
            // if there's another pending outbound, transmit it, else restart the receive
            if (!SentPendingOutbound()) {
                RestartReceive()
            }
        } else {
            fmt.Printf("LPWAN xmt2 error\n")
            RestartReceive()
        }

    }

}

func RestartReceive() {
    ioSendCommandString("radio rx 0")
    cmdSetState(CMD_STATE_LPWAN_RCVRPL)
}

func SentPendingOutbound() bool {
    hexchar := []byte("0123456789ABCDEF")

    // We test this because we can never afford to block here,
    // and we knkow that we're the only consumer of this queue

    if (len(outboundQueue) != 0) {

        for ocmd := range outboundQueue {
            outbuf := []byte("radio tx ")
            for _, databyte := range ocmd.Command {
                loChar := hexchar[(databyte & 0x0f)]
                hiChar := hexchar[((databyte >> 4) & 0x0f)]
                outbuf = append(outbuf, hiChar)
                outbuf = append(outbuf, loChar)
            }
            ioSendCommand(outbuf)
            cmdSetState(CMD_STATE_LPWAN_TXRPL1)
            return true
        }

    }
    return false
}

func cmdProcessReceived(hex []byte) {

    // Convert received message from hex to binary
    bin := make([]byte, len(hex)/2)
    for i := 0; i < len(hex)/2; i++ {

        var hinibble, lonibble byte
        hinibblechar := hex[2*i]
        lonibblechar := hex[(2*i)+1]

        if (hinibblechar >= '0' && hinibblechar <= '9') {
            hinibble = hinibblechar - '0'
        } else if (hinibblechar >= 'A' && hinibblechar <= 'F') {
            hinibble = (hinibblechar - 'A') + 10
        } else if (hinibblechar >= 'a' && hinibblechar <= 'f') {
            hinibble = (hinibblechar - 'a') + 10
        } else {
            hinibble = 0
        }

        if (lonibblechar >= '0' && lonibblechar <= '9') {
            lonibble = lonibblechar - '0'
        } else if (lonibblechar >= 'A' && lonibblechar <= 'F') {
            lonibble = (lonibblechar - 'A') + 10
        } else if (lonibblechar >= 'a' && lonibblechar <= 'f') {
            lonibble = (lonibblechar - 'a') + 10
        } else {
            lonibble = 0
        }

        bin[i] = (hinibble << 4) | lonibble

    }

    // Process the received protocol buffer

    cmdProcessReceivedProtobuf(bin)

}

func cmdProcessReceivedProtobuf(buf []byte) {

    // Unmarshal the buffer into a golang object

    msg := &teletype.Telecast{}
    err := proto.Unmarshal(buf, msg)
    if err != nil {
        fmt.Printf("cmdProcessReceivedProtobuf unmarshaling error: ", err);
        return
    }

	// Do special handling based on whether the message contains special hashtags

	str := msg.GetMessage() + " "		// So that the test works with hashtags at the end of the string

	if strings.Contains(str, "#safecast ") {
		cmdProcessReceivedSafecastMessage(msg)
	} else {
	    fmt.Printf("Received Msg from Device %s: '%s'\n", msg.GetDeviceID(), msg.GetMessage())
	}

}

func cmdProcessReceivedSafecastMessage(msg *teletype.Telecast) {

	fmt.Printf("Received Safecast Message:\n")
	fmt.Printf("    Message: %s\n", msg.GetMessage())
	fmt.Printf("    DeviceID: %s\n", msg.GetDeviceID())
	fmt.Printf("    DeviceType: %s\n", msg.GetDeviceType())
	if (msg.CapturedAt != nil) {
		fmt.Printf("    CapturedAt: %s\n", msg.GetCapturedAt())
	}
	if (msg.Unit != nil) {
		fmt.Printf("    Unit: %s\n", msg.GetUnit())
	}
	if (msg.Value != nil) {
		fmt.Printf("    Value: %d\n", msg.GetValue())
	}
	if (msg.Latitude != nil) {
		fmt.Printf("    Latitude: %f\n", msg.GetLatitude())
	}
	if (msg.Longitude != nil) {
		fmt.Printf("    Longitude: %f\n", msg.GetLongitude())
	}
	if (msg.Altitude != nil) {
		fmt.Printf("    Longitude: %d\n", msg.GetAltitude())
	}

}

// eof
