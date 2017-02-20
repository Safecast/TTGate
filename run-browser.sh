# Copyright 2017 Inca Roads LLC.  All rights reserved.
# Use of this source code is governed by licenses granted by the
# copyright holder including that found in the LICENSE file.

# Sleep to avoid any startup race conditions that manifested themselves
# because the gateway hadn't yet started its web server
sleep 60s
# Run midori under x11
startx $GOPATH/src/ttgate/run-midori.sh
