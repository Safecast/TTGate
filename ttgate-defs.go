package main

// DataUpAppReq represents the actual payloads sent to application on uplink
type TTGateReq struct {
	Payload    []byte  `json:"payload"`
	Longitude  float32 `json:"longitude"`
	Latitude   float32 `json:"latitude"`
	Altitude   int32   `json:"altitude"`
	Snr        float32 `json:"snr"`
	Location   string  `json:"location"`
}
