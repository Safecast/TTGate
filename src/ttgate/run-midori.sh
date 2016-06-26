xset s off
xset -dpms
xset s noblank
sleep 15s
while [ : ]; do
    midori -e Fullscreen -a http://localhost:8080/
done
