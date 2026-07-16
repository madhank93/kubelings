> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters, not a specific company's incident. The full pattern

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
