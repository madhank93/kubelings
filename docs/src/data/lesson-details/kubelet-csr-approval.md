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
