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

    for {
        time.Sleep(5 * time.Second)
//		ioReadIncoming()
    }

}

func ioReadIncoming() {
    var thisbuf = make([]byte, 128)     // make this big after we know that it works
    var prevbuf []byte = []byte("")
    n, err := serialPort.Read(thisbuf)
    if (err != nil) {
        fmt.Printf("read err: %d", err)
    } else {
        prevbuf = ProcessInbound(bytes.Join([][]byte{prevbuf, thisbuf[:n]}, []byte("")))
    }
}

func ProcessInbound(buf []byte) []byte  {

    length := len(buf)
    begin := 0
    end := 0

    fmt.Printf("ProcessInbound(%s)\n", buf)

    for begin<length {

        // Skip leading cr and lf lying around in buffer

        for ; begin<length; begin++ {
            if (buf[begin] != '\r' && buf[begin] != '\n') {
                break
            }
        }

        // Scan to see if there's a cr or lf in this buffer,
        // and just exit if not.

        for end = begin; end<length; end++ {
            if (buf[end] == '\r' || buf[end] == '\n') {
                cmdProcess(buf[begin:end])
                begin = end
                break
            }
        }

        // If we've processed it all, stop

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

    fmt.Printf("ioSendCommand(%s)\n", bytes.Join([][]byte{cmd, []byte("")}, []byte("\r\n")))

    _, err := serialPort.Write(bytes.Join([][]byte{cmd, []byte("")}, []byte("\r\n")))
    if (err != nil) {
        fmt.Printf("write err: %d", err)
    }

}

// eof
