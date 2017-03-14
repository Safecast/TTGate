// Copyright 2017 Inca Roads LLC.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

// Handling of Safecast messages for local HDMI display
package main

import (
    "fmt"
    "sort"
    "time"
    "strconv"
    "encoding/json"
    "github.com/safecast/ttproto/golang"
)

// Data structure maintained for devices from which we received data
type SeenDevice struct {
    DeviceId           string    `json:"device_id"`
    DeviceNo		   uint64    `json:"-"`
    capturedAt         string    `json:"-"`
    captured           time.Time `json:"-"`
    CapturedAtLocal    string    `json:"captured_local"`
    MinutesAgoStr      string    `json:"minutes_ago"`
    minutesAgo         int64     `json:"-"`
    minutesApproxAgo   int64     `json:"-"`
    Lnd_7318U          string	 `json:"lnd7318u"`
    Lnd_7318C          string	 `json:"lnd7318c"`
    Lnd_7128Ec         string	 `json:"lnd7128ec"`
    BatVoltage		   string    `json:"bat_voltage"`
    BatSoc			   string    `json:"bat_soc"`
    BatCurrent         string    `json:"bat_current"`
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
func cmdLocallyDisplaySafecastMessage(msg ttproto.Telecast, snr float32) {
    var dev SeenDevice

    // Bump stats
    totalMessagesReceived = totalMessagesReceived + 1

    // Exit if we can't display the value
    if  msg.DeviceId == nil {
        return
    }

    // Extract essential info to be recorded
    if msg.DeviceId != nil {
		dev.DeviceNo = uint64(msg.GetDeviceId())
        dev.DeviceId = strconv.FormatUint(dev.DeviceNo, 10)
    }

    if msg.CapturedAt != nil {
        dev.capturedAt = msg.GetCapturedAt()
    } else {
        dev.capturedAt = time.Now().Format(time.RFC3339)
    }
    dev.captured, _ = time.Parse(time.RFC3339, dev.capturedAt)
    dev.CapturedAtLocal = dev.captured.In(OurTimezone).Format("Mon 3:04pm")

    if msg.Lnd_7318U != nil {
        dev.Lnd_7318U = fmt.Sprintf("%dcpm", msg.GetLnd_7318U())
	} else {
		dev.Lnd_7318U = ""
	}

    if msg.Lnd_7318C != nil {
        dev.Lnd_7318C = fmt.Sprintf("%dcpm", msg.GetLnd_7318C())
	} else {
		dev.Lnd_7318C = ""
	}

    if msg.Lnd_7128Ec != nil {
        dev.Lnd_7128Ec = fmt.Sprintf("%dcpm", msg.GetLnd_7128Ec())
	} else {
		dev.Lnd_7128Ec = ""
	}

    if msg.BatSoc != nil {
        dev.BatSoc = fmt.Sprintf("%.2f%%", msg.GetBatSoc())
    } else {
        dev.BatSoc = ""
    }

    if msg.BatVoltage != nil {
        dev.BatVoltage = fmt.Sprintf("%.2fV", msg.GetBatVoltage())
    } else {
        dev.BatVoltage = ""
    }

    if msg.BatCurrent != nil {
        dev.BatCurrent = fmt.Sprintf("%.3fV", msg.GetBatCurrent())
    } else {
        dev.BatCurrent = ""
    }

    // Note that we make a valiant attempt to localize the temp to C or F
    if msg.EnvTemp != nil {
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
            dev.EnvTemp = fmt.Sprintf("%.1fF", ((msg.GetEnvTemp()*9.0)/5.0)+32)
        default:
            dev.EnvTemp = fmt.Sprintf("%.1fC", msg.GetEnvTemp())
        }
    } else {
        dev.EnvTemp = ""
    }

    if msg.EnvHumid != nil {
        dev.EnvHumid = fmt.Sprintf("%.1f%%", msg.GetEnvHumid())
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
        if dev.DeviceId == seenDevices[i].DeviceId {

            if dev.Lnd_7318U == "" {
                dev.Lnd_7318U = seenDevices[i].Lnd_7318U
            }
            if dev.Lnd_7318C == "" {
                dev.Lnd_7318C = seenDevices[i].Lnd_7318C
            }
            if dev.Lnd_7128Ec == "" {
                dev.Lnd_7128Ec = seenDevices[i].Lnd_7128Ec
            }
            if dev.BatVoltage == "" {
                dev.BatVoltage = seenDevices[i].BatVoltage
            }
            if dev.BatSoc == "" {
                dev.BatSoc = seenDevices[i].BatSoc
            }
            if dev.BatCurrent == "" {
                dev.BatCurrent = seenDevices[i].BatCurrent
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
	str1 := "-"
	str2 := "-"
	str3 := "-"
	if dev.Lnd_7318U != "" {
		str1 = dev.Lnd_7318U
	}
	if dev.Lnd_7318C != "" {
		str2 = dev.Lnd_7318C
	}
	if dev.Lnd_7128Ec != "" {
		str3 = dev.Lnd_7128Ec
	}
    fmt.Printf("\n%s %s: %s %s %s\n\n", dev.CapturedAtLocal, dev.DeviceId, str1, str2, str3)

}

// Get the device data sorted and classified in a way useful in local web browser
func GetSafecastDevicesString() string {

    // Duplicate the device list
    sortedDevices := seenDevices

    // Zip through the list, updating how many minutes it was captured ago
	s := ""
    for i := 0; i < len(sortedDevices); i++ {
		if s != "" {
			s += ","
		}
		s += fmt.Sprintf("%d", sortedDevices[i].DeviceNo)
	}

	return s
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
