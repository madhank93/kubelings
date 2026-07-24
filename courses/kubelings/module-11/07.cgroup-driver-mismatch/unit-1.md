---
kind: unit
title: "cgroup driver mismatch: two managers, one hierarchy"
name: cgroup-driver-mismatch-unit
---


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

<details>
<summary>Hint</summary>

The two drivers must match. `init` moved the kubelet off systemd; containerd is
still on systemd, so the fix is to put the kubelet back:

```sh
grep cgroupDriver /var/lib/kubelet/config.yaml         # cgroupfs  <- wrong
containerd config dump | grep -i SystemdCgroup         # true (systemd)

sed -i 's/^cgroupDriver:.*/cgroupDriver: systemd/' /var/lib/kubelet/config.yaml
systemctl restart kubelet
```

The baseline `init` recorded is in `/root/kubelings-cgroup-baseline`. Give the
kubelet a few seconds after restart to re-register the node as Ready.

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
# 1 · confirm the mismatch on node-02 (the node may still read Ready)
grep cgroupDriver /var/lib/kubelet/config.yaml         # cgroupfs
containerd config dump | grep -i SystemdCgroup         # SystemdCgroup = true (systemd)

# 2 · align the kubelet back to systemd (containerd is already systemd)
sed -i 's/^cgroupDriver:.*/cgroupDriver: systemd/' /var/lib/kubelet/config.yaml
systemctl restart kubelet

# 3 · confirm
grep cgroupDriver /var/lib/kubelet/config.yaml         # systemd
kubectl --kubeconfig=/etc/kubernetes/kubelet.conf get node node-02
```

Here containerd was the correct side, so only the kubelet moved. If a bad
containerd upgrade had reset `SystemdCgroup` to `false` instead, the fix would
be the mirror image: set `SystemdCgroup = true` in
`/etc/containerd/config.toml` and `systemctl restart containerd`.

</details>

## Root cause, restated

Two components manage cgroups on a node; they must agree on how.

- **kubelet and runtime must share one cgroup driver.** Disagreement splits
  resource accounting across two incompatible views of one hierarchy — the node
  gets unstable, not cleanly "off."
- **On a systemd host, the answer is systemd.** systemd wants to be the single
  cgroup writer, so both the kubelet (`cgroupDriver`) and containerd
  (`SystemdCgroup = true`) should say systemd.
- **The mismatch is a change-management bug.** It appears when one side is
  edited or upgraded and the other is forgotten — which is why "check both,
  align both" is the fix, every time.
