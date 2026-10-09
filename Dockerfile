# openvpn-controller for k3s.
#
# Build:  IMPORT=1 ./build.sh
#
# Arch, not alpine or distroless, on purpose: OpenVPN 2.7 applies the pushed
# DNS servers through its dns-updown script, which calls resolvectl. The pod
# runs on the host network and talks to the host's systemd-resolved over the
# mounted D-Bus socket, so the work domain resolves on spiff exactly as it did
# when openvpn ran there directly. Arch's openvpn is the same build as the host.
FROM docker.io/library/golang:1.27-alpine@sha256:738d1cf061836894ff6bb8c33881080ac66de8cf0586615012a0c8f592649cfa AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/openvpn-controller ./cmd

FROM docker.io/library/archlinux:base@sha256:2fd1ae548076c67bcd41098ddd44e799058010e8d55f085d453dbb890e706860
RUN pacman -Syu --noconfirm --needed openvpn expect \
    && pacman -Scc --noconfirm \
    && rm -rf /var/cache/pacman/pkg/* /var/lib/pacman/sync/*

# dns-updown only picks resolved when /etc/resolv.conf is a symlink into
# systemd; in a pod kubelet bind-mounts a plain file there, so it would fall
# back to rewriting the container's resolv.conf and the work DNS would never
# reach the host. Test for resolved's varlink socket (mounted from the host)
# instead. grep -q first so a changed upstream script fails the build rather
# than silently skipping the patch.
RUN f=/usr/lib/openvpn/dns-updown \
    && grep -q 'readlink /etc/resolv.conf)" =~ systemd' "$f" \
    && sed -i 's|\[\[ "$(readlink /etc/resolv.conf)" =~ systemd \]\]|[[ -S /run/systemd/resolve/io.systemd.Resolve ]]|' "$f" \
    && grep -q 'S /run/systemd/resolve/io.systemd.Resolve' "$f"

WORKDIR /app
COPY --from=build --chmod=755 /out/openvpn-controller /app/openvpn-controller
# The controller runs ./expect.sh from its working directory; expect.sh runs
# openvpn against /etc/openvpn-controller/cadc.ovpn, mounted from a Secret.
COPY --chmod=755 expect.sh /app/expect.sh

ENV PLAIN_HTTP=true
EXPOSE 9172
ENTRYPOINT ["/app/openvpn-controller"]
