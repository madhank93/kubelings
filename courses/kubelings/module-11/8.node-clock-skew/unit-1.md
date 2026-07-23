---
kind: unit
title: "clock skew: time is a security input"
name: node-clock-skew-unit
---


> **☁ iximiuz Labs only.** You'll move a real node's system clock and repair its
> time synchronisation — `timedatectl`, `chronyd`, root. There's no clock to
> skew inside the kubectl sandbox; this needs a machine.

## The failure that looks like broken certs but isn't

A node goes `NotReady`, and `journalctl -u kubelet` is full of:

```
x509: certificate has expired or is not yet valid
```

Your first instinct is a cert problem — did something expire, is rotation
broken (7.12)? But every certificate is fine. The problem is the *clock*.
Certificate validity is a pair of timestamps, `notBefore` and `notAfter`, and
"is this cert valid?" is answered by comparing them to **the local clock**.
Skew the clock far enough and a perfectly good cert reads as expired (clock too
far forward) or not-yet-valid (clock too far back) — on that node only.

TLS, ServiceAccount tokens (their `exp`/`nbf` claims), and log ordering all
assume clocks across the cluster are close. A node whose clock has drifted
minutes-to-days out isn't just inaccurate — it's **cryptographically
untrusted**, because it can't agree with anyone else about whether a credential
is currently valid. Time is a security input, not just a display value.

## Why nodes keep time, and how it breaks

Nodes run a time-sync daemon — **chrony** (`chronyd`) or
**systemd-timesyncd** — that disciplines the clock against upstream NTP servers,
correcting the small constant drift of any hardware clock. Skew happens when
that breaks: the daemon is stopped or misconfigured, NTP egress is firewalled,
a VM resumes from a long pause, or someone hand-sets the clock and disables
sync. Left uncorrected, drift accumulates until it crosses something's
tolerance — and TLS has very little.

## Diagnosing and fixing

```sh
date                 # the node's current wall-clock time — is it obviously wrong?
timedatectl          # Local time, whether NTP is enabled, and sync status
journalctl -u kubelet -n 30 --no-pager    # x509 expired/not-yet-valid errors
```

The fix is to restore synchronisation, not to hand-set the clock once (that
just drifts again):

```sh
timedatectl set-ntp true              # re-enable time synchronisation
systemctl restart chronyd             # (or systemd-timesyncd) start the daemon
chronyc makestep                      # chrony: step the clock to correct NOW,
                                      # instead of slewing slowly toward it
```

`chronyc makestep` matters when the skew is large: chronyd normally *slews*
(adjusts gradually) to avoid jumping time backwards on a running system, but a
400-day error would take forever to slew out. `makestep` forces an immediate
correction. Once the clock is right, the kubelet's next TLS handshake succeeds
and the node returns to `Ready` on its own.

## Your turn

`init` pushed **node-01**'s clock ~400 days into the future and turned time sync
off. Its kubelet now sees the cluster's certs as expired and can't authenticate.

Repair it:

1. On **node-01**, confirm the skew — `date`, `timedatectl` — and the resulting
   `x509` errors in `journalctl -u kubelet`.
2. Turn synchronisation back on and get the clock corrected (a large step may
   need `chronyc makestep`).
3. Confirm node-01 is `Ready`.

The check requires time sync to be **on** and node-01 back to `Ready` — and
because the check reaches the API over TLS, a still-skewed clock can't pass it.

<details>
<summary>Hint</summary>

`date` will show the clock is wildly ahead. The fix is to restore NTP, not to
`date -s` it by hand:

```sh
timedatectl                       # NTP: no, clock far ahead
timedatectl set-ntp true
systemctl restart chronyd 2>/dev/null || systemctl restart systemd-timesyncd
chronyc makestep 2>/dev/null      # force an immediate step if using chrony
timedatectl                       # NTP: yes, time correct
```

Give the kubelet a few seconds after the clock corrects — its next handshake
re-authenticates the node and it goes `Ready`. Still seeing x509 errors means
the clock hasn't actually stepped yet.

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
# 1 · confirm the skew and the symptom on node-01
date
timedatectl                                   # NTP: no; Local time ~400d ahead
journalctl -u kubelet -n 30 --no-pager        # x509: certificate has expired...

# 2 · restore synchronisation and step the clock back to now
timedatectl set-ntp true
systemctl restart chronyd 2>/dev/null || systemctl restart systemd-timesyncd
chronyc makestep 2>/dev/null || true          # large skew: step, don't slew

# 3 · confirm
timedatectl                                   # NTP: yes; time correct
kubectl --kubeconfig=/etc/kubernetes/kubelet.conf get node node-01
```

The kubelet re-establishes TLS the moment the clock is sane again — you don't
touch any certificate, because none were ever wrong.

</details>

## Root cause, restated

The certs were fine the whole time; the node just disagreed with everyone about
what time it was.

- **Cert validity is checked against the local clock.** Skew a node far enough
  and good certificates read as expired or not-yet-valid — but only on that
  node. `x509: certificate has expired or is not yet valid` plus a NotReady node
  should make you check `date` before you touch any PKI.
- **Restore sync, don't hand-set.** `timedatectl set-ntp true` and a running
  chronyd/timesyncd fix the cause; a manual `date -s` fixes the symptom and
  drifts out again.
- **Large skew needs a step, not a slew.** chronyd corrects gradually by design;
  `chronyc makestep` forces the immediate jump a big error requires.
