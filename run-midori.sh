# Copyright 2017 Inca Roads LLC.  All rights reserved.
# Use of this source code is governed by licenses granted by the
# copyright holder including that found in the LICENSE file.

while [ : ]; do
    # disable DPMS (Energy Star) features
    xset -dpms
    # disable screen saver
    xset s off
    # don't blank the video device
    xset s noblank
    # remove cursor, run truly full-screen and not just the top left quarter, and run midori browser in fullscreen mode
    unclutter &
    matchbox-window-manager &
    midori -e Fullscreen -a http://localhost:8080/
done
