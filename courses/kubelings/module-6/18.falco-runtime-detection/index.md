---
kind: lesson
title: 'Falco: alarms for the attack you didn''t prevent'
description: |
  Runtime detection with Falco: how the eBPF probe sees every syscall, the
  rules language, writing the shell-in-a-container rule, and the install
  runbook. Then run it for real — install the DaemonSet on a live node,
  spawn an interactive shell in a container, and prove your rule actually
  fired an alert (and that the tty-less exec everyone tries first does not).
name: falco-runtime-detection
slug: falco-runtime-detection
createdAt: "2026-07-13"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 360
    run: |
      set -euo pipefail
      BASE=/etc/kubernetes/kubelings-falco-baseline

      export KUBECONFIG=/etc/kubernetes/admin.conf

      # A custom rule with a unique marker in its output, so the verify can
      # grep for exactly THIS rule firing — not some unrelated default alert.
      cat >/tmp/kubelings-falco-rule.yaml <<'RULE'
      - rule: Kubelings shell in container
        desc: A shell was spawned inside a container with a controlling TTY
        condition: >
          spawned_process and container
          and proc.name in (shell_binaries) and proc.tty != 0
        output: >
          KUBELINGS-ALERT shell in container
          (pod=%k8s.pod.name ns=%k8s.ns.name container=%container.name
          proc=%proc.name parent=%proc.pname)
        priority: WARNING
      RULE

      echo "Installing Falco (modern eBPF) as a DaemonSet — this pulls the"
      echo "probe and rolls out one privileged sensor per node, ~1-2 min..."
      helm repo add falcosecurity https://falcosecurity.github.io/charts >/dev/null 2>&1 || true
      helm repo update >/dev/null 2>&1 || true
      helm install falco falcosecurity/falco \
        --namespace falco --create-namespace \
        --set driver.kind=modern_ebpf \
        --set tty=true \
        --set-file "customRules.kubelings-shell\.yaml"=/tmp/kubelings-falco-rule.yaml \
        >/dev/null

      kubectl -n falco rollout status ds/falco --timeout=300s

      # A container for the learner to exec into.
      kubectl run alarm-test --image=busybox:1.36 --restart=Never \
        -- sleep 3600 >/dev/null 2>&1 || true
      kubectl wait --for=condition=Ready pod/alarm-test --timeout=60s >/dev/null 2>&1 || true

      # Baseline: how many of our alerts exist right now (zero — nobody has
      # exec'd yet). The check passes only once this count goes up.
      base_count="$(kubectl -n falco logs -l app.kubernetes.io/name=falco \
        --tail=-1 --prefix 2>/dev/null | grep -c 'KUBELINGS-ALERT' || true)"
      printf 'ALERT_BASELINE=%s\n' "${base_count:-0}" >"$BASE"
      chmod 600 "$BASE"

      echo
      echo "Falco is up, your custom rule is loaded, and pod default/alarm-test"
      echo "is waiting. KUBELINGS-ALERT count so far: ${base_count:-0}."
      echo
      echo "Make Falco fire your rule by getting a real interactive shell inside"
      echo "the container. (baseline in $BASE — don't edit it, the check reads it)"
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    user: root
    run: |
      BASE=/etc/kubernetes/kubelings-falco-baseline
      export KUBECONFIG=/etc/kubernetes/admin.conf

      if [ ! -s "$BASE" ]; then
        echo "not yet: baseline file $BASE is missing — re-run init for this lesson."
        exit 1
      fi
      # shellcheck disable=SC1090
      . "$BASE"

      if ! kubectl -n falco rollout status ds/falco --timeout=10s >/dev/null 2>&1; then
        echo "not yet: the Falco DaemonSet isn't Ready — the sensor has to be up"
        echo "         before it can see anything. kubectl -n falco get pods"
        exit 1
      fi

      now_count="$(kubectl -n falco logs -l app.kubernetes.io/name=falco \
        --tail=-1 --prefix 2>/dev/null | grep -c 'KUBELINGS-ALERT' || true)"

      if [ "${now_count:-0}" -le "${ALERT_BASELINE:-0}" ]; then
        echo "not yet: no shell-in-container alert has fired since init."
        echo
        echo "         The most common miss: a tty-less exec. Your rule has"
        echo "         'proc.tty != 0', so:"
        echo "           kubectl exec -it alarm-test -- sh -c 'id'   # NO tty -> quiet"
        echo "           kubectl exec -it alarm-test -- sh           # real shell -> fires"
        echo "         Run the second one, type a command, exit, then check again."
        exit 1
      fi

      echo "PASS — your rule fired. Falco saw the interactive shell and logged it:"
      echo
      kubectl -n falco logs -l app.kubernetes.io/name=falco --tail=-1 2>/dev/null \
        | grep 'KUBELINGS-ALERT' | tail -3
---
