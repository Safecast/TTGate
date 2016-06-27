/*
 *	Teletype Gateway
 *
 */

package main

import (
    "time"
    "fmt"
    "os"
    "encoding/json"
    "runtime"
	"net"
    "net/http"
    "io/ioutil"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
)

type IPInfoData struct {
	AS			 string `json:"as"`
	City         string `json:"city"`
	Country      string `json:"country"`
	CountryCode  string `json:"countryCode"`
	ISP			 string `json:"isp"`
	Latitude	 float32 `json:"lat"`
	Longitude	 float32 `json:"lon"`
	Organization string `json:"org"`
	IP           net.IP `json:"query"`
	Region       string `json:"region"`
	RegionName   string `json:"regionName"`
	Timezone     string `json:"timezone"`
	Zip			 string `json:"zip"`
}

var debug bool = false
var OurTimezone *time.Location

func main() {
    var s string

    fmt.Printf("Teletype Gateway\n")

    s = os.Getenv("HALT")       // Resin debugging via terminal requires quitting the main instance
    if (s != "") {
        fmt.Printf("HALT environment variable detected\n");
        fmt.Printf("Exiting.\n");
        os.Exit(0);
    }

    s = os.Getenv("DEBUG")      // For verbose debugging info
    debug = s != ""

	// Spawn our localhost web server

	loadLocalTimezone()

	go webServer()
	
	// Spawn various timer tasks

    go timer15m()
	go timer1m()
	go timer5s()

	// Initialize I/O devices

    ioInit()

    // Initialize the command processing and state machine

    cmdInit()
	
	// Infinitely loop here

    for {
        time.Sleep(30 * time.Second)
    }
	

}

func timer5s() {
    for {
        time.Sleep(5 * time.Second)
        ioWatchdog()
    }
}

func timer1m() {
    for {
        time.Sleep(1 * 60 * time.Second)
        cmdWatchdog1m()
		webUpdateData()
    }
}

func timer15m() {
    for {
        time.Sleep(15 * 60 * time.Second)
        heartbeat15m()
    }
}

func heartbeat15m() {

    // Get the stats in the form of a message

    totalReceived, totalSent := cmdGetStats()
    message := fmt.Sprintf("#gateway received %d sent %d", totalReceived, totalSent)
    fmt.Printf("%s\n", message);

    // Broadcast a test message

    deviceType := teletype.Telecast_TTGATE
    msg := &teletype.Telecast {}
    msg.DeviceType = &deviceType
    msg.DeviceIDString = proto.String(getDeviceID())
    msg.Message = proto.String(message)
    data, err := proto.Marshal(msg)
    if err != nil {
        fmt.Printf("marshaling error: ", err)
    }
    cmdEnqueueOutbound(data)

    // Print resource usage, just as an FYI

    var mem runtime.MemStats
    runtime.ReadMemStats(&mem)
    fmt.Printf("mem.Alloc: %d\n", mem.Alloc)
    fmt.Printf("mem.TotalAlloc: %d\n", mem.TotalAlloc)
    fmt.Printf("mem.HeapAlloc: %d\n", mem.HeapAlloc)
    fmt.Printf("mem.HeapSys: %d\n", mem.HeapSys)

}

func loadLocalTimezone () {

	// Get our local time zone, defaulting to UTC
	// Note:
	// https://golang.org/src/time/example_test.go
	// https://golang.org/pkg/time
	
	OurTimezone, _ = time.LoadLocation("UTC")
	
	response, err := http.Get("http://ip-api.com/json/")
	if err == nil {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err == nil {
			var info IPInfoData
			err = json.Unmarshal(contents, &info)
			if (err == nil) {
				OurTimezone, _ = time.LoadLocation(info.Timezone)
			}
		}
	}

}

func webServer() {
	http.Handle("/", http.FileServer(http.Dir("./web")))
    http.ListenAndServe(":8080", nil)
}

func webUpdateData() {

	// Get the sorted list of device info
    sorted := GetSortedDeviceList()

	// Marshall it to text
    buffer, _ := json.MarshalIndent(sorted, "", "    ")

	// Write it
	ioutil.WriteFile("./web/data.json", buffer, 0644)
	
}

// eof
