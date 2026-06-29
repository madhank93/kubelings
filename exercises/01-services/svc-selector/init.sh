#!/usr/bin/env bash
# Creates the broken scenario: a healthy Deployment + a Service whose selector
# does not match the pods, so the Service has no endpoints.
set -euo pipefail

NS="${KUBELINGS_NS:-kubelings}"

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

# Wait for the pods to be Ready so the ONLY remaining fault is the selector.
kubectl -n "$NS" rollout status deploy/web --timeout=120s

echo
echo "Scenario ready in namespace '$NS'."
echo "The 'web' Service currently has no endpoints. Read task.md and fix it."
