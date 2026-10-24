#!/usr/bin/expect -f

set timeout -1
# dns-split-updown keeps the tunnel's DNS to the work domains (see that file).
spawn openvpn --config /etc/openvpn-controller/cadc.ovpn --script-security 2 --dns-updown /app/dns-split-updown
interact