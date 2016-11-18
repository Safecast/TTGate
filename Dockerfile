FROM resin/raspberrypi-golang:1.7.3

# The app
ENV PKG ttgate

# Enable systemd
ENV INITSYSTEM on

# Install browser
#RUN apt-get update && apt-get upgrade && apt-get install -y ntpdate xorg midori matchbox unclutter screen

# Copy all the source code to the place where golang will find it
COPY . $GOPATH/src/$PKG

# Build all the golang source
WORKDIR $GOPATH/src/$PKG
#RUN go get -v google.golang.org/genproto/protobuf && go get -v && go install && go build all
RUN go get -v && go install && go build all

# Tell the container to run our shell script
CMD ["sh", "-c", "$GOPATH/src/$PKG/run.sh"]
