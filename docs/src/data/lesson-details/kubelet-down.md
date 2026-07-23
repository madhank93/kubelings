> **☁ iximiuz Labs only.** This fault lives in a systemd drop-in on the node —
> below the API, outside this course's kubectl-only sandbox. You get a real,
> disposable worker with root and systemd, and you'll diagnose the kubelet the
> way you would on a pager: `systemctl`, `journalctl`, fix, restart.

## The one process the control plane can't restart for you

Every other lesson so far, the cluster could self-heal: a crashed pod gets
recreated, a drained node's work reschedules. The kubelet is where that stops.
It is the agent that *makes* a node a node — it registers with the API server,
posts the `Ready` condition, pulls images, and drives the CRI to run
containers. Nothing in the control plane can start it, because starting it is
the host's job, not the cluster's.

So when the kubelet is down, you see the symptom from `kubectl` — `node-01
NotReady`, its pods stuck `Terminating` or `NodeLost` — but the *fix* is not a
`kubectl` command. It's `ssh`, `systemctl`, `journalctl`. M8's node-notready
lesson taught you to read the symptom. This is the failure it was pointing at.

## What NotReady actually means

The node's `Ready` condition is a lease the kubelet renews. Stop renewing and,
after `node-monitor-grace-period` (~40s), the node controller flips the node to
`NotReady`; after `pod-eviction-timeout` the pods are marked for eviction. The
control plane is reacting correctly to silence — it has no way to tell "kubelet
crashed" from "node caught fire." That ambiguity is the whole point: **a node
going NotReady tells you a node stopped talking, never why.** The why is always
on the host.

## Reading the host like an operator

Two commands answer almost every kubelet-down page:

```sh
systemctl status kubelet        # active? failed? what was the last exit code?
journalctl -u kubelet -n 40 --no-pager   # WHY it exited — the actual error
```

`status` tells you the state; `journalctl` tells you the reason. A kubelet that
`activating (auto-restart)` in a tight loop is failing at startup — and startup
failures are almost always **config**: a flag it doesn't recognize, a cert it
can't read, a `--config` file that won't parse, a container runtime endpoint
it can't reach. The journal names it on the line right before the exit.

The kubelet's command line is assembled by systemd: kubeadm's drop-in
(`/usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf`) builds the
`ExecStart` and sources extra flags from an environment file — on kubeadm,
`KUBELET_EXTRA_ARGS` in `/etc/default/kubelet`. That file is the sanctioned
place to add a kubelet flag, and so it's exactly where a well-meaning "just add
one flag" change becomes a node outage: one bad flag there and the kubelet
won't parse its own command line.

## Your turn

`init` planted the classic version of this outage on **node-01**: a drop-in
that injects a kubelet flag the binary doesn't understand, and the kubelet is
stopped. On `cplane-01` you'll see node-01 sliding to `NotReady`.

Bring it back:

1. On **node-01**, find out why the kubelet won't start — `systemctl status
   kubelet`, then `journalctl -u kubelet`. Read the error, don't guess.
2. Remove the bad configuration, reload systemd, and start the kubelet.
3. Confirm node-01 reports `Ready` again.

The check verifies the *cause* is gone, not just the symptom: the bad drop-in
must be removed (not merely overridden), the kubelet running, and node-01
Ready.
