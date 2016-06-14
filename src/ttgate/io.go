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

// Initialize the i/o subsystem

func ioInit() bool {

    port := os.Getenv("SERIAL")
    if (port == "") {
        port = "/dev/ttyS0"
    }

    speed := 57600

    s, err := serial.OpenPort(&serial.Config{Name: port, Baud: speed, ReadTimeout: (time.Second * 60 * 3)})
    if (err != nil) {
        fmt.Printf("Cannot open %s\n", port)
        return false
    }
    serialPort = s

    fmt.Printf("Serial I/O Initialized\n")

    // Give the port a real chance of initializing
    time.Sleep(5 * time.Second)

    // Initialize the command processing and state machine
    cmdInit()

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

	pin.Low()
	time.Sleep(time.Second)
	pin.High()
	time.Sleep(time.Second)
	pin.Low()
	time.Sleep(time.Second)
//	pin.Toggle()       // Toggle pin (Low -> High -> Low)

	rpio.Close()

//    time.Sleep(10 * time.Second)
	fmt.Printf("ioInitMicrochip: ...completed\n");
	
}

func InboundMain() {

    var thisbuf = make([]byte, 128)
    var prevbuf []byte = []byte("")

    for {
        n, err := serialPort.Read(thisbuf)
        if (err != nil) {
            if (err != io.EOF) {
                fmt.Printf("serial: read error %v\n", err)
            }
        } else {
            prevbuf = ProcessInbound(bytes.Join([][]byte{prevbuf, thisbuf[:n]}, []byte("")))
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
