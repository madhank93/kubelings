---
kind: challenge

title: Service selector mismatch
description: |
  A web Deployment is running and healthy, but the Service in front of it routes
  traffic nowhere — its label selector matches no pod. Fix the Service so it gets
  ready endpoints again. A kubelings exercise.

categories:
- kubernetes

tagz:
- services
- labels
- selectors
- endpoints

createdAt: 2026-06-29
updatedAt: 2026-06-29

difficulty: easy

# Single-node k8s playground. Swap for k8s-omni (3-node) if you prefer.
playground:
  name: k3s

tasks:
  # Builds the broken scenario. Mirrors
  # exercises/01-services/svc-selector/init.sh — keep them in sync.
  init_scenario:
    init: true
    user: root
    timeout_seconds: 240
    run: |
      set -euo pipefail
      NS=kubelings

      kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -

      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: web
        labels:
          app: web
      spec:
        replicas: 2
        selector:
          matchLabels:
            app: web
        template:
          metadata:
            labels:
              app: web        # <-- the pods are labelled app=web
          spec:
            containers:
              - name: web
                image: nginx:1.27-alpine
                ports:
                  - containerPort: 80
                readinessProbe:
                  httpGet:
                    path: /
                    port: 80
                  initialDelaySeconds: 1
                  periodSeconds: 2
      YAML

      # Broken Service: selector points at app=webserver, which matches no pod.
      kubectl apply -n "$NS" -f - <<'YAML'
      apiVersion: v1
      kind: Service
      metadata:
        name: web
      spec:
        selector:
          app: webserver      # BUG: should be app=web
        ports:
          - port: 80
            targetPort: 80
      YAML

      kubectl -n "$NS" rollout status deploy/web --timeout=180s

  # Passes when the web Service has >=1 ready endpoint. Mirrors
  # exercises/01-services/svc-selector/verify.sh.
  verify_endpoints:
    needs:
      - init_scenario
    run: |
      NS=kubelings
      n="$(kubectl -n "$NS" get endpointslices \
        -l kubernetes.io/service-name=web \
        -o jsonpath='{range .items[*].endpoints[*]}{.conditions.ready}{"\n"}{end}' \
        2>/dev/null | grep -c true || true)"
      if [ "${n:-0}" -ge 1 ]; then
        echo "PASS — Service 'web' has ${n} ready endpoint(s)."
        exit 0
      fi
      echo "Not yet: Service 'web' has no ready endpoints."
      echo "Compare the Service selector with the pod labels:"
      echo "  kubectl -n $NS get pods --show-labels"
      echo "  kubectl -n $NS get svc web -o jsonpath='{.spec.selector}{\"\\n\"}'"
      exit 1
---

A `web` Deployment is running fine — two nginx pods, both Ready. But the `web`
Service in front of them sends traffic **nowhere**: it has no endpoints, so
anything calling `web` gets connection refused.

A Service finds its pods by **label selector**. If the selector doesn't match the
pods' labels, the Service selects nothing.

## Your task

Make the `web` Service route to the `web` pods, in the `kubelings` namespace.

## Inspect

```sh
kubectl -n kubelings get deploy,pods --show-labels
kubectl -n kubelings get svc web -o yaml
kubectl -n kubelings get endpointslices -l kubernetes.io/service-name=web
```

Notice the pods are labelled `app=web`, but the Service selects something else.

## Done when

The `web` Service has at least one **ready endpoint** (its selector matches the
running pods). The check polls for this.

<details>
<summary>Hint</summary>

The pods are `app=web`; the Service selects `app=webserver`. Fix it:

```sh
kubectl -n kubelings patch svc web --type=merge -p '{"spec":{"selector":{"app":"web"}}}'
```

or `kubectl -n kubelings edit svc web` and change `webserver` → `web`.

</details>
