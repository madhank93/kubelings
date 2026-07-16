## Welcome 👋

Kubelings teaches Kubernetes the way *rustlings* teaches Rust: a series of small,
broken-on-purpose scenarios you fix one at a time.

### The loop

1. **Read the situation** — each lesson explains what's broken (or what to build)
   and what "done" looks like.
2. **Fix the cluster** — you get a live, multi-node Kubernetes cluster in the tabs
   on the right. Use `kubectl` to diagnose and repair.
3. **Watch the check** — every lesson has an automated check. It turns green only
   when the cluster is genuinely in the target state. Re-run after each change.

### Tips

- Each lesson runs in its **own** fresh cluster — you can't break anything that
  matters, and "reset" just means reloading the scenario.
- Prefer inspecting before changing: `kubectl get`, `describe`, `logs`, `events`.
- Stuck? Most lessons include a collapsible **Hint** and a full solution.

### Run it locally too

Every scenario is portable. The project repo ships a `kind`-based runner so you
can practice offline:

```sh
scripts/run-challenge-local.sh up
scripts/run-challenge-local.sh <challenge> init
scripts/run-challenge-local.sh <challenge> verify
```

Ready? Head to **Module 2 — Workloads & Scheduling** and fix your first cluster.
