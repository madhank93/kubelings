---
kind: unit
title: "Falco: alarms for the attack you didn't prevent"
name: falco-runtime-detection-unit
---


> **Runbook reading.** Falco loads an eBPF probe into each node's kernel —
> host-level by nature, outside the kubectl sandbox (M6.8's confinement).
> The install runbook below works on a Linux Docker host's kind node
> (kernel ≥ 5.8 for the modern eBPF probe) or an iximiuz playground VM;
> macOS/Windows kind users: the playground is your lab. Deep dive behind
> survey §5 of `control-plane-hardening`.

## Everything so far was a lock; this is the alarm

RBAC, admission policy, seccomp, image signing — all *pre-execution*
controls. The cryptominer (M6.2) passed every one of them: it arrived
through a legitimate deploy path with a legitimate token. What would have
caught it in minutes instead of a billing cycle: something watching what
pods *do* — "this container just spawned a shell", "this pod opened a
connection to a mining pool", "something read /etc/shadow".

That's **runtime detection**. Falco (CNCF-graduated; Tetragon and Tracee are
the same idea) watches the syscall stream and matches it against rules.
Detection, not prevention — it pages, it doesn't block (pair it with
Falco Talon or an operator if you want kills).

## Architecture: why it must live on the node

Syscalls happen in the node kernel, so the sensor does too:

```
per node:  kernel ──eBPF probe──▶ falco daemon ──▶ rules engine ──▶ outputs
                                        ▲
                    k8s metadata (pod/ns/labels) via CRI
```

- **Driver**: the modern default is the **eBPF/CO-RE probe** (kernel ≥ 5.8)
  — no kernel module to compile, survives kernel upgrades. The legacy
  `falco.ko` module and older non-CO-RE eBPF still exist for old kernels;
  driver-compile failures on mismatched headers were historically Falco's
  #1 install problem, which CO-RE mostly ended.
- **Deployment**: a DaemonSet — one privileged sensor per node, enriching
  raw syscalls with pod/namespace/image from the CRI so rules can say
  "in a container" at all.

## The rules language in one rule

Rules are YAML: `condition` (syscall filter expression), `output`
(interpolated alert text), `priority`, plus reusable `macro`s and `list`s.
The classic — shell spawned inside a container:

```yaml
- macro: container
  condition: (container.id != host)

- macro: spawned_process
  condition: (evt.type in (execve, execveat) and evt.dir = <)

- list: shell_binaries
  items: [bash, sh, zsh, ash, dash]

- rule: Terminal shell in container
  desc: A shell was spawned inside a container — interactive access or RCE
  condition: >
    spawned_process and container and proc.name in (shell_binaries)
    and proc.tty != 0
  output: >
    Shell in container (user=%user.name container=%container.name
    image=%container.image.repository pod=%k8s.pod.name ns=%k8s.ns.name
    parent=%proc.pname cmdline=%proc.cmdline)
  priority: WARNING
```

Read the condition like a sentence: *an exec completed, inside a container,
the binary is a shell, attached to a TTY.* The `proc.tty != 0` clause is
tuning-in-action — it exempts non-interactive `sh -c` entrypoints, killing
the noisiest false-positive class. (Yes: `kubectl exec -it` — and M2.15's
`kubectl debug` — trip this rule. That's correct behavior; your allowlist
for break-glass debugging is a rules decision, not a sensor gap.)

## Install runbook (Linux node / iximiuz VM)

Helm is upstream's path (values shown pinned); on a bare VM, `apt`/`dnf`
packages exist too:

```sh
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm install falco falcosecurity/falco \
  --namespace falco --create-namespace \
  --set driver.kind=modern_ebpf \
  --set tty=true
kubectl -n falco rollout status ds/falco   # one pod per node
```

Trigger and observe:

```sh
kubectl run alarm-test --image=busybox:1.36 --restart=Never -- sleep 300
kubectl exec -it alarm-test -- sh -c 'id'   # tty-less: quiet
kubectl exec -it alarm-test -- sh           # interactive: alert
kubectl -n falco logs ds/falco | grep "Shell in container"
# 21:14:07.421: Warning Shell in container (user=root container=alarm-test …)
```

Custom rules land via the chart's `customRules:` value or
`/etc/falco/rules.d/` on the node; `falco --validate <file>` lints before
rollout.

## The part that actually fails in production

Not the sensor — the *pipeline after it*. A runtime alert nobody reads
within minutes is a postmortem footnote:

- **Route**: falcosidekick fans out to Slack/PagerDuty/SIEM —
  `--set falcosidekick.enabled=true` in the chart.
- **Tune**: run the stock ruleset in observe-mode for a week; the noisy 10%
  of rules produce 90% of alerts. Tune with `append`/`exceptions`, don't
  delete rules wholesale.
- **Drill**: schedule the `kubectl exec` test quarterly. An alarm you've
  never heard ring is a hypothesis, not a control.

## Takeaway

- Runtime detection is the *only* layer that catches attacks arriving
  through legitimate channels — the M6.2 class.
- eBPF/CO-RE ended the driver-compile era; the sensor is now the easy part.
- Rules are code: version them, lint them, test them with real triggers.
- Detection completes the M6 stack: locks (1–15), evidence (17), alarm
  (here). CKS's runtime-security section is this page.
