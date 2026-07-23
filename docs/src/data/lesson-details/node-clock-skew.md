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
