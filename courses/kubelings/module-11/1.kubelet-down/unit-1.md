---
kind: unit
title: "kubelet down: the node is only as healthy as its host"
name: kubelet-down-unit
---


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

<details>
<summary>Hint</summary>

`journalctl -u kubelet -n 40 --no-pager` prints the startup error verbatim —
it will name a flag it doesn't recognize. That flag isn't in the kubelet's own
`--config`; it's being handed to the process as an extra command-line arg.

See what's feeding the command line:

```sh
systemctl cat kubelet          # shows ExecStart and its EnvironmentFile= sources
grep KUBELET_EXTRA_ARGS /etc/default/kubelet
```

`KUBELET_EXTRA_ARGS` in `/etc/default/kubelet` carries the bad flag. Clear it
(remove the flag, leaving `KUBELET_EXTRA_ARGS=`) and `systemctl restart
kubelet` — restart re-reads the environment file.

Give the kubelet a few seconds after it starts to re-register and post its
`Ready` status before you re-check the node.

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
# 1 · on node-01, read WHY it won't start — the journal names the flag
systemctl status kubelet
journalctl -u kubelet -n 40 --no-pager      # "unknown flag: --kubelings-nonexistent-flag"

# 2 · find where the flag comes from — the extra-args env file
systemctl cat kubelet                       # ExecStart sources EnvironmentFile=/etc/default/kubelet
grep KUBELET_EXTRA_ARGS /etc/default/kubelet

# 3 · clear the bad flag and restart (restart re-reads the env file)
sed -i 's|^KUBELET_EXTRA_ARGS=.*|KUBELET_EXTRA_ARGS=|' /etc/default/kubelet
systemctl restart kubelet

# 4 · confirm it's healthy and the node is back
systemctl is-active kubelet
kubectl --kubeconfig=/etc/kubernetes/kubelet.conf get node node-01
```

`KUBELET_EXTRA_ARGS` is where kubeadm expects operator-added flags, which is why
one typo there takes the node down — and why the fix is to empty it, not to
touch the unit file.

</details>

## Root cause, restated

The kubelet is the seam between the cluster and the host, and it fails on the
host's terms — a bad flag, an unreadable cert, a runtime socket that isn't
there. The tell is always the same: `NotReady` in `kubectl`, the reason in
`journalctl -u kubelet`.

Three things to carry out of this drill:

- **NotReady is a symptom with one source of truth.** The API server can only
  report that a node went quiet. Every diagnosis starts with `systemctl status
  kubelet` and `journalctl -u kubelet` on the node itself.
- **Startup failures are config failures.** A kubelet that won't stay up is
  almost never "broken software" — it's a flag, a `--config` file, a cert path,
  or a CRI endpoint it can't use. The journal line before the exit names it.
- **Know where the flags come from.** kubeadm sources `KUBELET_EXTRA_ARGS` from
  `/etc/default/kubelet` into the `ExecStart`. A restart re-reads it; a bad flag
  there crash-loops the kubelet until you clear it.
