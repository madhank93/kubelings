---
kind: unit
title: "Scan it, then pin it: trivy and the digest"
name: image-scanning-pipeline-unit
---


## The situation

`legacy-api` runs `nginx:1.14` — released 2018, EOL 2019. Ask trivy what
that means (init installed it, pinned at v0.72.0):

```sh
trivy image --severity HIGH,CRITICAL nginx:1.14
```

The first scan downloads trivy's vulnerability DB (~a minute), then prints a
wall of CVEs — the OS packages *and* the nginx binary itself. Two distinct
problems are stacked here:

1. **The image is vulnerable** — known, catalogued, patched-upstream CVEs.
2. **The reference is mutable** — `nginx:1.14` is a tag, a movable pointer.
   Even if someone re-pushed a patched build to that tag, you couldn't tell
   the difference from a malicious swap. M6.10 (`image-digests`) taught the
   fix: `@sha256:` digests name *bytes*, not intentions.

## Your task

1. **Scan the incumbent** — see for yourself:

   ```sh
   trivy image --severity CRITICAL nginx:1.14
   ```

2. **Find a clean replacement** — scan candidates until one has **zero
   CRITICAL** findings (recent alpine-based nginx tags are good hunting
   ground):

   ```sh
   trivy image --severity CRITICAL --exit-code 1 nginx:1.29-alpine && echo CLEAN
   ```

3. **Resolve its digest.** No docker CLI needed — run it and ask the kubelet
   what it actually pulled:

   ```sh
   kubectl -n kubelings set image deploy/legacy-api legacy-api=nginx:1.29-alpine
   kubectl -n kubelings rollout status deploy/legacy-api
   kubectl -n kubelings get pods -l app=legacy-api --sort-by=.metadata.creationTimestamp \
     -o jsonpath='{range .items[*]}{.status.containerStatuses[0].imageID}{"\n"}{end}' | tail -1
   # docker.io/library/nginx@sha256:…
   ```

4. **Pin it** — set the image to that full `repo@sha256:…` reference and let
   it roll out.

The check requires: image reference contains `@sha256:`, rollout complete,
and `trivy image --severity CRITICAL --exit-code 1 <pinned>` passing.

<details>
<summary>Hint</summary>

```sh
DIGEST=$(kubectl -n kubelings get pods -l app=legacy-api --sort-by=.metadata.creationTimestamp \
  -o jsonpath='{range .items[*]}{.status.containerStatuses[0].imageID}{"\n"}{end}' \
  | tail -1 | sed 's|^docker.io/library/||')
kubectl -n kubelings set image deploy/legacy-api legacy-api="$DIGEST"
kubectl -n kubelings rollout status deploy/legacy-api
```

The sort + `tail -1` matters: right after a rollout the old pod may still be
Terminating, and grabbing `.items[0]` can hand you the *old* image's digest.

If your chosen tag still shows CRITICALs (images rot — what was clean last
year may not be today), try a newer tag and re-scan. That rot is itself the
lesson.

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


## Fix

```sh
trivy image --severity CRITICAL nginx:1.14                      # the horror show
trivy image --severity CRITICAL --exit-code 1 nginx:1.29-alpine # candidate gate
kubectl -n kubelings set image deploy/legacy-api legacy-api=nginx:1.29-alpine
kubectl -n kubelings rollout status deploy/legacy-api
DIGEST=$(kubectl -n kubelings get pods -l app=legacy-api --sort-by=.metadata.creationTimestamp \
  -o jsonpath='{range .items[*]}{.status.containerStatuses[0].imageID}{"\n"}{end}' \
  | tail -1 | sed 's|^docker.io/library/||')
kubectl -n kubelings set image deploy/legacy-api legacy-api="$DIGEST"
kubectl -n kubelings rollout status deploy/legacy-api
```

## Why both halves matter

- **Scan without pin**: your clean image's tag gets re-pushed tomorrow with
  different bytes — scan result void, and you can't even detect it happened.
- **Pin without scan**: you've immutably locked in the vulnerabilities.
- **Scan then pin**: the digest *is* the scan's receipt — those exact bytes
  had zero CRITICALs at scan time.

"At scan time" is the catch: new CVEs are published against old bytes
constantly. Pinning freezes the image, not the world — which is why scanning
belongs in a *pipeline* (CI gate + periodic re-scan of what's running), not
a one-off heroic audit.

## Prevention / takeaway

- CI gate: `trivy image --exit-code 1 --severity HIGH,CRITICAL $IMG` fails
  the build — cheapest possible supply-chain control.
- Admission tie-in: M6.11/6.12 engines can require digest references
  cluster-wide (Kyverno even has a `verifyImages` rule that checks
  signatures — next lesson).
- Re-scan the *running* estate on a schedule; trivy also has `trivy k8s` for
  exactly that.
- `nginx:1.14` in prod is not hypothetical — base-image rot is how most
  real clusters accumulate their CVE backlog: nobody chose the
  vulnerabilities, they just stopped choosing updates.

</details>
