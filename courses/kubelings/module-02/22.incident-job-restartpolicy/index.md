---
kind: lesson
title: 'Incident replay — restartPolicy: Never did not mean never (Universe)'
description: |
  Replay of Universe's cited 2017 outage: a reindex Job was written with
  `restartPolicy: Never` in the belief that it would run once. That field belongs
  to the *pod* — the Job controller kept making new ones, forever, and the greedy
  retries took the nodes and the cluster with them. Bound the failure without
  fixing the crash.
name: incident-job-restartpolicy
slug: incident-job-restartpolicy
source: http://status.universe.com/incidents/115n3vxqwzcf
createdAt: "2026-07-27"
playground:
  name: k8s-omni
tasks:
  init_scenario:
    init: true
    machine: cplane-01
    user: root
    timeout_seconds: 240
    run: |
      set -euo pipefail
      NS=kubelings
      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
      kubectl -n "$NS" delete job search-reindex --ignore-not-found --wait=true
      kubectl -n "$NS" delete pods -l job-name=search-reindex --ignore-not-found >/dev/null 2>&1 || true
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: search-reindex
        labels:
          app: search
      spec:
        # BUG 1: the author read "restartPolicy: Never" as "this will not retry".
        # That field is the POD's. The JOB retries by creating a NEW pod — and
        # backoffLimit here is int32 max, so it does that essentially forever.
        backoffLimit: 2147483647
        # BUG 2: no activeDeadlineSeconds — nothing bounds wall-clock time.
        # BUG 3: no ttlSecondsAfterFinished — the pod graveyard is never collected.
        template:
          metadata:
            labels:
              app: search
              role: reindex
          spec:
            restartPolicy: Never
            containers:
              - name: reindex
                image: busybox:1.36
                # The reindexer dies on the very document it was shipped to fix.
                command:
                  - sh
                  - -c
                  - 'echo "reindexing search corpus…"; sleep 3; echo "FATAL: corrupt document 41c9"; exit 1'
                resources:
                  # BUG 4: requests but no limits — every retry is another
                  # unbounded consumer racing the node it lands on.
                  requests:
                    cpu: 250m
                    memory: 32Mi
      YAML
      # Let the graveyard start filling so the learner sees the pattern, not a theory.
      for _ in $(seq 1 30); do
        n=$(kubectl -n "$NS" get pods -l job-name=search-reindex --no-headers 2>/dev/null | wc -l | tr -d ' ')
        [ "${n:-0}" -ge 2 ] && break
        sleep 3
      done
      kubectl -n "$NS" get job search-reindex
  verify_done:
    needs:
      - init_scenario
    machine: cplane-01
    run: |
      NS=kubelings
      J=search-reindex
      if ! kubectl -n "$NS" get job "$J" >/dev/null 2>&1; then
        echo "not yet: Job '$J' is gone — the fix is a bounded reindex Job, not no reindex Job"; exit 1
      fi
      backoff=$(kubectl -n "$NS" get job "$J" -o jsonpath='{.spec.backoffLimit}' 2>/dev/null)
      if [ -z "$backoff" ] || [ "$backoff" -gt 4 ]; then
        echo "not yet: backoffLimit is ${backoff:-unset} — cap the retries (<= 4). Job spec is immutable: delete and recreate"; exit 1
      fi
      dl=$(kubectl -n "$NS" get job "$J" -o jsonpath='{.spec.activeDeadlineSeconds}' 2>/dev/null)
      if [ -z "$dl" ] || [ "$dl" -gt 300 ]; then
        echo "not yet: activeDeadlineSeconds is ${dl:-unset} — bound wall-clock time too (<= 300)"; exit 1
      fi
      ttl=$(kubectl -n "$NS" get job "$J" -o jsonpath='{.spec.ttlSecondsAfterFinished}' 2>/dev/null)
      if [ -z "$ttl" ]; then
        echo "not yet: ttlSecondsAfterFinished unset — finished Jobs and their pods must collect themselves"; exit 1
      fi
      lcpu=$(kubectl -n "$NS" get job "$J" -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}' 2>/dev/null)
      lmem=$(kubectl -n "$NS" get job "$J" -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}' 2>/dev/null)
      if [ -z "$lcpu" ] || [ -z "$lmem" ]; then
        echo "not yet: the retry container still has no cpu/memory limits — a greedy retry must not be able to take the node"; exit 1
      fi
      # Terminal state: the Job must give up on its own. Poll — retries take a moment.
      state=""
      for _ in $(seq 1 40); do
        failed=$(kubectl -n "$NS" get job "$J" -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' 2>/dev/null)
        done_=$(kubectl -n "$NS" get job "$J" -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null)
        [ "$failed" = "True" ] && { state=failed; break; }
        [ "$done_" = "True" ] && { state=complete; break; }
        sleep 5
      done
      if [ "$state" = "complete" ]; then
        echo "not yet: the Job succeeded — this drill is about bounding a job that KEEPS failing. Keep the failing reindex command; fix the blast radius, not the exit code"; exit 1
      fi
      if [ "$state" != "failed" ]; then
        echo "not yet: Job has not reached a terminal state — with a real backoffLimit/deadline it must give up by itself"; exit 1
      fi
      pods=$(kubectl -n "$NS" get pods -l job-name="$J" --no-headers 2>/dev/null | wc -l | tr -d ' ')
      max=$((backoff + 1))
      if [ "${pods:-0}" -gt "$max" ]; then
        echo "not yet: $pods pods carry job-name=$J but the bounded Job allows at most $max — clear the graveyard the old Job left behind"; exit 1
      fi
      reason=$(kubectl -n "$NS" get job "$J" -o jsonpath='{.status.conditions[?(@.type=="Failed")].reason}' 2>/dev/null)
      echo "PASS — the reindexer still can't index the corrupt document, and that is now a bounded failure (${reason:-Failed}), not a cluster-wide one."
---
