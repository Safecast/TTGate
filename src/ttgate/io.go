/*
 * Teletype Gateway Serial I/O
 *
 * This module contains the actual I/O interface, as well as the goroutines
 * that dispatch inbound and outbound traffic.
 */

package main

import (
    "fmt"
    "bytes"
    "time"
    "github.com/tarm/serial"
)

var serialPort *serial.Port

// Initialize the i/o subsystem

func ioInit() bool {

    port := "/dev/ttyS0"
    speed := 57600

    s, err := serial.OpenPort(&serial.Config{Name: port, Baud: speed})
    if (err != nil) {
        fmt.Printf("Cannot open %s\n", port)
        return false
    }

    serialPort = s

    // Process receives on a different thread because I/O is synchronous

    go InboundMain()

    // Initialize the command processing and state machine

    cmdInit()

    // Success

    return true

}

func InboundMain() {

    fmt.Printf("Inbound Task Initiated\n");

    var thisbuf = make([]byte, 128)
    var prevbuf []byte = []byte("")

    for {
        n, err := serialPort.Read(thisbuf)
        if (err != nil) {
            fmt.Printf("read err: %d", err)
        } else {
            prevbuf = ProcessInbound(bytes.Join([][]byte{prevbuf, thisbuf[:n]}, []byte("")))
        }
    }

}

func ProcessInbound(buf []byte) []byte  {

    length := len(buf)
    begin := 0
    end := 0

    fmt.Printf("ProcessInbound(%s)\n", buf)

    // Loop over the buffer, which could have multiple lines in it

    for begin<length {

        // Parse out a single line delineated by begin:end

        for end = begin; end<length; end++ {

            // Process the line if it ends in \r\n or \r or \n

            if (buf[end] == '\r' || buf[end] == '\n') {

                // Process if non-blank (which it will be on the \n of \r\n)

                if (end > begin) {
                    cmdProcess(buf[begin:end])
                }

                // Skip past this delimeter and look for the next command

                begin = end+1
                break

            }
        }

        if (end >= length) {
            break
        }

    }

    // Return unprocessed portion of the buffer for next time

    return(buf[begin:])

}

func ioSendCommandString(cmd string) {
    ioSendCommand([]byte(cmd))
}

func ioSendCommand(cmd []byte) {

    fmt.Printf("ioSendCommand(%s)\n", cmd)

    _, err := serialPort.Write(bytes.Join([][]byte{cmd, []byte("")}, []byte("\r\n")))
    if (err != nil) {
        fmt.Printf("write err: %d", err)
    }

}

// eof
