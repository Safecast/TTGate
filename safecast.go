// Handling of Safecast messages for local HDMI display
package main

import (
    "fmt"
    "sort"
    "time"
    "strconv"
    "encoding/json"
    "github.com/rayozzie/teletype-proto/golang"
)

// Data structure maintained for devices from which we received data
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
    BatteryCurrent     string    `json:"bat_current"`
    EnvTemp            string    `json:"env_temp"`
    EnvHumid           string    `json:"env_humid"`
    EnvPress           string    `json:"env_press"`
    SNR                string    `json:"snr"`
    snr                float32   `json:"-"`
    DeviceType         string    `json:"device_type"`
    Latitude           string    `json:"lat"`
    Longitude          string    `json:"lon"`
    Altitude           string    `json:"alt"`
    PmsPm01_0          string    `json:"pms_pm01_0"`
    PmsPm02_5          string    `json:"pms_pm02_5"`
    PmsPm10_0          string    `json:"pms_pm10_0"`
    OpcPm01_0          string    `json:"opc_pm01_0"`
    OpcPm02_5          string    `json:"opc_pm02_5"`
    OpcPm10_0          string    `json:"opc_pm10_0"`
}
var seenDevices []SeenDevice

// Class used to sort this data in a way that makes visual sense,
// trying to stabilize the first entry as what might be the "closest" one
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

