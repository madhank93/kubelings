> **Runbook reading.** `kubeadm certs` runs as root on control-plane nodes —
> host-level, outside the kubectl sandbox. Rehearse on a kind node
> (`docker exec -it <cp-node> bash` — kubeadm is inside) or an iximiuz VM.
> The PKI files here are the ones `kubeadm-bootstrap` (7.10) wrote.

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
