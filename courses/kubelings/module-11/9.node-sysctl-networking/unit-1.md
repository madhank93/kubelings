---
kind: unit
title: "the kernel knobs kubeadm needs: br_netfilter, ip_forward"
name: node-sysctl-networking-unit
---


> **☁ iximiuz Labs only.** Loading kernel modules and writing sysctls is host,
> root, kernel work — there's nothing to `modprobe` inside the kubectl sandbox.
> You get a real node whose networking you can break at the kernel and repair.

## Below the CNI: the kernel has to cooperate

M4 taught the CNI — how pods get addresses and how Services map to endpoints.
All of that assumes the **node's kernel** is set up to forward and filter
packets the way Kubernetes needs. When those kernel prerequisites are missing,
the CNI config can be perfect and pods still can't talk: this is a layer *below*
the CNI, which is why `kubectl` shows you nothing useful and the node can even
be `Ready`. Two knobs do most of the work.

**`net.ipv4.ip_forward = 1`.** A pod's traffic leaving for another node — or the
internet — is *routed through* the node. Linux only forwards packets between
interfaces when IP forwarding is on. With it off, the node is a host that talks
for itself but won't route for the pods behind it, so cross-node and egress pod
traffic dies.

**`br_netfilter` + `net.bridge.bridge-nf-call-iptables = 1`.** Pods on a node
usually hang off a Linux **bridge**. By default, traffic that stays on a bridge
is *not* seen by iptables — but kube-proxy implements Services as iptables (or
ipvs) rules. Loading the `br_netfilter` module and setting
`bridge-nf-call-iptables = 1` forces bridged packets through iptables, so
kube-proxy's Service NAT actually applies to pod traffic. Without it, a pod's
call to a Service ClusterIP is never DNAT'd to a backend — the connection just
hangs.

This is the exact checklist in the kubeadm install docs, and the most common
"I followed the guide but nothing works" cause.

## Runtime *and* boot — the part people miss

There are two places every one of these lives, and you must set both:

- **Runtime** — the live kernel, effective immediately:

  ```sh
  modprobe br_netfilter
  sysctl -w net.ipv4.ip_forward=1
  sysctl -w net.bridge.bridge-nf-call-iptables=1
  ```

- **Persistent** — reapplied on every boot:

  ```sh
  echo br_netfilter > /etc/modules-load.d/k8s.conf          # load the module at boot
  cat >/etc/sysctl.d/k8s.conf <<'EOF'
  net.ipv4.ip_forward = 1
  net.bridge.bridge-nf-call-iptables = 1
  net.bridge.bridge-nf-call-ip6tables = 1
  EOF
  sysctl --system                                           # apply all sysctl.d now
  ```

The classic trap is doing only the first set: the node works *today*, everyone
moves on, and then a reboot months later silently reverts the sysctls and the
node comes back broken with no obvious change. `bridge-nf-call-iptables` has a
second gotcha — the sysctl key **only exists once `br_netfilter` is present**, so
the module must be there *before* the sysctl is applied, which is exactly why
both the modules-load.d and sysctl.d files are needed.

(Some kernels — including this playground's — compile `br_netfilter` *into* the
kernel rather than as a loadable module. Then `modprobe br_netfilter` is a
harmless no-op, `lsmod` won't list it, and the sysctl key is always available.
You still write the `modules-load.d` entry: it's what makes the setup correct on
the many hosts where `br_netfilter` *is* a module, and it costs nothing where it
isn't.)

## Your turn

`init` turned off **node-02**'s pod-networking kernel prerequisites — `ip_forward`
and `bridge-nf-call-iptables` are 0, `br_netfilter` may be unloaded, and the
config files that would restore them at boot are disabled.

Restore them properly:

1. On **node-02**, confirm the state — `lsmod | grep br_netfilter`, `sysctl
   net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables`.
2. Load the module and set both sysctls to 1 **at runtime**.
3. **Persist** all three under `/etc/modules-load.d/` and `/etc/sysctl.d/` so a
   reboot doesn't undo your work.

The check verifies the live values *and* that they're persisted — a runtime-only
fix won't pass, because it wouldn't survive a reboot.

<details>
<summary>Hint</summary>

Fix it in both places. Runtime first:

```sh
modprobe br_netfilter
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.bridge.bridge-nf-call-iptables=1
```

Then persist — this is what the check is really looking for:

```sh
echo br_netfilter > /etc/modules-load.d/k8s.conf
cat >/etc/sysctl.d/k8s.conf <<'EOF'
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-iptables = 1
EOF
sysctl --system
```

Load the module *before* the sysctl — `net.bridge.bridge-nf-call-iptables`
doesn't exist as a key until `br_netfilter` is in the kernel.

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
# 1 · see the broken state on node-02
sysctl net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables   # 0 (or key absent)
lsmod | grep br_netfilter                       # empty here (built into this kernel)

# 2 · runtime: load the module, then set the sysctls
modprobe br_netfilter
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.bridge.bridge-nf-call-iptables=1

# 3 · persist across reboot
echo br_netfilter > /etc/modules-load.d/k8s.conf
cat >/etc/sysctl.d/k8s.conf <<'EOF'
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF
sysctl --system

# 4 · confirm
lsmod | grep br_netfilter
sysctl net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables
```

The module has to load before the sysctl is applied — that ordering is why the
persistent config is split into `modules-load.d` (loads it) and `sysctl.d`
(sets the key it exposes).

</details>

## Root cause, restated

Kubernetes networking is built on kernel features that don't turn themselves on.

- **This is below the CNI.** IP forwarding and bridge-netfilter are the kernel
  plumbing the CNI and kube-proxy assume. Missing, pods can't route and Service
  NAT silently doesn't apply — and `kubectl` won't tell you, because the fault
  is in the kernel, not the API.
- **Two knobs, one dependency.** `ip_forward=1` lets the node route pod traffic;
  `br_netfilter` + `bridge-nf-call-iptables=1` make kube-proxy's rules see
  bridged traffic. The sysctl key only exists once the module is loaded.
- **Set it twice: runtime and boot.** A runtime-only fix works until the next
  reboot, then reverts with no visible cause. Persist to `/etc/modules-load.d/`
  and `/etc/sysctl.d/` so the node comes up correct every time.
