---
kind: unit
title: "pending CSRs: the approval that lets a kubelet talk"
name: kubelet-csr-approval-unit
---


> **☁ iximiuz Labs only.** This is a control-plane operation on real cluster
> PKI — reading and approving a CertificateSigningRequest that a kubelet's
> bootstrap flow submits. You do it on a control plane you own, with cluster
> admin, against the live signer.

## How a node earns the right to talk

A kubelet doesn't start life with a client certificate. When a node joins
(11's kubeadm story, and 7.10), it has only a short-lived **bootstrap token** —
just enough credential to do one thing: submit a **CertificateSigningRequest**
asking for a real client cert. The control plane's signer issues that cert,
the kubelet writes it to disk, and from then on it authenticates as
`system:node:<name>`. This is **TLS bootstrapping**, and the same machinery
drives kubelet **cert rotation** (7.12): before a cert expires, the kubelet
submits a fresh CSR for a renewal.

Between "submitted" and "issued" sits one gate: **approval**. A CSR names a
signer and a set of usages; the matching signer only issues a certificate once
the request is *approved*. For the kubelet **client** cert that approval is
usually automatic — the cluster auto-approves node bootstrap/rotation requests
of the right shape. But kubelet **serving** certs
(`kubernetes.io/kubelet-serving`) are a deliberate exception: **they are never
auto-approved by default**, so every one waits Pending for a human. That's the
CSR that lands in your lap — along with anything else that doesn't match an
auto-approval rule. A Pending CSR is a node stuck at the door.

## Reading a CSR before you sign it

Approving a CSR mints a trusted identity — so read what you're signing:

```sh
kubectl get csr                       # NAME, SIGNERNAME, REQUESTOR, CONDITION
kubectl get csr <name> -o yaml        # the full request
```

Three fields decide what a signed cert can do:

- **`signerName`** — which signer handles it, and therefore what kind of cert
  it is. `kubernetes.io/kube-apiserver-client-kubelet` = a kubelet *client*
  cert (authenticates the kubelet to the API server).
  `kubernetes.io/kubelet-serving` = the kubelet's *serving* cert (for the API
  server calling back to the kubelet). `kubernetes.io/legacy-unknown` should
  make you suspicious.
- **The subject** — decoded from the request, it's `O=system:nodes,
  CN=system:node:<name>`. The `O` becomes the cert's group, the `CN` its user.
  Approving this grants exactly the `system:node:<name>` identity — check the
  name is a node you expect.
- **`usages`** — `client auth` for a client cert, `server auth` for a serving
  cert. Usages that don't match the signer won't be issued.

Only after that do you approve:

```sh
kubectl certificate approve <name>
# or, to refuse:
kubectl certificate deny <name>
```

Approval flips the CSR's `Approved` condition; the signer then issues the cert
and fills in `status.certificate`. `kubectl get csr` shows the condition go
from `Pending` to `Approved,Issued`. Approved-but-not-Issued means the signer
rejected the request's shape — usually a usage or subject the signer won't sign.

## Your turn

`init` submitted a node's kubelet **serving**-cert CSR on **cplane-01**. Because
kubelet-serving certs are never auto-approved, it's Pending and will stay that
way until you act.

Get the node in:

1. Find the pending request — `kubectl get csr` — and inspect it: `kubectl get
   csr kubelings-node-csr -o yaml`. Confirm it's a `system:node:...`
   kubelet-serving cert.
2. Approve it — `kubectl certificate approve kubelings-node-csr`.
3. Confirm a certificate was issued — `kubectl get csr` should show
   `Approved,Issued`.

The check passes only when the CSR is **Approved and a certificate has been
issued** — approval that the signer then refuses doesn't count.

<details>
<summary>Hint</summary>

```sh
kubectl get csr                                  # find it: CONDITION Pending
kubectl get csr kubelings-node-csr -o yaml       # signerName + decoded subject
kubectl certificate approve kubelings-node-csr
kubectl get csr kubelings-node-csr               # CONDITION -> Approved,Issued
```

If it reads `Approved` but not `Issued` after a few seconds, the signer
declined the request's shape — but this one is a well-formed
`kubernetes.io/kubelet-serving` request (right subject, SAN, and `server auth`
usage), so a plain `approve` issues it.

</details>

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>

```sh
# 1 · find and read the pending request
kubectl get csr
kubectl get csr kubelings-node-csr -o yaml
#   signerName: kubernetes.io/kubelet-serving
#   subject decodes to O=system:nodes, CN=system:node:node-bootstrap (+ SAN)
#   usages: server auth, digital signature, key encipherment

# 2 · approve it
kubectl certificate approve kubelings-node-csr

# 3 · confirm the signer issued the cert
kubectl get csr kubelings-node-csr               # Approved,Issued
kubectl get csr kubelings-node-csr -o jsonpath='{.status.certificate}' | head -c 40; echo
```

In a real bootstrap the kubelet is watching its own CSR and writes the issued
cert to `/var/lib/kubelet/pki/` the moment it appears — you just supplied the
approval the flow was waiting on.

</details>

## Root cause, restated

Certificates are how a node proves it's a node, and approval is the human (or
policy) gate on minting that identity.

- **Bootstrap and rotation are the same CSR flow.** A joining kubelet and a
  kubelet renewing an expiring cert both submit a CSR and wait for it to be
  signed. A Pending CSR is a node that can't authenticate yet.
- **Read `signerName`, subject, and usages before approving.** You're granting
  `system:node:<name>` — a real, trusted identity. The signer name tells you
  client vs serving; the subject tells you *who* you're vouching for.
- **`Approved` ≠ `Issued`.** Approval only unblocks the signer; if the usages or
  subject don't fit the signer, no cert comes out. The finish line is
  `Approved,Issued` with a populated `status.certificate`.
