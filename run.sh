#! /bin/bash

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

# If we ever return, restart the application and reboot the device
curl -X POST --header "Content-Type:application/json" --data '{"appId": <appId>}' "$RESIN_SUPERVISOR_ADDRESS/v1/restart?apikey=$RESIN_SUPERVISOR_API_KEY"
curl -X POST --header "Content-Type:application/json" "$RESIN_SUPERVISOR_ADDRESS/v1/reboot?apikey=$RESIN_SUPERVISOR_API_KEY"

