---
kind: lesson
title: 'Signatures and SBOMs: trust, but verify with cosign'
description: |
  Reading with a hands-on core — what image signing actually proves, how
  Sigstore keyless signatures work, and what an SBOM buys you. Then do it
  for real: cosign-verify a publicly signed image and record the digest the
  signature vouches for.
name: sbom-cosign
slug: sbom-cosign
createdAt: "2026-07-13"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 300
    run: |
      set -euo pipefail
      # cosign is not preinstalled — pin and install it.
      COSIGN_VERSION=v3.1.1
      if ! command -v cosign >/dev/null 2>&1; then
        os=$(uname -s | tr '[:upper:]' '[:lower:]')
        arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; esac
        curl -fsSL "https://github.com/sigstore/cosign/releases/download/${COSIGN_VERSION}/cosign-${os}-${arch}" -o /tmp/cosign
        install /tmp/cosign /usr/local/bin/cosign
      fi
      mkdir -p /tmp/kubelings-cosign
      echo "cosign $(cosign version 2>/dev/null | grep GitVersion || true) ready; workdir /tmp/kubelings-cosign"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    timeout_seconds: 300
    run: |
      MARKER=/tmp/kubelings-cosign/verified-digest.txt
      if [ ! -s "$MARKER" ]; then
        echo "not yet: no digest recorded at $MARKER — cosign verify the image and save the digest the signature vouches for"; exit 1
      fi
      recorded=$(tr -d '[:space:]' < "$MARKER")
      case "$recorded" in
        sha256:*) : ;;
        *) echo "not yet: $MARKER should hold a bare sha256:… digest, got '$recorded'"; exit 1 ;;
      esac
      actual=$(cosign verify gcr.io/distroless/static:latest \
        --certificate-oidc-issuer https://accounts.google.com \
        --certificate-identity keyless@distroless.iam.gserviceaccount.com \
        -o json 2>/dev/null | grep -o '"docker-manifest-digest":"sha256:[a-f0-9]*"' | head -1 | grep -o 'sha256:[a-f0-9]*')
      if [ -z "$actual" ]; then
        echo "not yet: cosign verify failed against gcr.io/distroless/static:latest — network hiccup? re-run and check the flags"; exit 1
      fi
      if [ "$recorded" != "$actual" ]; then
        echo "not yet: recorded digest does not match what the signature vouches for right now ($actual) — re-run the verify and record its docker-manifest-digest"; exit 1
      fi
      echo "PASS — signature verified against the keyless identity, and the digest it vouches for is on record."
---
