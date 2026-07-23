---
kind: lesson
title: 'pending CSRs: approve a node into the cluster'
description: |
  A kubelet's serving-certificate request is sitting Pending — and unlike client
  certs, kubelet-serving CSRs are never auto-approved, so nothing issues it until
  you do. This is the manual-approval step operators hit on real clusters, tied
  to the same TLS-bootstrap machinery behind node joins and cert rotation. You'll
  read the pending CSR, verify what it's asking for, approve it, and confirm the
  certificate was issued.
name: kubelet-csr-approval
slug: kubelet-csr-approval
createdAt: "2026-07-23"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      set -euo pipefail
      KC=/etc/kubernetes/admin.conf
      NAME=kubelings-node-csr
      D=$(mktemp -d)

      # Clean any leftover from a prior attempt.
      kubectl --kubeconfig="$KC" delete csr "$NAME" >/dev/null 2>&1 || true

      # Forge exactly the CSR a kubelet submits for its SERVING cert: subject
      # O=system:nodes, CN=system:node:<name>, plus a SAN for the node. The
      # kubelet-serving signer is NOT auto-approved (unlike client certs), so it
      # sits Pending until a human approves it — the real-world reason these
      # land in your lap.
      openssl genrsa -out "$D/k.key" 2048 >/dev/null 2>&1
      openssl req -new -key "$D/k.key" \
        -subj "/O=system:nodes/CN=system:node:node-bootstrap" \
        -addext "subjectAltName=DNS:node-bootstrap,IP:10.0.0.99" \
        -out "$D/k.csr" >/dev/null 2>&1
      REQ=$(base64 -w0 "$D/k.csr")

      cat <<EOF | kubectl --kubeconfig="$KC" apply -f - >/dev/null
      apiVersion: certificates.k8s.io/v1
      kind: CertificateSigningRequest
      metadata:
        name: $NAME
      spec:
        request: $REQ
        signerName: kubernetes.io/kubelet-serving
        usages:
        - digital signature
        - key encipherment
        - server auth
      EOF
      rm -rf "$D"

      echo "A node's kubelet SERVING-cert CSR is Pending: $NAME"
      echo
      echo "On cplane-01:"
      echo "    kubectl get csr"
      echo "    kubectl get csr $NAME -o yaml     # inspect what it's requesting"
      echo
      echo "Approve it, then confirm a certificate was issued:"
      echo "    kubectl certificate approve $NAME"
      echo "    kubectl get csr $NAME"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    timeout_seconds: 120
    run: |
      KC=/etc/kubernetes/admin.conf
      NAME=kubelings-node-csr

      approved="$(kubectl --kubeconfig="$KC" get csr "$NAME" \
        -o jsonpath='{.status.conditions[?(@.type=="Approved")].status}' 2>/dev/null || true)"
      if [ -z "$approved" ]; then
        if ! kubectl --kubeconfig="$KC" get csr "$NAME" >/dev/null 2>&1; then
          echo "not yet: CSR '$NAME' is gone — re-run init for this lesson."
        else
          echo "not yet: '$NAME' is still Pending — approve it:"
          echo "         kubectl certificate approve $NAME"
        fi
        exit 1
      fi

      # Approval alone isn't the finish line: the signer must have ISSUED a cert.
      cert="$(kubectl --kubeconfig="$KC" get csr "$NAME" \
        -o jsonpath='{.status.certificate}' 2>/dev/null || true)"
      if [ -z "$cert" ]; then
        echo "not yet: '$NAME' is Approved but no certificate has been issued yet."
        echo "         Give kube-controller-manager's signer a moment; re-check"
        echo "         'kubectl get csr $NAME' (CONDITION should read Approved,Issued)."
        exit 1
      fi

      echo "PASS — '$NAME' is Approved and a client certificate was issued."
      echo
      kubectl --kubeconfig="$KC" get csr "$NAME" 2>/dev/null || true
---
