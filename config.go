// Copyright 2017 Inca Roads LLC.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

// Telecast message handling
package main

// Service
var ttUploadAddress = "tt.safecast.org"
var ttUploadURLPattern = "http://%s/send"
var ttUploadIP = ""
var ttStatsURL = "http://tt.safecast.org/gateway"

// Timeouts
var restartWhenUnreachableMinutes = (60 * 2)
var restartEveryDays = 7

