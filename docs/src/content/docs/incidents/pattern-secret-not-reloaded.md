---
title: "Pattern: Secret rotated, never reloaded"
description: "[PATTERN] Synthetic composite — a rotated Secret never reaches the running pod because env vars are resolved once at startup; the outage fires later, at revocation."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/reference/incident-library/).)

## Situation

Security rotates a database password. The Secret object shows the new value;
every dashboard is green. Days later the *old* credential is revoked on
schedule — and services start failing auth. The postmortem is confusing by
design: the outage is far from the change, and the Secret has been "correct"
the whole time.

## Root cause

The consuming pods received the credential as an **environment variable**
(`env.valueFrom.secretKeyRef`). Env vars are resolved once, at container
start, and copied into the process environment — no rotation ever reaches a
running process. The pods needed a restart at rotation time, and nothing in
the runbook said so.

The subtle variant: the Secret *is* volume-mounted, but via `subPath` —
subPath copies are also frozen at pod start.

## Diagnosis

```sh
# how do consumers inject it?
kubectl get pods -A -o json | jq -r '
  .items[] | select(.spec.containers[].env[]?.valueFrom.secretKeyRef.name=="<secret>")
  | .metadata.namespace + "/" + .metadata.name'
# live value vs Secret value:
kubectl exec <pod> -- sh -c 'echo $DB_PASSWORD'
kubectl get secret <secret> -o jsonpath='{.data.password}' | base64 -d
```

## Fix

Mount the Secret as a volume (no `subPath`) and read the file:

```yaml
containers:
  - name: app
    volumeMounts:
      - {name: creds, mountPath: /etc/db-creds, readOnly: true}
volumes:
  - name: creds
    secret: {secretName: db-creds}
```

The kubelet re-syncs mounted Secrets within ~a minute of a change and updates
the file in place — provided the app re-reads it per connection instead of
caching at boot.

| Injection | Rotation behavior |
|---|---|
| env var | frozen at start — restart required |
| volume mount | auto-syncs in ~1 min |
| volume mount + `subPath` | frozen — the classic surprise |

## Prevention

- Policy: credentials via volume mounts, never env — enforceable with
  admission policy (Kyverno/Gatekeeper, Kubelings M6).
- If env injection must stay, rotation runbooks get a mandatory
  `kubectl rollout restart` of every consumer — and an owned inventory of
  consumers.
- Verify rotations end-to-end: check the value *inside* a pod, not the
  Secret object.

## What it teaches

| Concept | Kubelings module |
|---|---|
| Secret injection modes & reload semantics | M3 (`pattern-secret-not-reloaded`) |
| Enforcing injection policy at admission | M6 Security |
