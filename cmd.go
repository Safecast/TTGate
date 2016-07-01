/*
 * Teletype Command Processing
 *
 * This module contains the command state machine
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/rayozzie/teletype-proto/golang"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

// States

const (
	CMD_STATE_IDLE = iota
	CMD_STATE_LPWAN_RESETREQ
	CMD_STATE_LPWAN_RESETRPL
	CMD_STATE_LPWAN_GETVERRPL
	CMD_STATE_LPWAN_MACPAUSERPL
	CMD_STATE_LPWAN_SETWDTRPL
	CMD_STATE_LPWAN_RCVRPL
	CMD_STATE_LPWAN_TXRPL1
	CMD_STATE_LPWAN_TXRPL2
	CMD_STATE_LPWAN_SNRRPL
)

type OutboundCommand struct {
	Command []byte
}

type SeenDevice struct {
	DeviceID           string    `json:"device_id"`
	originalDeviceNo   uint64    `json:"-"`
	normalizedDeviceNo uint64    `json:"-"`
	capturedAt         string    `json:"-"`
	captured           time.Time `json:"-"`
	CapturedAtLocal    string    `json:"captured_local"`
	MinutesAgoStr      string    `json:"minutes_ago"`
	minutesAgo         int64     `json:"-"`
	minutesApproxAgo   int64     `json:"-"`
	Value0             string    `json:"value0"`
	Value1             string    `json:"value1"`
	BatteryVoltage     string    `json:"bat_voltage"`
	BatterySOC         string    `json:"bat_soc"`
	EnvTemp            string    `json:"env_temp"`
	EnvHumid           string    `json:"env_humid"`
	SNR                string    `json:"snr"`
	snr                float32   `json:"-"`
	DeviceType         string    `json:"device_type"`
	Latitude           string    `json:"lat"`
	Longitude          string    `json:"lon"`
	Altitude           string    `json:"alt"`
}

type ByKey []SeenDevice

func (a ByKey) Len() int      { return len(a) }
func (a ByKey) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool {

	// Primary:
	// Treat things captured reasonably coincident  as all being equivalent
	if a[i].minutesApproxAgo < a[j].minutesApproxAgo {
		return true
	} else if a[i].minutesApproxAgo > a[j].minutesApproxAgo {
		return false
	}

	// Secondary:
	// Treat things with higher SNR as being more significant than things with lower SNR
	if a[i].SNR != "" && a[j].SNR != "" {
		if a[i].snr > a[j].snr {
			return true
		} else if a[i].snr < a[j].snr {
			return false
		}
	}

	// Tertiary:
	// In an attempt to keep things reasonably deterministic, use device number
	if a[i].normalizedDeviceNo < a[j].normalizedDeviceNo {
		return true
	} else if a[i].normalizedDeviceNo > a[j].normalizedDeviceNo {
		return false
	}

	return false
}

var seenDevices []SeenDevice
var cmdInitialized bool = false
var receivedMessage []byte
var inReinit bool = false
var invalidSNR float32 = 123.456
var outboundQueue chan OutboundCommand
var currentState uint16
var totalMessagesReceived int = 0
var totalMessagesSent int = 0
var busyCount = 0
var watchdog1mCount = 0
var ipinfo string = ""

func cmdWatchdog1m() {

	// Exit if we're not yet initialized

	if !cmdInitialized || inReinit {
		return
	}

	// Ignore the first increment, which could occur at any time 0-59s.
	// But then, on the second increment, reset the world.

	watchdog1mCount = watchdog1mCount + 1

	switch watchdog1mCount {
	case 1:
	case 2:
		fmt.Printf("*** cmdWatchdog: Warning!\n")
	case 3:
		fmt.Printf("*** cmdWatchdog: Reinitializing!\n")
		cmdReinit(true)
	}

}

func cmdBusy() {

	// Ignore the first increment, which could occur at any time 0-59s.
	// But then, on the second increment, reset the world.

	busyCount = busyCount + 1
	if busyCount > 10 {
		cmdReinit(true)
	}

}

func cmdWatchdogReset() {
	watchdog1mCount = 0
}

func cmdBusyReset() {
	busyCount = 0
}

func cmdGetStats() (received int, sent int) {
	return totalMessagesReceived, totalMessagesSent
}

func cmdReinit(rebootLPWAN bool) {

	// Prevent recursion because we call this from multiple goroutines

	if inReinit {
		fmt.Printf("cmdReinit: [[[[[ Aborting nested init ]]]]]\n")
		return
	}

	inReinit = true

	// Reinitialize the Microchip in case it's wedged.

	if rebootLPWAN {
		ioInitMicrochip()
	}

	// Initialize the state machine and kick off a device reset

	cmdSetState(CMD_STATE_LPWAN_RESETREQ)
	cmdProcess(nil)

	// Done

	inReinit = false

}

func cmdInit() {

	// Initialize the outbound queue

	outboundQueue = make(chan OutboundCommand, 100) // Don't exhibit backpressure for a long time

	// Init state machine, etc.

	cmdReinit(true)

	// We're now fully initialized

	cmdInitialized = true

}

func cmdEnqueueOutbound(cmd []byte) {
	var ocmd OutboundCommand
	ocmd.Command = cmd
	outboundQueue <- ocmd
}

func cmdSetState(newState uint16) {
	currentState = newState
	cmdWatchdogReset()
}

func cmdProcess(cmd []byte) {
	var cmdstr string

	// Special case of the very first init, which is called from outside the inbound task

	if cmd == nil {

		cmd = []byte("")

	} else {

		// This is a special, necessary delay because we DO get called here from
		// the inbound task even in the middle of initialization, and we're simply
		// not prepared to deal with it yet.

		for !cmdInitialized || inReinit {
			time.Sleep(1 * time.Second)
		}

	}

	cmdstr = string(cmd)

	fmt.Printf("recv(%s)\n", cmdstr)

	switch currentState {

	case CMD_STATE_LPWAN_RESETREQ:
		time.Sleep(4 * time.Second)
		ioSendCommandString("sys get ver")
		cmdSetState(CMD_STATE_LPWAN_GETVERRPL)

	case CMD_STATE_LPWAN_GETVERRPL:
		time.Sleep(4 * time.Second)
		if (!bytes.HasPrefix(cmd, []byte("RN2483"))) && (!bytes.HasPrefix(cmd, []byte("RN2903"))) {
			ioSendCommandString("sys get ver")
			cmdSetState(CMD_STATE_LPWAN_GETVERRPL)
		} else {
			ioSendCommandString("sys reset")
			cmdSetState(CMD_STATE_LPWAN_RESETRPL)
		}

	case CMD_STATE_LPWAN_RESETRPL:
		time.Sleep(4 * time.Second)
		ioSendCommandString("mac pause")
		cmdSetState(CMD_STATE_LPWAN_MACPAUSERPL)

	case CMD_STATE_LPWAN_MACPAUSERPL:
		time.Sleep(4 * time.Second)
		// If we're still getting these responses, it's because we're still
		// flushing the buffer of incoming sys get ver's or sys resets from
		// previous commands.  In this case, do NOT issue new commands
		// because we'll just aggravate the situation.  Just flush,
		// and keep waiting for the expected command.
		if (bytes.HasPrefix(cmd, []byte("RN2483"))) || (bytes.HasPrefix(cmd, []byte("RN2903"))) {
			cmdSetState(CMD_STATE_LPWAN_MACPAUSERPL)
		} else {
			i64, err := strconv.ParseInt(cmdstr, 10, 64)
			if err != nil || i64 < 100000 {
				fmt.Printf("Bad response from mac pause: %s\n", cmdstr)
			} else {
				ioSendCommandString("radio set wdt 60000")
				cmdSetState(CMD_STATE_LPWAN_SETWDTRPL)
			}
		}

	case CMD_STATE_LPWAN_SETWDTRPL:
		time.Sleep(4 * time.Second)
		RestartReceive()

	case CMD_STATE_LPWAN_RCVRPL:
		if bytes.HasPrefix(cmd, []byte("ok")) {
			// this is expected response from initiating the rcv,
			// so just ignore it and keep waiting for a message to come in
		} else if bytes.HasPrefix(cmd, []byte("radio_err")) {
			// Expected from receive timeout of WDT seconds.
			// if there's a pending outbound, transmit it (which will change state)
			// else restart the receive
			if !SentPendingOutbound() {
				RestartReceive()
			}
		} else if bytes.HasPrefix(cmd, []byte("busy")) {
			// This is not at all expected, but it means that we're
			// moving too quickly and we should try again.
			time.Sleep(5 * time.Second)
			RestartReceive()
			// reset the world if too many consecutive busy errors
			cmdBusy()
		} else if bytes.HasPrefix(cmd, []byte("radio_rx")) {
			// skip whitespace, then remember the message that we received,
			// because we'll need it after we get the SNR of the transmission
			var hexstarts int
			for hexstarts = len("radio_rx"); hexstarts < len(cmd); hexstarts++ {
				if cmd[hexstarts] > ' ' {
					break
				}
			}
			receivedMessage = cmd[hexstarts:]
			// Get the SNR of the last message received
			ioSendCommandString("radio get snr")
			cmdSetState(CMD_STATE_LPWAN_SNRRPL)
		} else {
			// Totally unknown error, but since we cannot just
			// leave things in a state without a pending receive,
			// we need to just restart the world.
			fmt.Printf("LPWAN rcv error\n")
			cmdReinit(true)
		}

	case CMD_STATE_LPWAN_SNRRPL:
		{
			// Get the number in the commanbd buffer
			snr64, err := strconv.ParseFloat(cmdstr, 64)
			if err != nil {
				snr64 = float64(invalidSNR)
			}
			// Parse and process the received message
			cmdProcessReceived(receivedMessage, float32(snr64))
			// If there's a pending outbound, transmit it (which will change state)
			// else restart the receive
			if !SentPendingOutbound() {
				RestartReceive()
			}
		}

	case CMD_STATE_LPWAN_TXRPL1:
		if bytes.HasPrefix(cmd, []byte("ok")) {
			cmdSetState(CMD_STATE_LPWAN_TXRPL2)
		} else if bytes.HasPrefix(cmd, []byte("busy")) {
			// This is not at all expected, but it means that we're
			// moving too quickly and we should try again.
			time.Sleep(5 * time.Second)
			RestartReceive()
			// reset the world if too many consecutive busy errors
			cmdBusy()
		} else {
			fmt.Printf("LPWAN xmt1 error\n")
			RestartReceive()
		}

	case CMD_STATE_LPWAN_TXRPL2:
		if bytes.HasPrefix(cmd, []byte("radio_tx_ok")) {
			// if there's another pending outbound, transmit it, else restart the receive
			if !SentPendingOutbound() {
				RestartReceive()
			}
		} else {
			fmt.Printf("LPWAN xmt2 error\n")
			RestartReceive()
		}

	}

}

func RestartReceive() {
	ioSendCommandString("radio rx 0")
	cmdBusyReset()
	cmdSetState(CMD_STATE_LPWAN_RCVRPL)
}

func SentPendingOutbound() bool {
	hexchar := []byte("0123456789ABCDEF")

	// We test this because we can never afford to block here,
	// and we knkow that we're the only consumer of this queue

	if len(outboundQueue) != 0 {

		for ocmd := range outboundQueue {
			outbuf := []byte("radio tx ")
			for _, databyte := range ocmd.Command {
				loChar := hexchar[(databyte & 0x0f)]
				hiChar := hexchar[((databyte >> 4) & 0x0f)]
				outbuf = append(outbuf, hiChar)
				outbuf = append(outbuf, loChar)
			}
			ioSendCommand(outbuf)
			cmdBusyReset()
			cmdSetState(CMD_STATE_LPWAN_TXRPL1)
			return true
		}

	}
	return false
}

func cmdProcessReceived(hex []byte, snr float32) {

	// Convert received message from hex to binary
	bin := make([]byte, len(hex)/2)
	for i := 0; i < len(hex)/2; i++ {

		var hinibble, lonibble byte
		hinibblechar := hex[2*i]
		lonibblechar := hex[(2*i)+1]

		if hinibblechar >= '0' && hinibblechar <= '9' {
			hinibble = hinibblechar - '0'
		} else if hinibblechar >= 'A' && hinibblechar <= 'F' {
			hinibble = (hinibblechar - 'A') + 10
		} else if hinibblechar >= 'a' && hinibblechar <= 'f' {
			hinibble = (hinibblechar - 'a') + 10
		} else {
			hinibble = 0
		}

		if lonibblechar >= '0' && lonibblechar <= '9' {
			lonibble = lonibblechar - '0'
		} else if lonibblechar >= 'A' && lonibblechar <= 'F' {
			lonibble = (lonibblechar - 'A') + 10
		} else if lonibblechar >= 'a' && lonibblechar <= 'f' {
			lonibble = (lonibblechar - 'a') + 10
		} else {
			lonibble = 0
		}

		bin[i] = (hinibble << 4) | lonibble

	}

	// Process the received protocol buffer

	cmdProcessReceivedProtobuf(bin, snr)

}

func cmdProcessReceivedProtobuf(buf []byte, snr float32) {

	msg := &teletype.Telecast{}
	err := proto.Unmarshal(buf, msg)
	if err != nil {
		fmt.Printf("cmdProcessReceivedProtobuf unmarshaling error: ", err)
		return
	}

	cmdProcessReceivedTelecastMessage(msg, buf, snr)

}

func cmdProcessReceivedTelecastMessage(msg *teletype.Telecast, pb []byte, snr float32) {

	// Do various things baed upon the message type

	switch msg.GetDeviceType() {

	// processing a simplecast message?
	case teletype.Telecast_SIMPLECAST:
		cmdProcessReceivedSafecastMessage(msg, snr)

		// Forwarding a message from a nano?
	case teletype.Telecast_BGEIGIE_NANO:
		cmdProcessReceivedSafecastMessage(msg, snr)

		// If this is a ping request (indicated by null Message), then send that device back the same thing we received,
		// but WITH a message (so that we don't cause a ping storm among multiple ttgates with visibility to each other)
	case teletype.Telecast_TTGATE:
		if msg.Message == nil {
			msg.Message = proto.String("ping")
			data, err := proto.Marshal(msg)
			if err != nil {
				fmt.Printf("marshaling error: ", err)
			}
			// Importantly, sleep for a couple seconds to give the (slow) receiver a chance to get into receive mode
			time.Sleep(2 * time.Second)
			cmdEnqueueOutbound(data)
			fmt.Printf("Sent pingback to device %d\n", msg.GetDeviceIDNumber())
			return
		}

		// Display what we got from a non-Safecast device
	default:
		if msg.DeviceIDString != nil {
			fmt.Printf("Received Msg from Device %s: '%s'\n", msg.GetDeviceIDString(), msg.GetMessage())
		}
		if msg.DeviceIDNumber != nil {
			fmt.Printf("Received Msg from Device %d: '%s'\n", msg.GetDeviceIDNumber(), msg.GetMessage())
		}

	}

	// Forward the message to the service [and delete the stuff from processreceivedsafecastmessage!]
	cmdForwardMessageToTeletypeService(pb, snr)

}

func cmdForwardMessageToTeletypeService(pb []byte, snr float32) {

	// TTSERVE url

	UploadURL := "http://api.teletype.io:8080/send"

	// The first time through here, let's fetch our IPINFO

	if ipinfo == "" {
		response, err := http.Get("http://ip-api.com/json/")
		if err == nil {
			defer response.Body.Close()
			contents, err := ioutil.ReadAll(response.Body)
			if err == nil {
				ipinfo = string(contents)
			}
		}
	}

	// Use the same data structure as TTN, because we're simulating TTN inbound

	msg := &DataUpAppReq{}
	msg.Metadata = make([]AppMetadata, 1)
	msg.Payload = pb

	// Some devices don't have LAT/LON, and in this case the gateway must supply it

	Latitude := os.Getenv("LAT")
	if Latitude != "" {
		f64, err := strconv.ParseFloat(Latitude, 64)
		if err == nil {
			msg.Metadata[0].Latitude = float32(f64)
		}
	}
	Longitude := os.Getenv("LON")
	if Longitude != "" {
		f64, err := strconv.ParseFloat(Longitude, 64)
		if err == nil {
			msg.Metadata[0].Longitude = float32(f64)
		}
	}
	Altitude := os.Getenv("ALT")
	if Altitude != "" {
		i64, err := strconv.ParseInt(Altitude, 10, 64)
		if err == nil {
			msg.Metadata[0].Altitude = int32(i64)
		}
	}

	// The service might find it handy to see the SNR of the last message received from the gateway

	if snr != invalidSNR {
		msg.Metadata[0].Lsnr = float32(snr)
	}

	// Augment with ip info

	msg.Metadata[0].GatewayEUI = ipinfo

	// Send it

	msgJSON, _ := json.Marshal(msg)
	req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(msgJSON))
	req.Header.Set("User-Agent", "TTGATE")
	req.Header.Set("Content-Type", "application/json")
	httpclient := &http.Client{}
	resp, err := httpclient.Do(req)
	if err != nil {
		fmt.Printf("*** Error uploading to TTSERVE %s\n\n", err)
	} else {
		resp.Body.Close()
	}

}

func cmdProcessReceivedSafecastMessage(msg *teletype.Telecast, snr float32) {
	var dev SeenDevice
	var Unit string
	var Value string

	// Bump stats

	totalMessagesReceived = totalMessagesReceived + 1

	// Exit if we can't display the value

	if msg.DeviceIDString == nil && msg.DeviceIDNumber == nil {
		return
	}

	if msg.DeviceIDString != nil {
		dev.DeviceID = msg.GetDeviceIDString()
	}
	if msg.DeviceIDNumber != nil {
		dev.DeviceID = strconv.FormatUint(uint64(msg.GetDeviceIDNumber()), 10)
	}

	if msg.CapturedAt != nil {
		dev.capturedAt = msg.GetCapturedAt()
	} else {
		dev.capturedAt = time.Now().Format(time.RFC3339)
	}
	dev.captured, _ = time.Parse(time.RFC3339, dev.capturedAt)
	dev.CapturedAtLocal = dev.captured.In(OurTimezone).Format("Mon 3:04pm")

	if msg.Unit == nil {
		Unit = "cpm"
	} else {
		Unit = fmt.Sprintf("%s", msg.GetUnit())
	}

	if msg.Value == nil {
		Value = ""
	} else {
		Value = fmt.Sprintf("%d%s", msg.GetValue(), Unit)
	}

	if msg.BatterySOC != nil {
		dev.BatterySOC = fmt.Sprintf("%.2f%%", msg.GetBatterySOC())
	} else {
		dev.BatterySOC = ""
	}

	if msg.BatteryVoltage != nil {
		dev.BatteryVoltage = fmt.Sprintf("%.2fV", msg.GetBatteryVoltage())
	} else {
		dev.BatteryVoltage = ""
	}

	if msg.EnvTemperature != nil {
		switch OurCountryCode {
		case "BS": // Bahamas
			fallthrough
		case "BZ": // Belize
			fallthrough
		case "GU": // Guam
			fallthrough
		case "KY": // Cayman Islands
			fallthrough
		case "PW": // Palau
			fallthrough
		case "PR": // Puerto Rico
			fallthrough
		case "US": // United States
			fallthrough
		case "VI": // US Virgin Islands
			dev.EnvTemp = fmt.Sprintf("%.1fF", ((msg.GetEnvTemperature()*9.0)/5.0)+32)
		default:
			dev.EnvTemp = fmt.Sprintf("%.1fC", msg.GetEnvTemperature())
		}
	} else {
		dev.EnvTemp = ""
	}

	if msg.EnvHumidity != nil {
		dev.EnvHumid = fmt.Sprintf("%.1f%%", msg.GetEnvHumidity())
	} else {
		dev.EnvHumid = ""
	}

	if snr != invalidSNR {
		dev.snr = snr
		iSNR := int32(snr)
		dev.SNR = fmt.Sprintf("%ddB", iSNR)
	} else {
		dev.snr = 0.0
		dev.SNR = ""
	}

	dev.DeviceType = msg.GetDeviceType().String()
	if msg.Latitude != nil {
		dev.Latitude = fmt.Sprintf("%f", msg.GetLatitude())
	} else {
		dev.Latitude = ""
	}
	if msg.Longitude != nil {
		dev.Longitude = fmt.Sprintf("%f", msg.GetLongitude())
	} else {
		dev.Longitude = ""
	}
	if msg.Altitude != nil {
		dev.Altitude = fmt.Sprintf("%dm", msg.GetAltitude())
	} else {
		dev.Altitude = ""
	}

	// Add or update the seen entry, as the case may be.
	// Note that we handle the case of 2 geiger units in a single device by always folding both together via device ID mask

	dev.originalDeviceNo = 0
	dev.normalizedDeviceNo = dev.originalDeviceNo
	deviceno, err := strconv.ParseInt(dev.DeviceID, 10, 64)
	if err == nil {
		dev.originalDeviceNo = uint64(deviceno)
		dev.normalizedDeviceNo = dev.originalDeviceNo
		if (dev.originalDeviceNo & 0x01) != 0 {
			dev.normalizedDeviceNo = uint64(dev.normalizedDeviceNo - 1)
			dev.DeviceID = fmt.Sprintf("%d", dev.normalizedDeviceNo)
		}
	}

	var found bool = false
	for i := 0; i < len(seenDevices); i++ {

		// Handle non-numeric device ID
		if dev.originalDeviceNo == 0 && dev.DeviceID == seenDevices[i].DeviceID {
			dev.Value0 = Value
			dev.Value1 = ""
			found = true
		}

		// For numerics, folder the even/odd devices into a single device (dual-geigers)
		if dev.originalDeviceNo != 0 && dev.normalizedDeviceNo == seenDevices[i].normalizedDeviceNo {
			if (dev.originalDeviceNo & 0x01) == 0 {
				dev.Value0 = Value
				dev.Value1 = seenDevices[i].Value1
			} else {
				dev.Value0 = seenDevices[i].Value0
				dev.Value1 = Value
			}
			found = true
		}

		// Retain values for those items that are only transmitted occasionaly
		if found {
			if dev.BatteryVoltage == "" {
				dev.BatteryVoltage = seenDevices[i].BatteryVoltage
			}
			if dev.BatterySOC == "" {
				dev.BatterySOC = seenDevices[i].BatterySOC
			}
			if dev.EnvTemp == "" {
				dev.EnvTemp = seenDevices[i].EnvTemp
			}
			if dev.EnvHumid == "" {
				dev.EnvHumid = seenDevices[i].EnvHumid
			}
			if dev.SNR == "" {
				dev.snr = seenDevices[i].snr
				dev.SNR = seenDevices[i].SNR
			}
			seenDevices[i] = dev
			break
		}

	}

	if !found {
		if dev.originalDeviceNo == 0 {
			dev.Value0 = Value
			dev.Value1 = ""
		} else {
			if (dev.originalDeviceNo & 0x01) == 0 {
				dev.Value0 = Value
				dev.Value1 = ""
			} else {
				dev.Value0 = ""
				dev.Value1 = Value
			}
		}
		seenDevices = append(seenDevices, dev)

	}

	// Display them

	UpdateDisplay()

}

func GetSortedDeviceList() []SeenDevice {

	// Duplicate the device list
	sortedDevices := seenDevices

	// Zip through the list, updating how many minutes it was captured ago
	t := time.Now()
	for i := 0; i < len(sortedDevices); i++ {
		sortedDevices[i].minutesAgo = int64(t.Sub(sortedDevices[i].captured) / time.Minute)
		sortedDevices[i].MinutesAgoStr = fmt.Sprintf("%dm", sortedDevices[i].minutesAgo)
		sortedDevices[i].minutesApproxAgo = int64(t.Sub(sortedDevices[i].captured) / (time.Duration(15) * time.Minute))
	}

	// Sort it
	sort.Sort(ByKey(sortedDevices))

	return (sortedDevices)

}

func UpdateDisplay() {

	sorted := GetSortedDeviceList()

	found := false
	for i := 0; i < len(sorted); i++ {
		s := sorted[i]
		if (i == 0) {
			found = true
			fmt.Printf("\n")
		}
		fmt.Printf("%s %s: ", s.CapturedAtLocal, s.DeviceID)
		if s.Value0 != "" && s.Value1 == "" {
			fmt.Printf("%s\n", s.Value0)
		} else if s.Value0 == "" && s.Value1 != "" {
			fmt.Printf("%s\n", s.Value1)
		} else {
			fmt.Printf("%s %s\n", s.Value0, s.Value1)
		}

	}

	if (found) {
		fmt.Printf("\n")
	}
	
}

// eof
