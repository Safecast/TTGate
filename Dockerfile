FROM resin/raspberrypi-golang

# What's the name of your go package?
ENV PKG test

# Enable systemd
ENV INITSYSTEM on

# Copy all the source code to the place where golang will find it
COPY ./src $GOPATH/src

# Build all the golang source
WORKDIR $GOPATH/src
RUN go get && go install && go build all

# Tell the container to run the golang program's binary on startup
CMD ["sh", "-c", "env && $GOPATH/bin/$PKG"]
