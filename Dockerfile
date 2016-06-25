FROM resin/raspberrypi-golang

# The app
ENV PKG ttgate

# Enable systemd
ENV INITSYSTEM on

# Install apt deps
RUN apt-get update && apt-get install -y \
 iceweasel \
 apt-utils \
 xserver-xorg-core \
 xserver-xorg-input-all \
 xserver-xorg-video-fbdev \
 xorg \
 build-essential \
 clang \
 libdbus-1-dev \
 libgtk2.0-dev \
 libnotify-dev \
 libgnome-keyring-dev \
 libgconf2-dev \
 libasound2-dev \
 libcap-dev \
 libcups2-dev \
 libxtst-dev \
 libxss1 \
 libnss3-dev \
 fluxbox \
 libsmbclient \
 libssh-4 \
 python-dev \
 python-pip \
 build-essential \
 git \
 curl \
 psmisc \
 libraspberrypi0 \
 libpcre3 \
 fonts-freefont-ttf \
 fbset \
 bind9 \
 libdbus-1-dev \
 libexpat-dev \
 usbutils \
 && rm -rf /var/lib/apt/lists/*

# Set Xorg and FLUXBOX preferences
RUN mkdir ~/.fluxbox
RUN echo "xset s off\nxserver-command=X -s 0 dpms" > ~/.fluxbox/startup
RUN echo "#!/bin/sh\n\nexec /usr/bin/X -s 0 dpms -nocursor -nolisten tcp "$@"" > /etc/X11/xinit/xserverrc

# Copy all the source code to the place where golang will find it
COPY ./src $GOPATH/src

# Build all the golang source
WORKDIR $GOPATH/src/$PKG
RUN go get && go install && go build all

# Tell the container to run the golang program's binary on startup
CMD ["sh", "-c", "env && $GOPATH/bin/$PKG"]
