// Copyright 2017 Inca Roads LLC.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package main

type TTGateReq struct {
	Payload    []byte  `json:"payload"`
	Longitude  float32 `json:"longitude"`
	Latitude   float32 `json:"latitude"`
	Altitude   int32   `json:"altitude"`
	Snr        float32 `json:"snr"`
	Location   string  `json:"location"`
}
