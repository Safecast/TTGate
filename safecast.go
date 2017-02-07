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
    DeviceNo		   uint64    `json:"-"`
    capturedAt         string    `json:"-"`
    captured           time.Time `json:"-"`
    CapturedAtLocal    string    `json:"captured_local"`
    MinutesAgoStr      string    `json:"minutes_ago"`
    minutesAgo         int64     `json:"-"`
    minutesApproxAgo   int64     `json:"-"`
    Cpm0               string	 `json:"cpm0"`
    Cpm1               string	 `json:"cpm1"`
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
    if a[i].DeviceNo < a[j].DeviceNo {
        return true
    } else if a[i].DeviceNo > a[j].DeviceNo {
        return false
    }

    return false
}

// Record this safecast message for display on local HDMI via embedded browser
func cmdLocallyDisplaySafecastMessage(msg *teletype.Telecast, snr float32) {
    var dev SeenDevice

    // Bump stats
    totalMessagesReceived = totalMessagesReceived + 1

    // Exit if we can't display the value
    if  msg.DeviceID == nil {
        return
    }

    // Extract essential info to be recorded
    if msg.DeviceID != nil {
		dev.DeviceNo = uint64(msg.GetDeviceID())
        dev.DeviceID = strconv.FormatUint(dev.DeviceNo, 10)
    }

    if msg.CapturedAt != nil {
        dev.capturedAt = msg.GetCapturedAt()
    } else {
        dev.capturedAt = time.Now().Format(time.RFC3339)
    }
    dev.captured, _ = time.Parse(time.RFC3339, dev.capturedAt)
    dev.CapturedAtLocal = dev.captured.In(OurTimezone).Format("Mon 3:04pm")

    if msg.Cpm0 != nil {
        dev.Cpm0 = fmt.Sprintf("%dcpm", msg.GetCpm0())
	} else {
		dev.Cpm0 = ""
	}

    if msg.Cpm1 != nil {
        dev.Cpm1 = fmt.Sprintf("%dcpm", msg.GetCpm1())
	} else {
		dev.Cpm1 = ""
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

    // Scan and update the list of seen devices
    var found bool = false
    for i := 0; i < len(seenDevices); i++ {

        // Make sure we retain values that aren't present
        if dev.DeviceID == seenDevices[i].DeviceID {

            if dev.Cpm0 == "" {
                dev.Cpm0 = seenDevices[i].Cpm0
            }
            if dev.Cpm1 == "" {
                dev.Cpm1 = seenDevices[i].Cpm1
            }
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

			// Update the entry
            seenDevices[i] = dev
			found = true;
            break
        }

    }

    if !found {
        seenDevices = append(seenDevices, dev)
    }

    // Display the received message on the Resin device console
    fmt.Printf("\n%s %s: ", dev.CapturedAtLocal, dev.DeviceID)
    if dev.Cpm0 != "" && dev.Cpm1 == "" {
        fmt.Printf("%s\n\n", dev.Cpm0)
    } else if dev.Cpm0 == "" && dev.Cpm1 != "" {
        fmt.Printf("%s\n\n", dev.Cpm1)
    } else {
        fmt.Printf("%s %s\n\n", dev.Cpm0, dev.Cpm1)
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
