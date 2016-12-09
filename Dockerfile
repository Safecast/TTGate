FROM resin/raspberrypi-golang

# The app
ENV PKG ttgate
ENV WIFI resin-wifi-connect-master

# Enable systemd
# ENV INITSYSTEM on

# Install node (for wifi-connect)
ENV NODE_VERSION 6.9.1

RUN curl -SLO "http://nodejs.org/dist/v$NODE_VERSION/node-v$NODE_VERSION-linux-armv6l.tar.gz" \
    && echo "0b30184fe98bd22b859db7f4cbaa56ecc04f7f526313c8da42315d89fabe23b2  node-v6.9.1-linux-armv6l.tar.gz" | sha256sum -c - \
        && tar -xzf "node-v$NODE_VERSION-linux-armv6l.tar.gz" -C /usr/local --strip-components=1 \
            && rm "node-v$NODE_VERSION-linux-armv6l.tar.gz" \
                && npm config set unsafe-perm true -g --unsafe-perm \
                    && rm -rf /tmp/*
                    
# Install wifi-connect, and browser
RUN apt-get update && apt-get upgrade \
	&& apt-get install -y dnsmasq hostapd iproute2 iw libdbus-1-dev libexpat-dev rfkill \
    && apt-get install -y ntpdate xorg midori matchbox unclutter screen \
    && rm -rf /var/lib/apt/lists/*

# Set up wifi-connect
RUN mkdir -p /usr/src/app/
WORKDIR /usr/src/app
COPY $WIFI/package.json /usr/src/app/

#RUN node -v
#RUN npm -v

RUN JOBS=MAX npm install --unsafe-perm --production && npm cache clean
COPY $WIFI/bower.json $WIFI/.bowerrc /usr/src/app/
RUN ./node_modules/.bin/bower --allow-root install && ./node_modules/.bin/bower --allow-root cache clean
COPY ./$WIFI /usr/src/app/
RUN ./node_modules/.bin/coffee -c ./src

# Copy and build all golang source code
COPY . $GOPATH/src/$PKG
WORKDIR $GOPATH/src/$PKG
RUN go get -v && go build && go install

# Tell the container to run our shell script
CMD ["sh", "-c", "$GOPATH/src/$PKG/run.sh"]
