// Copyright 2017 Inca Roads LLC.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

// Gateway between TTNode-based LoRa devices and the TTServe service
package main

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "time"
    "runtime"
    "strconv"
)

// OurTimezone is the time zone of the gateway
var OurTimezone *time.Location
// OurCountryCode is the country code of the gateway
var OurCountryCode string
// DebugFailover is true if the gateway is suggesting FailOver mode to clients
var DebugFailover = false

// Main entry point when launched by run.sh
func main() {

    // Welcome
    go fmt.Printf("\nLora Gateway\n")

    // Debug flags
    s := os.Getenv("DEBUG_FAILOVER")
    i, err := strconv.ParseInt(s, 10, 64)
    DebugFailover = (err == nil && i != 0)

    // Load localization information
    loadLocalTimezone()

    // Translate the DNS address to an IP address, because this can be slow
    UpdateTargetIP()

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
	minutesAlive := 0
    for {
		// For the first 5 minutes, send pings to service.  After
		// that, it's up to the 5m timer handler.
		if minutesAlive < 5 {
	        cmdSendStatsToTeletypeService()
		}
		
        // Time out commands
        cmd1mWatchdog()

        // Update what's on the browser connected to HDMI
        webUpdateData()

        // Sleep
        time.Sleep(1 * 60 * time.Second)
		minutesAlive++
    }
}

func timer5m() {
    for {
        cmdSendStatsToTeletypeService()
        UpdateTargetIP()
        time.Sleep(5 * 60 * time.Second)
    }
}

func timer15m() {
    var memBaseSet = false
    var memBase runtime.MemStats
    bootedAt := time.Now()

    for {
        time.Sleep(15 * 60 * time.Second)

        // Get original memory statistics, before we've done anything at all
        if (!memBaseSet) {
            memBaseSet = true;
            runtime.ReadMemStats(&memBase)
        }

        go fmt.Printf("\n")

        // Print stats
        t := time.Now()
        hoursAgo :=  int64(t.Sub(bootedAt) / time.Hour)
        minutesAgo := int64(t.Sub(bootedAt) / time.Minute) - (hoursAgo * 60)
        go fmt.Printf("STATS: %d received in the last %dh %dm\n", cmdGetStats(), hoursAgo, minutesAgo)
        go fmt.Printf("\n")

        // Print resource usage, just as an FYI
        var mem runtime.MemStats
        runtime.ReadMemStats(&mem)
        if (false) {
            go fmt.Printf("mem.Alloc: %d -> %d\n", memBase.Alloc, mem.Alloc)
            go fmt.Printf("mem.HeapAlloc: %d -> %d\n", memBase.HeapAlloc, mem.HeapAlloc)
            go fmt.Printf("mem.HeapObjects: %d -> %d\n", memBase.HeapObjects, mem.HeapObjects)
            go fmt.Printf("mem.HeapSys: %d -> %d\n", memBase.HeapSys, mem.HeapSys)
            go fmt.Printf("\n")
        }

        // Reboot the server if things get really borked
        if isOfflineForExtendedPeriod() {
            go fmt.Printf("Cannot reach service for many, many hours: rebooting device.\n");
            os.Exit(0)
        }

    }
}

// Load localization information
func loadLocalTimezone() {

    // Use the ip-api service, which handily provides the needed info
    isAvail, _, info := GetIPInfo()
    if !isAvail {
        // Default to UTC, with NO country standards, if we can't find our own info
        OurTimezone, _ = time.LoadLocation("UTC")
        OurCountryCode = ""
    } else {
        OurTimezone, _ = time.LoadLocation(info.Timezone)
        OurCountryCode = info.CountryCode
    }

}

// The localhost server used exclusively to update the local HDMI display
func webServer() {
    http.Handle("/", http.FileServer(http.Dir("./web")))
    http.ListenAndServe(":8080", nil)
}

// This periodically updates the JSON data file periodically reloaded by index.html
func webUpdateData() {
    buffer := GetSafecastDataAsJSON()
    ioutil.WriteFile("./web/data.json", buffer, 0644)
}
