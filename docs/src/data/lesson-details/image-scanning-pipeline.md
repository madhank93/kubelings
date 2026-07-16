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
