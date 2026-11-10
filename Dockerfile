# openvpn-controller for k3s.
#
# Build:  IMPORT=1 ./build.sh
#
# Arch, not alpine or distroless, on purpose: OpenVPN 2.7 applies the pushed
# DNS servers through its dns-updown script, which calls resolvectl. The pod
# runs on the host network and talks to the host's systemd-resolved over the
# mounted D-Bus socket, so the work domain resolves on spiff exactly as it did
# when openvpn ran there directly. Arch's openvpn is the same build as the host.
FROM docker.io/library/golang:1.27-alpine@sha256:eb61c3873c97416fff0116553fff65d67a3549f72e11d9758849a689cdd8b523 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/openvpn-controller ./cmd

FROM docker.io/library/archlinux:base@sha256:fcee73ea1b96e236248f53a213f71276cc9072d9025b58bce9915e32f19963ae
RUN pacman -Syu --noconfirm --needed openvpn expect \
    && pacman -Scc --noconfirm \
    && rm -rf /var/cache/pacman/pkg/* /var/lib/pacman/sync/*

# libcap's captree is a Go binary Arch builds with an old toolchain, so it
# carries ~23 stdlib CVEs into the image scan. Nothing here runs it.
RUN rm -f /usr/bin/captree

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
COPY --chmod=755 dns-split-updown /app/dns-split-updown

EXPOSE 9172
ENTRYPOINT ["/app/openvpn-controller"]
