# Copyright 2017 Inca Roads LLC.  All rights reserved.
# Use of this source code is governed by licenses granted by the
# copyright holder including that found in the LICENSE file.

FROM resin/raspberrypi3-golang

# Packaging parameters
ENV PKG ttgate
ENV WIFI resin-wifi-connect-master
ENV WIFIDIR /usr/src/app

# Install node (exclusively for wifi-connect)
ENV NODE_VERSION 6.9.1
RUN curl -SLO "http://nodejs.org/dist/v$NODE_VERSION/node-v$NODE_VERSION-linux-armv6l.tar.gz" \
    && echo "0b30184fe98bd22b859db7f4cbaa56ecc04f7f526313c8da42315d89fabe23b2  node-v6.9.1-linux-armv6l.tar.gz" | sha256sum -c - \
        && tar -xzf "node-v$NODE_VERSION-linux-armv6l.tar.gz" -C /usr/local --strip-components=1 \
            && rm "node-v$NODE_VERSION-linux-armv6l.tar.gz" \
                && npm config set unsafe-perm true -g --unsafe-perm \
                    && rm -rf /tmp/*
                    
# Install wifi-connect, and install midori browser
RUN apt-get update && apt-get upgrade \
	&& apt-get install -y dnsmasq hostapd iproute2 iw libdbus-1-dev libexpat-dev rfkill \
    && apt-get install -y ntpdate xorg midori matchbox unclutter screen \
    && rm -rf /var/lib/apt/lists/*

# Set up wifi-connect
RUN mkdir -p $WIFIDIR/
WORKDIR $WIFIDIR
COPY $WIFI/package.json $WIFIDIR/
RUN JOBS=MAX npm install --unsafe-perm --production && npm cache clean
COPY $WIFI/bower.json $WIFI/.bowerrc $WIFIDIR/
RUN ./node_modules/.bin/bower --allow-root install && ./node_modules/.bin/bower --allow-root cache clean
COPY ./$WIFI $WIFIDIR/
RUN ./node_modules/.bin/coffee -c ./src

# Set up our app
COPY . $GOPATH/src/$PKG
WORKDIR $GOPATH/src/$PKG
RUN go get -v && go build && go install

# Set current directory back to wifi-connect (because that's the shell script's assumption), and go go go
WORKDIR $WIFIDIR
CMD ["sh", "-c", "$GOPATH/src/$PKG/run.sh"]
