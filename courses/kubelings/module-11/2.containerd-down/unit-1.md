---
kind: unit
title: "containerd down: no CRI, no pods"
name: containerd-down-unit
---


> **☁ iximiuz Labs only.** The container runtime is a host service — systemd,
> a Unix socket, root. You get a real worker where you can stop containerd,
> watch the CRI vanish, and put it back with `crictl`, `systemctl`, and
> `journalctl`.

## The layer under the kubelet

Lesson 11.1 was the kubelet failing on its own terms. This one is the layer
directly beneath it. The kubelet doesn't run containers itself — it speaks the
**Container Runtime Interface** (CRI) over a Unix socket to a runtime, on
kubeadm nodes that's **containerd** at `/run/containerd/containerd.sock`.
containerd, in turn, calls `runc` to actually create the Linux namespaces and
cgroups.

Pull containerd out and the whole stack above it is blind: the kubelet's CRI
calls fail, no pod can be created or even torn down cleanly, and image pulls
stop. The node usually goes `NotReady` too, because a kubelet that can't reach
its runtime can't honestly report health. But the *cause* is one layer lower
than the kubelet — and you diagnose it from the runtime's own tools.

## crictl is your kubectl for the runtime

When the API path is broken or you're below it, `crictl` talks straight to the
CRI socket — the same socket the kubelet uses:

```sh
crictl ps                 # running containers, from the runtime's view
crictl images             # what's actually pulled on this node
crictl version            # smoke test: does the CRI answer at all?
```

If `crictl` returns *connection refused* or *no such file or directory* for
the socket, the runtime is down or listening somewhere else — that's your
signal to look at the service:

```sh
systemctl status containerd
journalctl -u containerd -n 40 --no-pager
```

Like the kubelet, containerd's common failure is startup config: a malformed
`/etc/containerd/config.toml`, a `--config` pointing at a file that isn't
there, a bad snapshotter or cgroup setting. The journal prints the parse or
open error and exits. A runtime that won't stay up is a runtime that couldn't
read its config.

## Your turn

`init` broke containerd on **node-01** by overriding its `ExecStart` to load a
config file that doesn't exist. containerd exits on startup, the CRI socket is
gone, and node-01 can't run pods.

Recover it:

1. On **node-01**, confirm the runtime is the problem: `crictl ps` (connection
   refused), then `systemctl status containerd` and `journalctl -u containerd`.
2. Find and remove the bad configuration, reload systemd, restart containerd.
3. Confirm `crictl` talks to the CRI again and node-01 is `Ready`.

The check verifies the runtime is genuinely healthy: the bad override removed,
containerd active, `crictl` connecting, and the node back to `Ready`.

<details>
<summary>Hint</summary>

`journalctl -u containerd` will show it failing to open a config file — a path
that doesn't exist. That path isn't containerd's default; it's being forced by
a systemd drop-in overriding `ExecStart`.

See exactly what systemd is running:

```sh
systemctl cat containerd       # unit + every drop-in; find the bad ExecStart
ls /etc/systemd/system/containerd.service.d/
```

Delete the drop-in that isn't part of the base install, `systemctl
daemon-reload` (so systemd drops the overridden command line), then `systemctl
restart containerd`. Verify with `crictl version` before you check the node —
`crictl` answering is the proof the socket is back.

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
# 1 · confirm the runtime is down, from the runtime's own tools
crictl ps                                   # connection refused: socket gone
systemctl status containerd                 # failed / auto-restart
journalctl -u containerd -n 40 --no-pager   # can't open .../kubelings-nonexistent.toml

# 2 · find the override forcing the bad --config
systemctl cat containerd                    # shows ExecStart pointing at the missing file
ls /etc/systemd/system/containerd.service.d/

# 3 · remove it, reload, restart
rm /etc/systemd/system/containerd.service.d/99-kubelings-broken.conf
systemctl daemon-reload
systemctl restart containerd

# 4 · prove the CRI is back, then the node
crictl version
kubectl --kubeconfig=/etc/kubernetes/kubelet.conf get node node-01
```

The kubelet reconnects to the restored socket on its own within a few seconds
and node-01 returns to `Ready` — you don't restart the kubelet, only the layer
you actually broke.

</details>

## Root cause, restated

containerd is the runtime the kubelet drives; when it's gone, everything above
it is blind, but the diagnosis lives one layer down from `kubectl`.

- **`crictl` is the runtime's `kubectl`.** When pods won't start and the node's
  gone quiet, `crictl ps` / `crictl version` tell you whether the CRI is even
  answering — before you touch anything else.
- **Runtime down = pods down, node NotReady.** A kubelet that can't reach its
  CRI can't run or report on workloads, so the symptom looks identical to a
  kubelet outage. `crictl` is what tells the two apart.
- **containerd fails on config, like the kubelet.** A bad `config.toml` or a
  `--config` path that doesn't exist stops it at startup. `journalctl -u
  containerd` names the file; `daemon-reload` after fixing it, then restart.
