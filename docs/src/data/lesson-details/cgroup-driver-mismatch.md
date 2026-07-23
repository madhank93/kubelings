> **☁ iximiuz Labs only.** This is a node-config fault in the kubelet and
> containerd's own config files, fixed with `systemctl` on the host. Real node,
> root, systemd — outside the kubectl sandbox.

## One cgroup hierarchy, two things trying to own it

Every container's CPU and memory limits are enforced by Linux **cgroups**.
Something has to create those cgroups and write the limits — and on a
Kubernetes node, *two* components do cgroup work: the **kubelet** and the
**container runtime** (containerd). For that to be coherent, they must use the
same **cgroup driver** — the same convention for how cgroups are named and
nested.

There are two drivers:

- **`systemd`** — cgroups are managed as systemd units/slices. On any host
  where systemd is PID 1 (all of these), systemd expects to be the single
  writer of the cgroup tree, so this is the correct choice.
- **`cgroupfs`** — the kubelet/runtime manage cgroups directly via the cgroup
  filesystem, bypassing systemd.

Pick different drivers on the two components and you get **two cgroup managers
maintaining separate views of one hierarchy**. Modern kubelet + containerd are
tolerant enough that a mismatched node can keep running and even look `Ready` —
which is exactly what makes this dangerous. The Kubernetes docs are blunt that
the configuration is unsupported: resource accounting is split, and the failure
shows up later and under pressure — inconsistent limits and eviction accounting,
and instability when the node is stressed or restarts. A green node is not proof
the drivers agree; you have to look.

## Where each side is configured

- **kubelet**: `cgroupDriver:` in `/var/lib/kubelet/config.yaml`. kubeadm sets
  this to `systemd`.
- **containerd**: `SystemdCgroup` under the runc runtime options in
  `/etc/containerd/config.toml`. `SystemdCgroup = true` means systemd.
  `containerd config dump | grep -i SystemdCgroup` shows the effective value.

The rule is simply: **both say systemd, or both say cgroupfs.** On a systemd
host you want both `systemd`. The mismatch usually appears exactly the way this
lesson stages it — someone changed one side (a containerd upgrade that reset
the config, a hand-edit to "fix" something) and forgot the other.

## Detecting it

You can't rely on a symptom here — the node may be `Ready` and pods may run.
The reliable detection is to read the two values and compare them directly:

```sh
grep cgroupDriver /var/lib/kubelet/config.yaml         # kubelet's driver
containerd config dump | grep -i SystemdCgroup         # containerd's driver (true = systemd)
```

If they disagree, that's the whole bug — no log will hand it to you. Align them
to `systemd` and restart the component you changed (kubelet, containerd, or
both). `journalctl -u kubelet` may be quiet throughout; that silence is the
lesson — the mismatch doesn't announce itself.

## Your turn

`init` set **node-02**'s kubelet to `cgroupDriver: cgroupfs` while containerd
stays on systemd. The two disagree.

Fix it:

1. On **node-02**, confirm the mismatch — `grep cgroupDriver
   /var/lib/kubelet/config.yaml` vs `containerd config dump | grep -i
   SystemdCgroup`.
2. Align **both to systemd** (the correct driver on this systemd host), and
   restart whatever you changed.
3. Confirm node-02 is `Ready`.

The check requires the kubelet's `cgroupDriver: systemd`, containerd's
`SystemdCgroup = true`, and node-02 back to `Ready`.
