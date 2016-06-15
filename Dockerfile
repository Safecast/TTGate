FROM resin/raspberrypi-golang

# The app
ENV PKG ttgate

# Enable systemd
ENV INITSYSTEM on

# Copy all the source code to the place where golang will find it
COPY ./src $GOPATH/src

# Build all the golang source
WORKDIR $GOPATH/src/$PKG
RUN go get && go install && go build all

# Tell the container to run the golang program's binary on startup
CMD ["sh", "-c", "env && $GOPATH/bin/$PKG"]
