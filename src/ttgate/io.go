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

var watchdog5s = false
var watchdog5sCount = 0
var verboseDebug = false;

// Initialize the i/o subsystem

func ioInit() {

    verboseDebug = false;
    verbose := os.Getenv("VERBOSE")
    if (verbose != "") {
        verboseDebug = true;
    }

    port := os.Getenv("SERIAL")
    if (port == "") {
        port = "/dev/ttyS0"
    }

    speed := 57600

    s, err := serial.OpenPort(&serial.Config{Name: port, Baud: speed})
    if (err != nil) {
        fmt.Printf("Cannot open %s\n", port)
        return
    }
    serialPort = s

    time.Sleep(2 * time.Second)

    fmt.Printf("Serial I/O Initialized\n")

    // Reset the watchdog timer
    ioWatchdogReset(false)

    // Process receives on a different thread because I/O is synchronous
    go InboundMain()

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
    // 2) Wire Xbee Pin 17 to the RPi's Pin 18 BCM Pin 24: http://pinout.xyz/pinout/pin18_gpio24
    pin := rpio.Pin(24)// BCM pin # on Raspberry Pi Pinout
    pin.Output()       // Output mode
    pin.Toggle()       // Toggle pin (Low -> High -> Low)
    rpio.Close()

    time.Sleep(10 * time.Second)
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
            if (n == 0) {
                time.Sleep(250 * time.Millisecond)
            } else {
                prevbuf = ProcessInbound(bytes.Join([][]byte{prevbuf, thisbuf[:n]}, []byte("")))
                // ** When debugging how the input stream actually appears on comm channel
                //                if (len(prevbuf) != 0) {
                //                    fmt.Printf("serial pending: (%s)\n[", string(prevbuf))
                //                    for _, databyte := range prevbuf {
                //                        fmt.Printf("%02x", databyte)
                //                    }
                //                    fmt.Printf("]\n")
                //                }
                // **
            }
        }
    }

}

func ProcessInbound(buf []byte) []byte  {

    length := len(buf)
    begin := 0
    end := 0

    // Skip over leading trash (such as nulls) that we see after a reset; this is an ASCII protocol

    for begin=0; begin<length; begin++ {
        if (buf[begin] == '\r' || buf[begin] == '\n' || (buf[begin] >= ' ' && buf[begin] < 0x7f)) {
            break
        }
    }

    // Loop over the buffer, which could have multiple lines in it

    for begin<length {

        // Parse out a single line delineated by begin:end

        for end = begin; end<length; end++ {

            // Process the line if it ends in \r\n or \r or \n

            if (buf[end] == '\r' || buf[end] == '\n') {

                // Process if non-blank (which it will be on the \n of \r\n)

                if (end > begin) {
                    watchdog5s = false;
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

    if (verboseDebug && watchdog5s) {
        fmt.Printf("Unprocessed: '%s' -> '%s'\n", buf, buf[begin:])
    }

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

    ioWatchdogReset(true)

}

func ioWatchdogReset(fEnable bool) {
    watchdog5s = fEnable
    watchdog5sCount = 0
}

func ioWatchdog5s() {
    // Ignore the first increment, which could occur at any time in the first interval.
    // But then, on the second increment, reset the world.

    if (verboseDebug) {
        if (watchdog5s) {
            fmt.Printf("ioWatchdog5s() %d\n", watchdog5sCount);
        } else {
            fmt.Printf("ioWatchdog5s() idle\n");
        }
    }

    if (watchdog5s) {
        watchdog5sCount = watchdog5sCount + 1
        switch (watchdog5sCount) {
        case 1:
        case 2:
            fmt.Printf("*** ioWatchdog: Warning!\n")
        case 3:
            fmt.Printf("*** ioWatchdog: Reinitializing!\n")
            ioWatchdogReset(false);
            cmdReinit(true)
        }
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
