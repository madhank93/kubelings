# Solution — HorizontalPodAutoscaler

## Approach

An HPA needs (a) metrics-server (already running) and (b) a CPU **request** on the
target pods to compute utilization against — `php-apache` has `requests.cpu: 200m`.

## Create the HPA

Imperative (simplest):

```sh
kubectl -n kubelings autoscale deploy php-apache --cpu=50% --min=1 --max=5
```

or declarative (`autoscaling/v2`):

```sh
kubectl -n kubelings apply -f - <<'EOF'
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: php-apache
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: php-apache
  minReplicas: 1
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 50
EOF
```

## Verify

```sh
kubectl -n kubelings get hpa php-apache
# generate load:
kubectl -n kubelings run load --rm -it --image=ghcr.io/iximiuz/labs/busybox:latest -- \
  sh -c "while true; do wget -q -O- http://php-apache; done"
kubectl -n kubelings get hpa php-apache -w   # replicas climb toward 5
```
