// Teletype Gateway
package main

import (
    "fmt"
    "net/http"
    "os"
    "time"
	"runtime"
)

// Statics
var OurTimezone *time.Location
var OurCountryCode string = ""

// Main entry point when launched by run.sh
func main() {

	// Welcome
    fmt.Printf("\nTeletype Gateway\n")

    // Load localization information to be used for the HDMI status display
    loadLocalTimezone()

    // Spawn our localhost web server, used to update the HDMI status display
    go webServer()

    // Spawn housekeeping and watchdog tasks
    go timer15m()
    go timer5m()
    go timer1m()
    go timer5s()

    // Initialize I/O devices
    ioInit()

    // Initialize the state machine and command processing
    cmdInit()

	// Wait for quite a while, and then exit, which will cause our
	// shell script to restart the container.  This is a failsafe 
	// to ensure that any Linux-level process usage (such as bugs in
	// the golang runtime or Midori) will be reset, and we will
	// occasionally start completely fresh and clean.
    time.Sleep(7 * 24 * time.Hour)
    os.Exit(0)
	
}

// Timer functions
func timer5s() {
    for {
        time.Sleep(5 * time.Second)
        io5sWatchdog()
    }
}

func timer1m() {
    for {
        time.Sleep(1 * 60 * time.Second)
        cmd1mWatchdog()
    }
}

func timer5m() {
    for {
        time.Sleep(5 * 60 * time.Second)
		cmdPingTeletypeService()
	}
}

func timer15m() {
	var memBaseSet bool = false
	var memBase runtime.MemStats
    bootedAt := time.Now()

    for {
        time.Sleep(15 * 60 * time.Second)

		// Get original memory statistics, before we've done anything at all
		if (!memBaseSet) {
			memBaseSet = true;
			runtime.ReadMemStats(&memBase)
		}
		
		fmt.Printf("\n")

        // Print stats
        t := time.Now()
        hoursAgo :=  int64(t.Sub(bootedAt) / time.Hour)
        minutesAgo := int64(t.Sub(bootedAt) / time.Minute) - (hoursAgo * 60)
        fmt.Printf("STATS: %d received in the last %dh %dm\n", cmdGetStats(), hoursAgo, minutesAgo)
		fmt.Printf("\n")

        // Print resource usage, just as an FYI
        var mem runtime.MemStats
        runtime.ReadMemStats(&mem)
        fmt.Printf("mem.Alloc: %d -> %d\n", memBase.Alloc, mem.Alloc)
        fmt.Printf("mem.HeapAlloc: %d -> %d\n", memBase.HeapAlloc, mem.HeapAlloc)
        fmt.Printf("mem.HeapObjects: %d -> %d\n", memBase.HeapObjects, mem.HeapObjects)
        fmt.Printf("mem.HeapSys: %d -> %d\n", memBase.HeapSys, mem.HeapSys)
		fmt.Printf("\n")

    }
}

// Load localization information
func loadLocalTimezone() {

    // Default to UTC, with NO country standards, if we can't find our own info
    OurTimezone, _ = time.LoadLocation("UTC")
    OurCountryCode = ""

}

// The localhost server used exclusively to update the local HDMI display
func webServer() {
    http.Handle("/", http.FileServer(http.Dir("./web")))
    http.ListenAndServe(":8080", nil)
}
