BINARY := health
SRC    := ./cmd

.PHONY: build clean install uninstall

build:
	go build -o $(BINARY) $(SRC)

clean:
	rm -f $(BINARY)

install: build
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
	install -d /etc/openvpn-controller
	test -f /etc/openvpn-controller/env || install -m 600 env.example /etc/openvpn-controller/env
	install -m 644 cadc.ovpn /etc/openvpn-controller/cadc.ovpn
	install -m 644 openvpn-controller.service /etc/systemd/system/openvpn-controller.service
	systemctl daemon-reload

uninstall:
	systemctl stop openvpn-controller || true
	systemctl disable openvpn-controller || true
	rm -f /etc/systemd/system/openvpn-controller.service
	rm -f /usr/local/bin/$(BINARY)
	systemctl daemon-reload
