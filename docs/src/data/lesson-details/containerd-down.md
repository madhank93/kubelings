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
