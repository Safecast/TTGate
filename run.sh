#!/bin/bash

##
## This first section is wifi-connect processing
##

# Look for the hard-wired network
GOTNET=0
for ((i=0; i<10; i++))
do
    if curl --output /dev/null --silent --head --fail http://www.google.com; then
        GOTNET=1
        break
    fi;
    sleep 1
done

# If no net, try app.js four times. It will try old credentials for 15 seconds for each iteration.
if [ "$GOTNET" -eq 0 ]; then
    export DBUS_SYSTEM_BUS_ADDRESS=unix:path=/host/run/dbus/system_bus_socket
    sleep 1
    for ((i=0; i<3; i++))
    do
        node src/app.js --clear=false
        if curl --output /dev/null --silent --head --fail http://www.google.com; then
            GOTNET=1
            break
        fi;
    done

fi

# If we still can't find the network, clear credentials and try again
if [ "$GOTNET" -eq 0 ]; then
    until $(curl --output /dev/null --silent --head --fail http://www.google.com); do
        node src/app.js --clear=true
    done
fi

##
## We've now got the network!
## We can now resume normal processing unrelated to Wifi-Connect
##

# If this resin.io env var is asserted, halt so we can play around in SSH
while [[ $HALT != "" ]]
do
      echo "HALT asserted to enable debugging..."
      sleep 60s
done

# Update the date/time NOW, so that it doesn't change dramatically
# during server operations
ntpdate time.nist.gov

# Run the kiosk browser in the background
screen -d -m -t browser sh $GOPATH/src/ttgate/run-browser.sh

# Run this in the foreground so we can watch the log
$GOPATH/bin/ttgate

# If we ever return HERE after restarting the device, restart the application
echo "Restarting app"
curl -X POST --header "Content-Type:application/json" --data '{"appId": '$RESIN_APP_ID'}' "$RESIN_SUPERVISOR_ADDRESS/v1/restart?apikey=$RESIN_SUPERVISOR_API_KEY"

# We won't ever get here, but FYI this is the code to restart the entire device.
echo "Restarting device"
curl -X POST --header "Content-Type:application/json" "$RESIN_SUPERVISOR_ADDRESS/v1/reboot?apikey=$RESIN_SUPERVISOR_API_KEY"
