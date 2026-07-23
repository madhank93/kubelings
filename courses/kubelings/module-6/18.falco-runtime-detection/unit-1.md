---
kind: unit
title: "Falco: alarms for the attack you didn't prevent"
name: falco-runtime-detection-unit
---


> **☁ iximiuz Labs only.** Falco loads an eBPF probe into each node's kernel —
> host-level by nature, outside the kubectl sandbox (M6.8's confinement), so
> this one can't run on your local `kind` cluster. Here you get a real node
> (kernel ≥ 5.8 for the modern eBPF probe): read the runbook, then run the
> drill at the bottom — `init` installs Falco with your rule loaded, and you
> make it fire for real. Deep dive behind survey §5 of `control-plane-hardening`.

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

## Your turn

`init` did the install for you: Falco is running as a DaemonSet with the
`modern_ebpf` driver. Its **default ruleset already ships the exact rule you
dissected above** — "shell spawned in a container with an attached terminal",
same `container` + `shell_binaries` + `proc.tty != 0` logic. A throwaway
container, `default/alarm-test`, is waiting.

Your job: make that rule fire, for real.

1. Confirm the sensor is up: `kubectl -n falco rollout status ds/falco`.
2. Get an **interactive** shell inside `alarm-test`.
3. Run something, exit, then read Falco's log and find the alert.

The check passes once a shell-in-container alert naming `alarm-test` has
appeared since `init`.

<details>
<summary>Hint</summary>

The whole trap is `proc.tty != 0` in the condition. It's there on purpose —
it exempts non-interactive `sh -c` entrypoints, the noisiest false positives.
So the exec everyone reaches for first stays *silent*:

```sh
kubectl exec alarm-test -- sh -c 'id'       # no controlling tty -> no alert
```

You need a real shell session with a tty attached:

```sh
kubectl exec -it alarm-test -- sh           # now proc.tty != 0 -> fires
```

Read the alert out of the daemon's stdout — one line per sensor pod, so
aggregate across the DaemonSet:

```sh
kubectl -n falco logs -l app.kubernetes.io/name=falco --tail=-1 \
  | grep alarm-test
```

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
# 1 · the sensor must be up first — it can't see what happened before it did
kubectl -n falco rollout status ds/falco

# 2 · a real interactive shell inside the container (tty attached)
kubectl exec -it alarm-test -- sh
#   ~ $ id
#   ~ $ exit

# 3 · read the alert back out of Falco's log
kubectl -n falco logs -l app.kubernetes.io/name=falco --tail=-1 \
  | grep alarm-test
# Notice A shell was spawned in a container with an attached terminal
#   (evt_type=execve process=sh terminal=34816 container_name=alarm-test
#   k8s_pod_name=alarm-test k8s_ns_name=default)
```

## Root cause, restated

Nothing here was *prevented* — that's the entire point of the layer. The
shell was a completely legitimate syscall on a completely legitimate pod;
no admission rule, seccomp profile, or image signature had any opinion about
it. Only something watching the running syscall stream could say "a human
just opened a shell in a production container" — which is precisely the M6.2
cryptominer class, the attack that arrives through legitimate channels.

And the tty clause is the lesson inside the lesson: a runtime rule is only as
good as its tuning. Too loose and every `sh -c` entrypoint pages you at 3
a.m.; too tight and `kubectl exec -it` slips through. `proc.tty != 0` is
someone's deliberate call about where that line sits.

</details>
