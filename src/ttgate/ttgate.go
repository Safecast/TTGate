/*
 * Testing...
 *
 * Contributors:
 *    Ray Ozzie
 */

package main

import (
    "fmt"
    "time"
    "log"
    "github.com/tarm/serial"
)

func main() {

    c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 57600}
    s, err := serial.OpenPort(c)
    if (err != nil) {
        log.Fatal(err);
    }

    for i := 0; ; i++ {
        fmt.Printf("sys get ver:")

        n, err := s.Write([]byte("sys get ver\r\n"))
        if (err != nil) {
            fmt.Printf("write err: %d");
        } else {
            buf := make([]byte, 128)
            n, err = s.Read(buf)
            if (err != nil) {
                fmt.Printf("read err: %d");
            } else {
                log.Printf("%q", buf[:n])
            }
        }
        time.Sleep(1 * time.Second)
    }

}
