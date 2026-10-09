#!/usr/bin/env bash
# Build the openvpn-controller image with podman.
#
#   ./build.sh              # -> ghcr.io/jeks313/openvpn-controller:latest
#   TAG=test ./build.sh     # tag :test instead
#   IMPORT=1 ./build.sh     # also load it straight into k3s containerd,
#                            # which is what lets the cluster run it without
#                            # the image ever leaving this box
set -euo pipefail
cd "$(dirname "$0")"

IMAGE="${IMAGE:-ghcr.io/jeks313/openvpn-controller}"
TAG="${TAG:-latest}"
IMPORT="${IMPORT:-0}"

# Provenance. The revision label says which commit this image is, so fleet can
# tell when what's running is behind main; the <date>-<sha> tag is the rollback
# target once :latest has moved on. A dirty tree is marked, not hidden.
REV="$(git rev-parse HEAD)"
DIRTY=""; [[ -z "$(git status --porcelain)" ]] || DIRTY="-dirty"
VTAG="${VTAG:-$(date -u +%Y%m%d)-$(git rev-parse --short=7 HEAD)${DIRTY}}"
LABELS=(--label "org.opencontainers.image.revision=${REV}${DIRTY}"
        --label "org.opencontainers.image.source=https://github.com/jeks313/openvpn-controller")

echo "Building ${IMAGE}:${TAG}"
podman build "${LABELS[@]}" -t "${IMAGE}:${TAG}" -t "${IMAGE}:${VTAG}" -f Dockerfile .

if [[ $IMPORT == 1 ]]; then
    echo "==> importing into k3s containerd (k8s.io namespace)"
    podman save "${IMAGE}:${TAG}" | sudo k3s ctr --namespace k8s.io images import -
    sudo k3s ctr --namespace k8s.io images tag --force "${IMAGE}:${TAG}" "${IMAGE}:${VTAG}"
    sudo k3s ctr --namespace k8s.io images ls -q | grep -F "${IMAGE}:"
fi

echo
echo "Built ${IMAGE}:${TAG} and ${IMAGE}:${VTAG} (revision ${REV:0:12}${DIRTY})"
