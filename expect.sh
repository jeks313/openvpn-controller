#!/usr/bin/expect -f

set timeout -1
spawn openvpn /etc/openvpn-controller/cadc.ovpn
interact