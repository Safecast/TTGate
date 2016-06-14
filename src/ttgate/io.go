/*
 * Teletype Gateway Serial I/O
 *
 * This module contains the actual I/O interface, as well as the goroutines
 * that dispatch inbound and outbound traffic.
 */

package main

import (
    "fmt"
    "time"
    "bytes"
    "os"
    "io"
    "net"
    "github.com/tarm/serial"
    "github.com/stianeikeland/go-rpio"
)

var serialPort *serial.Port
var reinitRequested = false

// Initialize the i/o subsystem

func ioInit() bool {

    port := os.Getenv("SERIAL")
    if (port == "") {
        port = "/dev/ttyS0"
    }

    speed := 57600

    s, err := serial.OpenPort(&serial.Config{Name: port, Baud: speed, ReadTimeout: (time.Second * 5)})
    if (err != nil) {
        fmt.Printf("Cannot open %s\n", port)
        return false
    }
    serialPort = s

    time.Sleep(5 * time.Second)

    fmt.Printf("Serial I/O Initialized\n")


    // Process receives on a different thread because I/O is synchronous
    go InboundMain()

    // Success
    return true

}

func ioInitMicrochip() {

    err := rpio.Open()
    if (err != nil) {
        fmt.Printf("ioInitMicrochip: err %v\n", err)
        return;
    }

    fmt.Printf("ioInitMicrochip: Hardware reset...\n")

    // Note that this requires two things to be true:
    // 1) On the back side of the RN2483/RN2903, use solder to close the gap of SJ1, which brings /RESET to Xbee Pin 17
    // 2) Wire Xbee Pin 17 to the RPi's header Pin 36, which is BCM Pin 16 (http://pinout.xyz/pinout/pin36_gpio16)
    pin := rpio.Pin(16)

    pin.Output()       // Output mode

    pin.Toggle()       // Toggle pin (Low -> High -> Low)
    time.Sleep(time.Second * 1)
    pin.Toggle()       // Toggle pin (Low -> High -> Low)
    time.Sleep(time.Second * 5)

    rpio.Close()

    fmt.Printf("ioInitMicrochip: ...completed\n");

}

func ioRequestReinit() {
    reinitRequested = true;
}

func InboundMain() {

    // Do all initialization and reinitialization in this goroutine
    cmdInit()

    // Loop reading from input, and dispatching to process it if something is received
    var thisbuf = make([]byte, 128)
    var prevbuf []byte = []byte("")

    for {

        fmt.Printf("<rd>\n")
        n, err := serialPort.Read(thisbuf)
        fmt.Printf("</rd>\n")

        if (err == io.EOF) {
            err = nil
            n = 0
        }

        if (n == 0) {
            if reinitRequested {
                reinitRequested = false;
                cmdReinit()
            } else {
                time.Sleep(100 * time.Millisecond)
            }
        }

        if (err != nil) {
            n = 0
            fmt.Printf("serial: read error %v\n", err)
        }

        if (n != 0) {
            prevbuf = ProcessInbound(bytes.Join([][]byte{prevbuf, thisbuf[:n]}, []byte("")))
        }
    }
}

}

func ProcessInbound(buf []byte) []byte  {

length := len(buf)
begin := 0
end := 0

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

fmt.Printf("ioSendCommand(%s) (sent)\n", cmd)

}

func getDeviceID() string {
ifs, _ := net.Interfaces()
for _, v := range ifs {
    h := v.HardwareAddr.String()
    if len(h) == 0 {
        continue
    }
    return(h)
}
return("");
}

// eof
