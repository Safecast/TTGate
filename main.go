/*
 *	Teletype Gateway
 *
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"
)

type IPInfoData struct {
	AS           string  `json:"as"`
	City         string  `json:"city"`
	Country      string  `json:"country"`
	CountryCode  string  `json:"countryCode"`
	ISP          string  `json:"isp"`
	Latitude     float32 `json:"lat"`
	Longitude    float32 `json:"lon"`
	Organization string  `json:"org"`
	IP           net.IP  `json:"query"`
	Region       string  `json:"region"`
	RegionName   string  `json:"regionName"`
	Timezone     string  `json:"timezone"`
	Zip          string  `json:"zip"`
}

var debug bool = false
var OurTimezone *time.Location
var bootedAt time.Time
var OurCountryCode string = ""

func main() {
	var s string

	fmt.Printf("Teletype Gateway\n")
	bootedAt = time.Now()

	s = os.Getenv("HALT") // Resin debugging via terminal requires quitting the main instance
	if s != "" {
		fmt.Printf("HALT environment variable detected\n")
		fmt.Printf("Exiting.\n")
		os.Exit(0)
	}

	s = os.Getenv("DEBUG") // For verbose debugging info
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
		heartbeat15m()
		time.Sleep(15 * 60 * time.Second)
	}
}

func heartbeat15m() {
	t := time.Now()
	hoursAgo :=  int64(t.Sub(bootedAt) / time.Hour)
	minutesAgo := int64(t.Sub(bootedAt) / time.Minute) - (hoursAgo * 60)
	fmt.Printf("\nSTATS: %d received in the last %dh %dm\n\n", cmdGetStats(), hoursAgo, minutesAgo)
}

func loadLocalTimezone() {

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
			if err == nil {
				OurTimezone, _ = time.LoadLocation(info.Timezone)
				OurCountryCode = info.CountryCode
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
