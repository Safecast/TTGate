FROM resin/raspberrypi-golang

# The app
ENV PKG ttgate

# Enable systemd
ENV INITSYSTEM on

# Install browser
RUN apt-get update && apt-get upgrade && apt-get install -y ntpdate xorg midori matchbox unclutter screen

# Copy all the source code to the place where golang will find it
COPY . $GOPATH/src/$PKG

# Build all the golang source
WORKDIR $GOPATH/src/$PKG
RUN go get -v && go build && go install

# Tell the container to run our shell script
CMD ["sh", "-c", "$GOPATH/src/$PKG/run.sh"]
