/*
 * Teletype Command Processing
 *
 * This module contains the command state machine
 */

package main

import (
    "os"
    "fmt"
    "strconv"
    "bytes"
    "time"
    "encoding/json"
    "net/http"
    "github.com/golang/protobuf/proto"
    "github.com/rayozzie/teletype-proto/golang"
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

var cmdInitialized = false;
var receivedMessage = false;
var gotSNR bool = false
var inReinit bool = false;
var SNR float64
var outboundQueue chan OutboundCommand
var currentState uint16
var totalMessagesReceived int = 0
var totalMessagesSent int = 0
var busyCount = 0
var watchdog1mCount = 0

func cmdWatchdog1m() {

    // Exit if we're not yet initialized

    if (!cmdInitialized || inReinit) {
        return
    }

    // Ignore the first increment, which could occur at any time 0-59s.
    // But then, on the second increment, reset the world.

    watchdog1mCount = watchdog1mCount + 1

    switch (watchdog1mCount) {
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
    if (busyCount > 10) {
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

    inReinit = true;

    // Reinitialize the Microchip in case it's wedged.

    if rebootLPWAN {
        ioInitMicrochip()
    }

    // Init statics

    gotSNR = false
    receivedMessage = false

    // Initialize the state machine and kick off a device reset

    cmdSetState(CMD_STATE_LPWAN_RESETREQ);
    cmdProcess(nil)

    // Done

    inReinit = false

}

func cmdInit() {

    // Initialize the outbound queue

    outboundQueue = make(chan OutboundCommand, 100)         // Don't exhibit backpressure for a long time

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

    if (cmd == nil) {

        cmd = []byte("")

    } else {

        // This is a special, necessary delay because we DO get called here from
        // the inbound task even in the middle of initialization, and we're simply
        // not prepared to deal with it yet.

        for (!cmdInitialized || inReinit) {
            time.Sleep(1 * time.Second)
        }

    }

    cmdstr = string(cmd)

    fmt.Printf("cmdProcess(%s) entry state=%v\n", cmdstr, currentState)

    switch currentState {

    case CMD_STATE_LPWAN_RESETREQ:
        time.Sleep(4 * time.Second)
        ioSendCommandString("sys get ver")
        cmdSetState(CMD_STATE_LPWAN_GETVERRPL)

    case CMD_STATE_LPWAN_GETVERRPL:
        time.Sleep(4 * time.Second)
        if ((!bytes.HasPrefix(cmd, []byte("RN2483"))) && (!bytes.HasPrefix(cmd, []byte("RN2903")))) {
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
        if ((bytes.HasPrefix(cmd, []byte("RN2483"))) || (bytes.HasPrefix(cmd, []byte("RN2903")))) {
            cmdSetState(CMD_STATE_LPWAN_MACPAUSERPL)
        } else {
            i64, err := strconv.ParseInt(cmdstr, 10, 64)
            if (err != nil || i64 < 100000) {
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
            if (!SentPendingOutbound()) {
                {
                    // Update the SNR stat if and only if we've received a message
                    if (receivedMessage && !gotSNR) {
                        ioSendCommandString("radio get snr")
                        cmdSetState(CMD_STATE_LPWAN_SNRRPL)
                    } else {
                        RestartReceive()
                    }
                }
            }
        } else if bytes.HasPrefix(cmd, []byte("busy")) {
            // This is not at all expected, but it means that we're
            // moving too quickly and we should try again.
            time.Sleep(5 * time.Second)
            RestartReceive()
            // reset the world if too many consecutive busy errors
            cmdBusy()
        } else if bytes.HasPrefix(cmd, []byte("radio_rx")) {
            // skip whitespace (there is more than one space)
            var hexstarts int
            for hexstarts = len("radio_rx"); hexstarts<len(cmd); hexstarts++ {
                if (cmd[hexstarts] > ' ') {
                    break
                }
            }
            // Remember that we received at least one message
            receivedMessage = true
            gotSNR = false
            // Parse and process the received message
            cmdProcessReceived(cmd[hexstarts:])
            // if there's a pending outbound, transmit it (which will change state)
            // else restart the receive
            if (!SentPendingOutbound()) {
                RestartReceive()
            }
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
            f, err := strconv.ParseFloat(cmdstr, 64)
            if (err == nil) {
                SNR = f
                gotSNR = true
            }
            // Always restart receive
            RestartReceive()
        }

    case CMD_STATE_LPWAN_TXRPL1:
        if bytes.HasPrefix(cmd, []byte("ok")) {
            cmdSetState(CMD_STATE_LPWAN_TXRPL2);
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
            if (!SentPendingOutbound()) {
                RestartReceive()
            }
        } else {
            fmt.Printf("LPWAN xmt2 error\n")
            RestartReceive()
        }

    }

    fmt.Printf("cmdProcess exit state=%v\n", currentState)

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

    if (len(outboundQueue) != 0) {

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

func cmdProcessReceived(hex []byte) {

    // Convert received message from hex to binary
    bin := make([]byte, len(hex)/2)
    for i := 0; i < len(hex)/2; i++ {

        var hinibble, lonibble byte
        hinibblechar := hex[2*i]
        lonibblechar := hex[(2*i)+1]

        if (hinibblechar >= '0' && hinibblechar <= '9') {
            hinibble = hinibblechar - '0'
        } else if (hinibblechar >= 'A' && hinibblechar <= 'F') {
            hinibble = (hinibblechar - 'A') + 10
        } else if (hinibblechar >= 'a' && hinibblechar <= 'f') {
            hinibble = (hinibblechar - 'a') + 10
        } else {
            hinibble = 0
        }

        if (lonibblechar >= '0' && lonibblechar <= '9') {
            lonibble = lonibblechar - '0'
        } else if (lonibblechar >= 'A' && lonibblechar <= 'F') {
            lonibble = (lonibblechar - 'A') + 10
        } else if (lonibblechar >= 'a' && lonibblechar <= 'f') {
            lonibble = (lonibblechar - 'a') + 10
        } else {
            lonibble = 0
        }

        bin[i] = (hinibble << 4) | lonibble

    }

    // Process the received protocol buffer

    cmdProcessReceivedProtobuf(bin)

}

func cmdProcessReceivedProtobuf(buf []byte) {

    // Unmarshal the buffer into a golang object

    msg := &teletype.Telecast{}
    err := proto.Unmarshal(buf, msg)
    if err != nil {
        fmt.Printf("cmdProcessReceivedProtobuf unmarshaling error: ", err);
        return
    }

    // Do various things baed upon the message type

    switch msg.GetDeviceType() {

        // Is it something we recognize as being from safecast?
    case teletype.Telecast_BGEIGIE_NANO:
        fallthrough
    case teletype.Telecast_SIMPLECAST:
        cmdProcessReceivedSafecastMessage(msg)

        // Display what we got from a non-Safecast device
    default:
        if (msg.DeviceIDString != nil) {
            fmt.Printf("Received Msg from Device %s: '%s'\n", msg.GetDeviceIDString(), msg.GetMessage())
        }
        if (msg.DeviceIDNumber != nil) {
            fmt.Printf("Received Msg from Device %d: '%s'\n", msg.GetDeviceIDNumber(), msg.GetMessage())
        }

    }

}

func cmdProcessReceivedSafecastMessage(msg *teletype.Telecast) {

    // Bump stats

    totalMessagesReceived = totalMessagesReceived+1

    // Debug

    fmt.Printf("Received Safecast Message:\n")
    fmt.Printf("%s\n", msg)

    // Combine the info with what we can find in the environment vars
    // Note that we support both unqualified and DeviceID-qualified variables,
    // both for convenience and in case a gateway could potentially pick up multiple
    // source devices.

    var rawDeviceID, DeviceID, CapturedAt, Unit, Value, Altitude, Latitude, Longitude, BatteryVoltage, BatterySOC string
    var hasBatteryVoltage, hasBatterySOC bool

    if (msg.DeviceIDString != nil) {
        rawDeviceID = msg.GetDeviceIDString();
    } else if (msg.DeviceIDNumber != nil) {
        rawDeviceID = strconv.FormatUint(uint64(msg.GetDeviceIDNumber()), 10);
    } else {
        rawDeviceID = "UNKNOWN";
    }
    prefix := rawDeviceID + "_"
    DeviceID = os.Getenv(prefix + "ID")
    if (DeviceID == "") {
        DeviceID = os.Getenv("ID")
    }
    if DeviceID == "" {
        DeviceID = rawDeviceID;
    }

    if msg.CapturedAt != nil {
        CapturedAt = msg.GetCapturedAt()
    } else {
        CapturedAt = time.Now().Format(time.RFC3339)
    }

    if (msg.Unit == nil) {
        Unit = "cpm"
    } else {
        Unit = fmt.Sprintf("%s", msg.GetUnit())
    }

    if msg.Value == nil {
        fmt.Printf("*** error: (Value) is required\n")
        return
    }
    Value = fmt.Sprintf("%d", msg.GetValue())

    if msg.Latitude != nil {
        Latitude = fmt.Sprintf("%f", msg.GetLatitude())
    } else {
        Latitude = os.Getenv(prefix + "LAT")
        if (Latitude == "") {
            Latitude = os.Getenv("LAT")
        }
        if Latitude == "" {
            fmt.Printf("*** error: env var LAT (or %sLAT) required\n", prefix)
            return;
        }
    }

    if msg.Longitude != nil {
        Longitude = fmt.Sprintf("%f", msg.GetLongitude())
    } else {
        Longitude = os.Getenv(prefix + "LON")
        if (Longitude == "") {
            Longitude = os.Getenv("LON")
        }
        if Longitude == "" {
            fmt.Printf("*** error: env var LON (or %sLON) required\n", prefix)
            return;
        }
    }

    if msg.Altitude != nil {
        Altitude = fmt.Sprintf("%d", msg.GetAltitude())
    } else {
        Altitude = os.Getenv(prefix + "ALT")
        if (Altitude == "") {
            Altitude = os.Getenv("ALT")
        }
        if Altitude == "" {
            Altitude = "0"
        }
    }

    if msg.BatterySOC != nil {
        BatterySOC = fmt.Sprintf("%.2f", msg.GetBatterySOC())
        hasBatterySOC = true
    } else {
        hasBatterySOC = false;
    }

    if msg.BatteryVoltage != nil {
        BatteryVoltage = fmt.Sprintf("%.4f", msg.GetBatteryVoltage())
        hasBatteryVoltage = true
    } else {
        hasBatteryVoltage = false;
    }

    // Get upload parameters

    URL := os.Getenv(prefix + "URL")
    if (URL == "") {
        URL = os.Getenv("URL")
    }
    if URL == "" {
        URL = "http://107.161.164.163/scripts/indextest.php?api_key=%s"
    }

    KEY := os.Getenv(prefix + "KEY")
    if (KEY == "") {
        KEY = os.Getenv("KEY")
    }
    if KEY == "" {
        KEY = "z3sHhgousVDDrCVXhzMT"
    }
    UploadURL := fmt.Sprintf(URL, KEY)

    // Upload it to safecast

    type SafecastData struct {
        CapturedAt   string `json:"captured_at,omitempty"`   // 2016-02-20T14:02:25Z
        ChannelID    string `json:"channel_id,omitempty"`    // nil
        DeviceID     string `json:"device_id,omitempty"`     // 140
        DeviceTypeID string `json:"devicetype_id,omitempty"` // nil
        Height       string `json:"height,omitempty"`        // 123
        ID           string `json:"id,omitempty"`            // 972298
        LocationName string `json:"location_name,omitempty"` // nil
        OriginalID   string `json:"original_id,omitempty"`   // 972298
        SensorID     string `json:"sensor_id,omitempty"`     // nil
        StationID    string `json:"station_id,omitempty"`    // nil
        Unit         string `json:"unit,omitempty"`          // cpm
        UserID       string `json:"user_id,omitempty"`       // 304
        Value        string `json:"value,omitempty"`         // 36
        Latitude     string `json:"latitude,omitempty"`      // 37.0105
        Longitude    string `json:"longitude,omitempty"`     // 140.9253
        BatVoltage   string `json:"bat_voltage,omitempty"`   // 0-N volts
        BatSOC       string `json:"bat_soc,omitempty"`       // 0%-100%
        WirelessSNR  string `json:"wireless_snr,omitempty"`  // -127db to +127db
    }

    // We upload 3 records to the safecast service; here's the stuff in common to all
    sc := &SafecastData{}
    sc.DeviceID = rawDeviceID
    sc.CapturedAt = CapturedAt
    sc.Latitude = Latitude
    sc.Longitude = Longitude
    sc.Height = Altitude

    // The first upload has everything
    sc1 := sc
    sc1.Unit = Unit
    sc1.Value = Value
    if (hasBatteryVoltage) {
        sc1.BatVoltage = BatteryVoltage
    }
    if (hasBatterySOC) {
        sc1.BatSOC = BatterySOC
    }
    if (gotSNR) {
        fstr := fmt.Sprintf("%.1f", SNR)
        sc1.WirelessSNR = fstr
    }

    scJSON, _ := json.Marshal(sc1)
    fmt.Printf("About to upload to %s:\n%s\n", UploadURL, scJSON)
    req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(scJSON))
    req.Header.Set("User-Agent", "TTGATE")
    req.Header.Set("Content-Type", "application/json")
    httpclient := &http.Client{}
    resp, err := httpclient.Do(req)
    if err != nil {
        fmt.Printf("*** Error uploading to Safecast %s\n\n", err)
    } else {
        resp.Body.Close()
        // Bump stats
        totalMessagesSent = totalMessagesSent+1
        fmt.Printf("Success!\n")
    }

    // The next upload has battery voltage
    if (hasBatteryVoltage) {
        // Prepare the data
        sc2 := sc
        sc2.Unit = "bat_voltage"
        sc2.Value = sc1.BatVoltage
        // Do the upload
        scJSON, _ = json.Marshal(sc2)
        req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(scJSON))
        req.Header.Set("User-Agent", "TTGATE")
        req.Header.Set("Content-Type", "application/json")
        httpclient = &http.Client{}
        resp, err = httpclient.Do(req)
        if err != nil {
            fmt.Printf("*** Error uploading bat_voltage to Safecast %s\n\n", err)
        } else {
            resp.Body.Close()
        }
    }

    // The next upload has battery SOC
    if (hasBatterySOC) {
        // Prepare the data
        sc3 := sc
        sc3.Unit = "bat_soc"
        sc3.Value = sc1.BatSOC
        // Do the upload
        scJSON, _ = json.Marshal(sc3)
        req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(scJSON))
        req.Header.Set("User-Agent", "TTGATE")
        req.Header.Set("Content-Type", "application/json")
        httpclient = &http.Client{}
        resp, err = httpclient.Do(req)
        if err != nil {
            fmt.Printf("*** Error uploading bat_voltage to Safecast %s\n\n", err)
        } else {
            resp.Body.Close()
        }
    }

    // The next upload has SNR
    if (gotSNR) {
        // Prepare the data
        sc4 := sc
        sc4.Unit = "wireless_snr"
        sc4.Value = sc1.WirelessSNR
        // Do the upload
        scJSON, _ = json.Marshal(sc4)
        req, err := http.NewRequest("POST", UploadURL, bytes.NewBuffer(scJSON))
        req.Header.Set("User-Agent", "TTGATE")
        req.Header.Set("Content-Type", "application/json")
        httpclient = &http.Client{}
        resp, err = httpclient.Do(req)
        if err != nil {
            fmt.Printf("*** Error uploading SNR to Safecast %s\n\n", err)
        } else {
            resp.Body.Close()
        }
    }

}

// eof
