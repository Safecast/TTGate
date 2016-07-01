FROM resin/raspberrypi-golang

# The app
ENV PKG ttgate

# Enable systemd
ENV INITSYSTEM on

# Install browser
RUN apt-get update && apt-get upgrade && apt-get install -y xorg midori matchbox unclutter screen

# Copy all the source code to the place where golang will find it
COPY ./src $GOPATH/src/$PKG

# Build all the golang source
WORKDIR $GOPATH/src/$PKG
RUN go get && go install && go build all

# Tell the container to run our shell script
CMD ["sh", "-c", "$GOPATH/src/$PKG/run.sh"]
