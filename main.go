// Teletype Gateway
package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
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

    fmt.Printf("Teletype Gateway\n")

    // Load localization information to be used for the HDMI status display
    loadLocalTimezone()

    // Resin debugging via Terminal requires halting the main instance via env var
    s := os.Getenv("HALT")
    if s != "" {
        fmt.Printf("HALT environment variable detected\n")
        fmt.Printf("Exiting.\n")
        os.Exit(0)
    }

    // Spawn our localhost web server, used to update the HDMI status display
    go webServer()

    // Spawn housekeeping and watchdog tasks
    go timer1m()
    go timer5s()

    // Initialize I/O devices
    ioInit()

    // Initialize the state machine and command processing
    cmdInit()

    // Infinitely loop, updating statistics
    bootedAt := time.Now()
    for {
        time.Sleep(15 * 60 * time.Second)

        // Print stats
        t := time.Now()
        hoursAgo :=  int64(t.Sub(bootedAt) / time.Hour)
        minutesAgo := int64(t.Sub(bootedAt) / time.Minute) - (hoursAgo * 60)
        fmt.Printf("\nSTATS: %d received in the last %dh %dm\n\n", cmdGetStats(), hoursAgo, minutesAgo)

        // Print resource usage, just as an FYI
        var mem runtime.MemStats
        runtime.ReadMemStats(&mem)
        fmt.Printf("mem.Alloc: %d\n", mem.Alloc)
        fmt.Printf("mem.TotalAlloc: %d\n", mem.TotalAlloc)
        fmt.Printf("mem.HeapAlloc: %d\n", mem.HeapAlloc)
        fmt.Printf("mem.HeapSys: %d\n", mem.HeapSys)

    }

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
        webUpdateData()
    }
}

// Load localization information
func loadLocalTimezone() {

    // Default to UTC, with NO country standards, if we can't find our own info
    OurTimezone, _ = time.LoadLocation("UTC")
    OurCountryCode = ""

    // Use the ip-api service, which handily provides the needed info
    response, err := http.Get("http://ip-api.com/json/")
    if err == nil {
        defer response.Body.Close()
        contents, err := ioutil.ReadAll(response.Body)
        if err == nil {
            var info IPInfoData
            err = json.Unmarshal(contents, &info)
            if err == nil {
                OurTimezone, _ = time.LoadLocation(info.Timezone)
                OurCountryCode = info.CountryCode
            }
        }
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
