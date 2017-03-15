// Copyright 2017 Inca Roads LLC.  All rights reserved. 
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

// Low-level I/O functions for gateway
package main

import (
    "bytes"
    "fmt"
    "io"
    "os"
    "time"
    "github.com/tarm/serial"
    "github.com/stianeikeland/go-rpio"
)

// Statics
var verboseDebug = false
var serialInitCompleted = false
var serialPort *serial.Port
var rpioIsOpen = false
var replyWatchdogEnabled = false
var replyWatchdogTickCount = 0
var flushBufferedData = false

// Initialize the i/o subsystem
func ioInit() {

    // Has been useful for hardware reset & serial port I/O debugging
    verboseDebug = false
    verbose := os.Getenv("VERBOSE")
    if verbose != "" {
        verboseDebug = true
    }

    // Useful for debugging in non-RPi environments, like your Mac
    port := os.Getenv("SERIAL")
    if port == "" {
        port = "/dev/ttyS0"
    }

    // This is the default speed for the Microchip RN2483/2903
    speed := 57600

    // Open the serial port
    s, err := serial.OpenPort(&serial.Config{Name: port, Baud: speed})
    if err != nil {
        fmt.Printf("Cannot open %s\n", port)
        return
    }
    serialPort = s
    serialInitCompleted = true

    // Allow for noise on the newly-opened serial port to get buffered in one large chunk
    time.Sleep(2 * time.Second)

    // Reset the watchdog timer used to notify us that the chip is wedged
    ioReplyWatchdogReset(false)

    // Process receives in a different goroutine because I/O is synchronous
    go InboundMain()

}

// Initialize the Microchip RN2483/RN2903 LPWAN controller
func ioInitMicrochip() {

    // The inbound task buffers incoming data until it gets a newline.
    // If we've accumulated buffered data, we need to force that goroutine to discard it.
    flushBufferedData = true

    // Leave the Raspberry Pi's GPIO open forever while we are running
    if !rpioIsOpen {
        err := rpio.Open()
        if err != nil {
            fmt.Printf("ioInitMicrochip: err %v\n", err)
            return
        }
        rpioIsOpen = true
    }

    // Perform a hardware reset of the device. Note that this requires two things to be true:
    // 1) On the back side of the RN2483/RN2903, use solder to close the gap of SJ1, which brings /RESET to Xbee Pin 17
    // 2) Wire Xbee Pin 17 to the RPi's Pin 18 BCM Pin 24: http://pinout.xyz/pinout/pin18_gpio24
    // Note that the 250ms reset and 5s settling period have been carefully determined and are very reliable.
    pin := rpio.Pin(24) // BCM pin # on Raspberry Pi Pinout
    pin.Output()
    pin.Low()
    time.Sleep(250 * time.Millisecond)
    pin.High()
    time.Sleep(5 * time.Second)

    fmt.Printf("\nLPWAN Hardware Reset\n\n")

}

// The inbound I/O goroutine used for handling of inbound synchronous serial I/O
func InboundMain() {

    // Two I/O buffers - one for the current read, and
    // the other containing the previous read's unprocessed data
    const bufsize = 1024
    var thisbuf = make([]byte, bufsize)
    var prevbuf []byte = []byte("")

    // Wait until init completed
    for !serialInitCompleted {
        time.Sleep(2 * time.Second)
    }

    // Primary I/O loop
    for {

        // We sleep before every read just to give the serial package a chance to accumulate
        // a buffer of characters rather than thrashing on every byte becoming ready.
        time.Sleep(100 * time.Millisecond)

        // Do the read
        n, err := serialPort.Read(thisbuf)
        if err != nil {
            if err != io.EOF {
                fmt.Printf("serial: read error %v\n", err)
            }

        } else {

            // The bufsize check is a sledgehammer that we use because just after reset
            // we've observed that we get LARGE buffers of zero and other noise.
            // If we ever get a full buffer, we just discard it.
            if n != 0 && n != bufsize {

                if verboseDebug {
                    fmt.Printf("read(%d): '%s'\n% 02x\n", n, thisbuf[:n], thisbuf[:n])
                }

                // If we've been asked to flush the data because this is the first
                // read after a hardware reset, do so.  Else, process it.
                if flushBufferedData {
                    flushBufferedData = false
                    ProcessInbound(thisbuf[:n])
                } else {
                    prevbuf = ProcessInbound(append(prevbuf[:], thisbuf[:n]...))
                }

            }
        }
    }

}

// Process data that has come inbound on the serial port
func ProcessInbound(buf []byte) []byte {

    length := len(buf)
    begin := 0
    end := 0

    // Skip over leading trash (such as nulls) that we see after a reset; this is an ASCII protocol
    for begin = 0; begin < length; begin++ {
        if buf[begin] == '\r' || buf[begin] == '\n' || (buf[begin] >= ' ' && buf[begin] < 0x7f) {
            break
        }
    }

    // Loop over the buffer, which could have multiple lines in it
    for begin < length {

        // Parse out a single line delineated by begin:end
        for end = begin; end < length; end++ {

            // Process the line if it ends in \r\n or \r or \n
            if buf[end] == '\r' || buf[end] == '\n' {

                // Process if non-blank (which it will be on the \n of \r\n)
                if end > begin {

                    // Reset the command watchdog because we received a reply
                    ioReplyWatchdogReset(false)

                    // Feed this line buffer to the state machine
                    cmdProcess(buf[begin:end])

                }

                // Skip past this delimeter and look for the next command
                begin = end + 1
                break

            }
        }

        // Done with buffer?
        if end >= length {
            break
        }

    }

    // Return the unprocessed portion of the buffer for next time
    if verboseDebug && replyWatchdogEnabled {
        fmt.Printf("Unprocessed: '%s' -> '%s'\n", buf, buf[begin:])
    }

    return (buf[begin:])

}

// Reset the watchdog timer as enabled or disabled
func ioReplyWatchdogReset(fEnable bool) {
    replyWatchdogEnabled = fEnable
    replyWatchdogTickCount = 0
}

// Monitor serial I/O as a way of handling the Microchip getting into a locked state
func io5sWatchdog() {
    // Process the watchdog monitoring request/response from the LPWAN chip
    if replyWatchdogEnabled {
        replyWatchdogTickCount = replyWatchdogTickCount + 1
        if (replyWatchdogTickCount >= 5) {
            fmt.Printf("*** ioReplyWatchdog: no cmd reply!\n")
            // Exit, which will cause our
            // shell script to restart the container.  This is a failsafe
            // to ensure that any Linux-level process usage (such as bugs in
            // the golang runtime or Midori) will be reset, and we will
            // occasionally start completely fresh and clean.
			if (replyWatchdogTickCount >= 100) {
	            os.Exit(0)
			}
        }
    }
}

// Send a string as a full newline-delimited command to the serial port
func ioSendCommandString(cmd string) {
    ioSendCommand([]byte(cmd))
}

// Send bytes to the serial port as a full newline-delimited command
func ioSendCommand(cmd []byte) {

    fmt.Printf("send(%s)\n", cmd)

    // Write this, appending newline
    if (serialInitCompleted) {
        _, err := serialPort.Write(bytes.Join([][]byte{cmd, []byte("")}, []byte("\r\n")))
        if err != nil {
            fmt.Printf("write err: %d", err)
        }
    }

    // Set the watchdog timer because we've successfully written to the port,
    // and we are now awaiting a reply from the chip.
    ioReplyWatchdogReset(true)

}
