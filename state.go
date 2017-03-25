// Copyright 2017 Inca Roads LLC.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

// State management for processing of Lora commands
package main

import (
    "os"
    "bytes"
    "fmt"
    "strconv"
    "time"
    "strings"
    "github.com/golang/protobuf/proto"
    "github.com/safecast/ttproto/golang"
)

// Buffered I/O header formats coordinated with TTNODE.  Note that although we are now starting
// with version number 0, we special-case version number 8 because of the old style "single protocl buffer"
// message format that always begins with 0x08. (see ttnode/send.c)
const BUFF_FORMAT_PB_ARRAY byte  =  0
const BUFF_FORMAT_SINGLE_PB byte =  8

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
    CMD_STATE_LPWAN_SENDFQRPL
    CMD_STATE_LPWAN_GETEUIRPL
)

// Constants
const invalidSNR float32 = 123.456

// Statics
var receivedMessage []byte
var currentState uint16
var deviceToNotifyIfServiceDown uint32 = 0
var hweui string = ""

// Localization
var Region string = ""
var lorafpRegionCommandNumber int

// Get the unique gateway device ID
func cmdGetGatewayID() string {
    return hweui
}

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
        Region = os.Getenv("REGION")
        lorafpRegionCommandNumber = 0
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
                ioSendCommandString("sys get hweui")
                cmdSetState(CMD_STATE_LPWAN_GETEUIRPL)
            }
        }


    case CMD_STATE_LPWAN_GETEUIRPL:
        hweui = cmdstr
        ioSendCommandString("radio set wdt 60000")
        cmdSetState(CMD_STATE_LPWAN_SETWDTRPL)

    case CMD_STATE_LPWAN_SETWDTRPL:
        time.Sleep(100 * time.Millisecond)
        isCommand, theCommand := lorafp_get_command(lorafpRegionCommandNumber)
        if (isCommand) {
            lorafpRegionCommandNumber++;
            ioSendCommandString(theCommand)
            cmdSetState(CMD_STATE_LPWAN_SETWDTRPL)
            break;
        }
        fallthrough
    case CMD_STATE_LPWAN_SENDFQRPL:
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

// Enqueue an outbound ttproto message
func cmdEnqueueOutboundPb(cmd []byte) {

    // Convert it to the new-format protocol buffer
    header := []byte{BUFF_FORMAT_PB_ARRAY, 1}
    header = append(header, byte(len(cmd)))
    command := append(header, cmd...)

	// Enqueue it
	cmdEnqueueOutboundPayload(command)
	
}

// Enqueue an outbound message that already has a PB_ARRAY header
func cmdEnqueueOutboundPayload(cmd []byte) {
    var ocmd OutboundCommand
    ocmd.Command = cmd
    outboundQueue <- ocmd
}

// Send the pending outbound (from command processing goroutine)
func SentPendingOutbound() bool {
    hexchar := []byte("0123456789ABCDEF")

    // Check to see if the service is currently offline and
    // if we recently received a message from a device that
    // will be interested in that fact.  We do this by packaging
    // the message as though it were sent by TTSERVE itself.
    if (deviceToNotifyIfServiceDown != 0 && !isTeletypeServiceReachable()) {
        msg := &ttproto.Telecast{}
        msg.Message = proto.String("down")
        deviceType := ttproto.Telecast_TTSERVE
        msg.DeviceType = &deviceType
        deviceID := deviceToNotifyIfServiceDown
        msg.DeviceId = &deviceID
        data, err := proto.Marshal(msg)
        if err == nil {
            // This will be dequeued below
            cmdEnqueueOutboundPb(data)
        }
        // Nullify so that we don't send the message more than once
        deviceToNotifyIfServiceDown = 0
    }

    // We test this because we can never afford to block here,
    // and we knkow that we're the only consumer of this queue
    if len(outboundQueue) != 0 {

        for ocmd := range outboundQueue {

            // Convert it to a hex commnd
            outbuf := []byte("radio tx ")
            for _, databyte := range ocmd.Command {
                loChar := hexchar[(databyte & 0x0f)]
                hiChar := hexchar[((databyte >> 4) & 0x0f)]
                outbuf = append(outbuf, hiChar)
                outbuf = append(outbuf, loChar)
            }

            // Send it
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
    buf := make([]byte, len(hex)/2)
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

        buf[i] = (hinibble << 4) | lonibble

    }

    // Make sure that we understand the format of the message.
    msg := &ttproto.Telecast{}
    buf_format := buf[0]
    switch (buf_format) {

    case BUFF_FORMAT_SINGLE_PB: {

        // Unpack the received message which is a protocol buffer
        err := proto.Unmarshal(buf, msg)
        if err != nil {
            fmt.Printf("*** message not recognized - likely a LoRaWAN transmission ***\n");
            return
        }

        // Output a debug message, because this should no longer be being received
        fmt.Printf("*** WARNING: OLD WIRE FORMAT: %d\n", msg.GetDeviceId())
    }

    case BUFF_FORMAT_PB_ARRAY: {
        count := int(buf[1])
        lengthArrayOffset := 2
        payloadOffset := lengthArrayOffset + count

        // For now, we only support single-PB messages.  If we need to support more,
        // this will be trivial because we can just transmit the msg as-is while just
        // iterating over it to extract data to be displayed on local HDMI monitor.
        if (count != 1) {
            fmt.Printf("*** ERROR: FOR NOW WE ONLY SUPPORT 1-MESSAGE PAYLOADS\n");
            return
        }
        i := 0

        // Extract the length
        length := int(buf[lengthArrayOffset+i])

        // Unmarshal payload
        payload := buf[payloadOffset:payloadOffset+length]
        err := proto.Unmarshal(payload, msg)
        if err != nil {
            fmt.Printf("*** message not recognized - likely a LoRaWAN transmission ***\n");
            return
        }

    }
    }

    // Remember the Device ID number of the last received message, for failover purposes
    if (msg.DeviceId != nil) {
        deviceToNotifyIfServiceDown = msg.GetDeviceId()
    }

    // Process it as a Telecast message
    cmdProcessReceivedTelecastMessage(*msg, buf, snr)

}

// Commands for setting frequency
func lorafp_get_command(cmdno int) (bool, string) {

    switch strings.ToLower(Region) {

    case "eu":
        eu_commands := []string{
            "radio set mod lora",
            "radio set freq 868100000",
        }
        if cmdno < len(eu_commands) {
            return true, eu_commands[cmdno]
        }

    case "us":
        us_commands := []string{
            "radio set mod lora",
            "radio set freq 923300000",
        }
        if cmdno < len(us_commands) {
            return true, us_commands[cmdno]
        }

    }

    return false, ""

}
