#!/usr/bin/env python3

import time

print("Starting openvpn (fake) ...")
otp = input("CHALLENGE: One Time Password ")
print("Aha!", otp)

while True:
    print("tick")
    time.sleep(10)
