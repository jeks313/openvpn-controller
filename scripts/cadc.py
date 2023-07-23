#!/usr/bin/env python3

import os
import sys
import pty
import subprocess

OVPN = "cadc.ovpn"
CHALLENGE = b'CHALLENGE: One Time Password '
OVPN_DIR = "/etc/openvpn/client"

code = os.environ['TOTP']

os.chdir(OVPN_DIR)

cp, fd = pty.fork()
if cp == 0:
    os.execlp("openvpn", "openvpn", OVPN)
else:
    f = os.fdopen(fd)
    challenge = False
    while not challenge:
        l = os.read(fd, len(CHALLENGE))
        challenge = (l == CHALLENGE)
        if not challenge:
            rest = f.readline()

    print("writing TOTP code!")
    os.write(fd, bytes("%s\n" % code, "utf8"))
    while True:
        try:
            print(f.readline().strip())
        except KeyboardInterrupt:
            os.kill(cp, 15)
        except IOError:
            print("Bye")
            sys.exit(0)
