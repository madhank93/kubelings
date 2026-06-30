# Solution — make the Job complete

## Root cause

The Job's container ran `sleep infinity`. A Job only reports `Complete` when its
pod's container **exits 0** — a process that never exits keeps the Job `Active`
forever.

## Fix

Job pod templates are immutable, so delete and recreate with a command that
finishes:

```sh
kubectl -n kubelings delete job data-import

kubectl -n kubelings apply -f - <<'EOF'
apiVersion: batch/v1
kind: Job
metadata:
  name: data-import
spec:
  backoffLimit: 4
  template:
    spec:
      restartPolicy: OnFailure
      containers:
        - name: importer
          image: ghcr.io/iximiuz/labs/busybox:latest
          command: ["sh","-c","echo importing...; sleep 2; echo done"]
EOF
```

## Verify

```sh
kubectl -n kubelings wait --for=condition=complete job/data-import --timeout=60s
kubectl -n kubelings get job data-import
```
