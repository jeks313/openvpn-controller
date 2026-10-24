#!/usr/bin/expect -f

set timeout -1
# dns-split-updown keeps the tunnel's DNS to the work domains (see that file).
spawn openvpn --config /etc/openvpn-controller/cadc.ovpn --script-security 2 --dns-updown /app/dns-split-updown
interact
# Reap openvpn before exiting. Without this it is orphaned to PID 1 (the
# controller, which never reaps strangers) and every session leaves a
# <defunct> openvpn behind. Pass its exit status up as ours.
set code [lindex [wait] 3]
if {![string is integer -strict $code]} { set code 1 }
exit $code