// Record this safecast message for display on local HDMI via embedded browser
func cmdLocallyDisplaySafecastMessage(msg *teletype.Telecast, snr float32) {
    var dev SeenDevice
    var Value string
    var fNewStyleCPM bool

    // Bump stats
    totalMessagesReceived = totalMessagesReceived + 1

    // Exit if we can't display the value
    if msg.DeviceIDString == nil && msg.DeviceIDNumber == nil {
        return
    }

    // Extract essential info to be recorded
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

    if msg.Cpm0 != nil || msg.Cpm1 != nil {
        fNewStyleCPM = true
        if msg.Cpm0 == nil {
            dev.Value0 = ""
        } else {
            dev.Value0 = fmt.Sprintf("%dcpm", msg.GetCpm0())
        }
        if msg.Cpm1 == nil {
            dev.Value1 = ""
        } else {
            dev.Value1 = fmt.Sprintf("%dcpm", msg.GetCpm1())
        }
    } else {
        var Unit string
        fNewStyleCPM = false
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

    if msg.BatteryCurrent != nil {
        dev.BatteryCurrent = fmt.Sprintf("%.3fV", msg.GetBatteryCurrent())
    } else {
        dev.BatteryCurrent = ""
    }

    // Note that we make a valiant attempt to localize the temp to C or F
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

    if msg.EnvPressure != nil {
        dev.EnvPress = fmt.Sprintf("%.0f", msg.GetEnvPressure())
    } else {
        dev.EnvPress = ""
    }

    if snr != invalidSNR {
        dev.snr = snr
        iSNR := int32(snr)
        dev.SNR = fmt.Sprintf("%ddB", iSNR)
    } else {
        dev.snr = 0.0
        dev.SNR = ""
    }

    if msg.PmsPm01_0 != nil {
        dev.PmsPm01_0 = fmt.Sprintf("%dug/m3", msg.GetPmsPm01_0())
    } else {
        dev.PmsPm01_0 = ""
    }
    if msg.PmsPm02_5 != nil {
        dev.PmsPm02_5 = fmt.Sprintf("%dug/m3", msg.GetPmsPm02_5())
    } else {
        dev.PmsPm02_5 = ""
    }
    if msg.PmsPm10_0 != nil {
        dev.PmsPm10_0 = fmt.Sprintf("%dug/m3", msg.GetPmsPm10_0())
    } else {
        dev.PmsPm10_0 = ""
    }

    if msg.OpcPm01_0 != nil {
        dev.OpcPm01_0 = fmt.Sprintf("%fug/m3", msg.GetOpcPm01_0())
    } else {
        dev.OpcPm01_0 = ""
    }
    if msg.OpcPm02_5 != nil {
        dev.OpcPm02_5 = fmt.Sprintf("%fug/m3", msg.GetOpcPm02_5())
    } else {
        dev.OpcPm02_5 = ""
    }
    if msg.OpcPm10_0 != nil {
        dev.OpcPm10_0 = fmt.Sprintf("%fug/m3", msg.GetOpcPm10_0())
    } else {
        dev.OpcPm10_0 = ""
    }

    dev.DeviceType = msg.GetDeviceType().String()
    if msg.Latitude != nil {
        dev.Latitude = fmt.Sprintf("%.2f", msg.GetLatitude())
    } else {
        dev.Latitude = ""
    }
    if msg.Longitude != nil {
        dev.Longitude = fmt.Sprintf("%.2f", msg.GetLongitude())
    } else {
        dev.Longitude = ""
    }
    if msg.Altitude != nil {
        dev.Altitude = fmt.Sprintf("%dm", msg.GetAltitude())
    } else {
        dev.Altitude = ""
    }

    // Add or update the seen entry, as the case may be.
    // Note that we handle the case of 2 geiger units in a single device by always folding both together
    dev.originalDeviceNo = 0
    dev.normalizedDeviceNo = dev.originalDeviceNo
    deviceno, err := strconv.ParseInt(dev.DeviceID, 10, 64)
    if err == nil {
        dev.originalDeviceNo = uint64(deviceno)
        dev.normalizedDeviceNo = dev.originalDeviceNo
        if (false) { // old style device id support removed 2017-02
            if (dev.originalDeviceNo & 0x01) != 0 {
                dev.normalizedDeviceNo = uint64(dev.normalizedDeviceNo - 1)
                dev.DeviceID = fmt.Sprintf("%d", dev.normalizedDeviceNo)
            }
        }
    }

    // Scan and update the list of seen devices
    var found bool = false
    for i := 0; i < len(seenDevices); i++ {

        // Handle non-numeric device ID
        if dev.originalDeviceNo == 0 && dev.DeviceID == seenDevices[i].DeviceID {
            if (!fNewStyleCPM) {
                dev.Value1 = ""
                if (Value != "") {
                    dev.Value0 = Value
                } else {
                    dev.Value0 = seenDevices[i].Value0
                }
            }
            found = true
        }

        // For numerics, folder the even/odd devices into a single device (dual-geigers)
        if dev.originalDeviceNo != 0 && dev.normalizedDeviceNo == seenDevices[i].normalizedDeviceNo {
            if (!fNewStyleCPM) {
                if (false) { // old style device id support removed 2017-02
                    if (dev.originalDeviceNo & 0x01) == 0 {
                        if (Value != "") {
                            dev.Value0 = Value
                        } else {
                            dev.Value0 = seenDevices[i].Value0
                        }
                        dev.Value1 = seenDevices[i].Value1
                    } else {
                        dev.Value0 = seenDevices[i].Value0
                        if (Value != "") {
                            dev.Value1 = Value
                        } else {
                            dev.Value1 = seenDevices[i].Value1;
                        }
                    }
                } else {
                    if (Value != "") {
                        dev.Value0 = Value
                    } else {
                        dev.Value0 = seenDevices[i].Value0
                    }
                    dev.Value1 = seenDevices[i].Value1
                }
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
            if dev.BatteryCurrent == "" {
                dev.BatteryCurrent = seenDevices[i].BatteryCurrent
            }
            if dev.EnvTemp == "" {
                dev.EnvTemp = seenDevices[i].EnvTemp
            }
            if dev.EnvHumid == "" {
                dev.EnvHumid = seenDevices[i].EnvHumid
            }
            if dev.EnvPress == "" {
                dev.EnvPress = seenDevices[i].EnvPress
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
        if (!fNewStyleCPM) {
            if dev.originalDeviceNo == 0 {
                dev.Value0 = Value
                dev.Value1 = ""
            } else {
                if (false) { // old style device number support removed 2017-02
                    if (dev.originalDeviceNo & 0x01) == 0 {
                        dev.Value0 = Value
                        dev.Value1 = ""
                    } else {
                        dev.Value0 = ""
                        dev.Value1 = Value
                    }
                } else {
                    dev.Value0 = Value
                    dev.Value1 = ""
                }
            }
        }
        seenDevices = append(seenDevices, dev)

    }

    // Display the received message on the Resin device console
    fmt.Printf("\n%s %s: ", dev.CapturedAtLocal, dev.DeviceID)
    if dev.Value0 != "" && dev.Value1 == "" {
        fmt.Printf("%s\n\n", dev.Value0)
    } else if dev.Value0 == "" && dev.Value1 != "" {
        fmt.Printf("%s\n\n", dev.Value1)
    } else {
        fmt.Printf("%s %s\n\n", dev.Value0, dev.Value1)
    }

}

// Get the device data sorted and classified in a way useful in local web browser
func GetSafecastDataAsJSON() []byte {

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

    // Reformat it into a JSON text buffer
    buffer, _ := json.MarshalIndent(sortedDevices, "", "    ")

    // Return that buffer to the caller
    return (buffer)

}
