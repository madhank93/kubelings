---
kind: unit
title: "Certificates: the outage scheduled a year in advance"
name: cert-rotation-unit
---


> **☁ iximiuz Labs only.** `kubeadm certs` runs as root on control-plane nodes —
> host-level, outside the kubectl sandbox, so this one can't run on your local
> `kind` cluster. Here you get a real, disposable control-plane VM: read the
> runbook, then run the drill for real at the bottom. The PKI files here are
> the ones `kubeadm-bootstrap` (7.10) wrote.

## The failure mode with a due date

Every arrow in the M7 architecture — kubectl→apiserver,
apiserver→kubelet, apiserver→etcd, scheduler→apiserver — is mTLS, backed by
the files kubeadm generated in `/etc/kubernetes/pki/`. kubeadm's defaults:

- **leaf certificates: 1 year**
- **CAs: 10 years**

Nothing renews leaves automatically on a plain kubeadm cluster (one
exception below). Day 366 on an untouched cluster:

```
Unable to connect to the server: x509: certificate has expired or is not yet valid
```

kubectl locked out, controllers can't talk to the apiserver, kubelets fail
their heartbeats, nodes drift NotReady — while every workload pod keeps
serving (they don't use this PKI). A control-plane-only outage that was
scheduled the day the cluster was born. It's a postmortem cliché for a
reason.

The accidental save: `kubeadm upgrade` (M8.7) **renews all certs as a side
effect** — clusters upgraded at least yearly never notice. Clusters that
"don't need upgrades" are exactly the ones that hit day 366.

## The runbook

**1 · Check — put this in monitoring, not in memory:**

```sh
kubeadm certs check-expiration
```

```
CERTIFICATE                EXPIRES                  RESIDUAL TIME   EXTERNALLY MANAGED
admin.conf                 Jul 12, 2027 19:04 UTC   364d            no
apiserver                  Jul 12, 2027 19:04 UTC   364d            no
apiserver-etcd-client      …                        364d            no
apiserver-kubelet-client   …                        364d            no
etcd-server / etcd-peer …  …                        364d            no
front-proxy-client         …                        364d            no

CERTIFICATE AUTHORITY   EXPIRES                  RESIDUAL TIME
ca                      Jul 10, 2036 19:04 UTC   9y
etcd-ca                 …                        9y
```

Two lifetimes, two problems: leaves are the yearly chore (this page); CA
expiry at year 10 is a genuinely painful cluster-wide re-trust — calendar
it the day you build anything long-lived.

**2 · Renew** — everything, or one cert:

```sh
kubeadm certs renew all            # the yearly move
kubeadm certs renew apiserver      # or surgical
```

Renewal re-signs with the existing CA — clients trust the CA, so nothing
else needs to change. This also rewrites the cert-embedding kubeconfigs
(`admin.conf`, `controller-manager.conf`, `scheduler.conf`).

**3 · Restart the consumers.** Renewed files on disk ≠ renewed certs in
memory. The static pods must reload them:

```sh
# the blunt, reliable way on each control-plane node:
mv /etc/kubernetes/manifests/*.yaml /tmp/manifests/ && sleep 20 \
  && mv /tmp/manifests/*.yaml /etc/kubernetes/manifests/
crictl ps | grep -E 'apiserver|scheduler|controller|etcd'   # fresh, Running
```

(Moving manifests out and back forces the kubelet to fully recreate the
pods — a `delete pod` of a static pod's mirror only restarts it via the
same manifest, which usually works too; the move is the documented sure
thing.)

**4 · Refresh the humans.** Your `~/.kube/config` is a copy of the *old*
`admin.conf` (7.10 step 2) — copy the renewed one again, and every operator
laptop with a year-old kubeconfig needs the same (M4.11: kubeconfigs are
standing credentials with lifetimes).

**5 · Verify:** `kubeadm certs check-expiration` → 364d across the board;
`kubectl get nodes` answers.

On HA (7.11): repeat per control-plane node — each has its own PKI dir —
one node at a time, the same discipline as any control-plane change.

## The one thing that does auto-rotate

**Kubelet client certs** rotate themselves: the kubelet requests a new cert
via the CSR API before expiry (`rotateCertificates: true`, default on
kubeadm clusters — the TLS-bootstrap flow from 7.10 join, repeating
forever). That's why `check-expiration` doesn't list kubelet certs. Verify
a node's rotation is alive:

```sh
ls -l /var/lib/kubelet/pki/           # kubelet-client-current.pem → dated file
kubectl get csr | tail                # the paper trail, Approved,Issued
```

Kubelet *serving* certs can rotate too (`serverTLSBootstrap: true`) but
require explicit CSR approval — the default self-signed serving certs are a
kube-bench (M6.7) finding you've possibly already met.

## If you're reading this after day 366

Locked out — kubectl can't authenticate to renew anything's trust? The
renewal itself never needed the API: `kubeadm certs renew all` works
directly on the files as root on the node. Restart the static pods, re-copy
`admin.conf`, and the cluster comes back with no data loss. Expired certs
are a lockout, not a corruption — etcd (7.6) never stopped holding your
state. Breathe first: renew, restart, recopy.

## Takeaway

- kubeadm leaves live 1 year, CAs 10 — both expiries are appointments, put
  `check-expiration` in monitoring (alert < 30d).
- Renew = re-sign + **restart the static pods** + re-copy `admin.conf`;
  forgetting either half is the classic incomplete fix.
- `kubeadm upgrade` renews as a side effect — regular upgrades (M8.7) make
  this page a non-event; frozen clusters make it a 3 a.m. one.
- Kubelet client certs self-rotate via CSR; that machinery working is worth
  a periodic `kubectl get csr` glance.
- CKA loves exactly this: `check-expiration` output reading, `renew`,
  which components restart, where admin.conf comes from.

## Your turn

Nothing is broken — this is the drill, run on a real control plane before the
day you have to run it at 3 a.m.

`init` recorded two timestamps: when the apiserver certificate **on disk**
expires, and when the certificate the apiserver is **actually serving**
expires. Today they're identical. Make both of them later:

1. Look at what you have: `kubeadm certs check-expiration`
2. Renew the leaves.
3. Make the running control plane actually *use* them.
4. Make sure `kubectl` still works afterwards.

The check tests those last three separately, and will tell you which one
you've skipped.

<details>
<summary>Hint</summary>

Step 3 is the one everyone forgets, and the check is built to catch it
specifically.

`kubeadm certs renew all` rewrites files in `/etc/kubernetes/pki/`. The
apiserver read its certificate into memory when it started and will keep
serving that one until the process restarts — so on disk you have a fresh
cert, on the wire you have the expiring one. Confirm the gap yourself:

```sh
openssl x509 -enddate -noout -in /etc/kubernetes/pki/apiserver.crt   # new
echo | openssl s_client -connect 127.0.0.1:6443 2>/dev/null \
  | openssl x509 -enddate -noout                                    # still old
```

Step 4: `renew all` also rewrites `/etc/kubernetes/admin.conf`, which embeds
a client certificate. Your `~/.kube/config` is a *copy* made earlier, so it
still holds the old one.

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
# 1 · what have we got
kubeadm certs check-expiration

# 2 · re-sign the leaves with the existing CA
kubeadm certs renew all

# 3 · force the kubelet to recreate the static pods so they reload the files
mkdir -p /tmp/manifests
mv /etc/kubernetes/manifests/*.yaml /tmp/manifests/
sleep 20
mv /tmp/manifests/*.yaml /etc/kubernetes/manifests/
until kubectl get --raw=/readyz >/dev/null 2>&1; do sleep 2; done
crictl ps | grep -E 'apiserver|scheduler|controller|etcd'   # fresh, Running

# 4 · your kubeconfig is a year-old copy of admin.conf
cp /etc/kubernetes/admin.conf ~/.kube/config

# 5 · prove it
kubeadm certs check-expiration          # 364d across the board
echo | openssl s_client -connect 127.0.0.1:6443 2>/dev/null \
  | openssl x509 -enddate -noout        # the served cert, now new
kubectl get nodes
```

## Root cause, restated

There is no root cause here — that's the lesson. This outage has no trigger,
no bad deploy, no traffic spike. It is an appointment made the day the
cluster was created, kept exactly one year later, and the only thing that
prevents it is someone remembering.

Which is why the real fix isn't this runbook, it's `check-expiration` in
monitoring with an alert under 30 days — plus knowing that regular
`kubeadm upgrade` (M8.7) renews as a side effect, so the clusters most at
risk are the "stable" ones nobody touches.

And if you are reading this *after* day 366: renewal never needed the API.
`kubeadm certs renew all` works on the files as root. Expired certs are a
lockout, not a corruption — etcd never stopped holding your state. Renew,
restart, re-copy.

</details>
