#! /bin/bash

# If this resin.io env var is asserted, halt so we can play around in SSH
if [[ $HALT != "" ]]; then
    echo "*** HALT asserted - exiting ***"
    exit 1
fi

# Run the kiosk browser in the background
screen -d -m -t browser sh $GOPATH/src/ttgate/run-browser.sh

# Run this in the foreground so we can watch the log
$GOPATH/bin/ttgate


