---
kind: unit
title: "Drill — the Secret that was rotated but never reloaded"
name: pattern-secret-not-reloaded-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern
> write-up: [Pattern: Secret rotated, never reloaded](https://kubelings.madhan.app/incidents/pattern-secret-not-reloaded/).

## The situation

Security rotated the database password an hour ago. The Secret is correct:

```sh
kubectl -n kubelings get secret db-creds -o jsonpath='{.data.password}' | base64 -d
# s3cure-NEW-9917
```

The `billing` pod disagrees:

```sh
kubectl -n kubelings logs billing --tail=1
# connecting to db with password=hunter2-2024Q4
```

Same cluster, same Secret, two different truths. The pod spec explains it:

```sh
kubectl -n kubelings get pod billing -o jsonpath='{.spec.containers[0].env}'
```

`DB_PASSWORD` comes from `secretKeyRef` — an **env var**. Env vars are
resolved **once, when the container starts**, copied into the process
environment, and never touched again. The Secret can rotate hourly; the
process keeps the value it was born with. (The old password gets revoked
next, and *then* billing goes down — at the revocation, not the rotation,
which makes the postmortem extra confusing.)

Mounted Secret **volumes** behave differently: the kubelet re-syncs them
(~1 min), and the file content updates in place, no restart.

## Your task

Recreate `billing` consuming `db-creds` as a **volume mount** instead of env:

```yaml
containers:
  - name: billing
    volumeMounts:
      - {name: creds, mountPath: /etc/db-creds, readOnly: true}
volumes:
  - name: creds
    secret: {secretName: db-creds}
```

Have the app read the file (`cat /etc/db-creds/password`) instead of
`$DB_PASSWORD`. Pod env is immutable — this is a delete-and-recreate
(`kubectl get pod billing -o yaml` → edit → `kubectl replace --force`), or
write the pod fresh.

<details>
<summary>Hint</summary>

Minimal fixed pod:

```sh
kubectl -n kubelings delete pod billing
kubectl apply -n kubelings -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: billing
  labels: {app: billing}
spec:
  containers:
    - name: billing
      image: busybox:1.36
      command: ["sh", "-c", "while true; do echo \"connecting to db with password=$(cat /etc/db-creds/password)\"; sleep 15; done"]
      volumeMounts:
        - {name: creds, mountPath: /etc/db-creds, readOnly: true}
  volumes:
    - name: creds
      secret: {secretName: db-creds}
EOF
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


## The pattern (why this recurs everywhere)

Secret rotation programs assume the platform delivers new values; env-var
injection silently breaks that assumption. The failure is *delayed*: rotation
succeeds, dashboards stay green, and the outage fires days later when the old
credential is revoked — far from the change that caused it. Audits miss it
because the Secret object *is* up to date.

## Fix

Recreate the pod with the Secret as a volume (hint above). Verify the live
value:

```sh
kubectl -n kubelings exec billing -- cat /etc/db-creds/password
# s3cure-NEW-9917
kubectl -n kubelings logs billing --tail=1
```

## The mechanics worth remembering

| Injection | Rotation behavior |
|---|---|
| `env.valueFrom.secretKeyRef` | frozen at container start — restart required |
| volume mount | kubelet syncs within ~1 min, file updates in place |
| volume mount with `subPath` | **frozen** — subPath copies don't update; the classic surprise |

The app must also *re-read* the file per connection (or watch it) — a
process that reads once at boot recreates the env-var problem one layer up.

## Prevention / takeaway

- House rule: credentials via volume mounts, never env — enforceable with
  admission policy (M6's Kyverno lesson can express exactly this).
- If env injection is unavoidable, rotation runbooks must include a rollout
  restart: `kubectl rollout restart deploy/<consumer>` — and an inventory of
  consumers, which is the hard part.
- M3.2 (`secret-not-mounted`) covered Secrets that never arrive; this drill
  is the subtler sibling — Secrets that arrive but never *update*.

</details>
