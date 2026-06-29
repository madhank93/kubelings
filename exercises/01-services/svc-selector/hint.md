# Hints

1. A Service routes to pods whose labels match its `spec.selector`. If the
   Service has no endpoints, the selector matches no pod.

2. Look at both sides:
   ```sh
   kubectl -n kubelings get pods --show-labels
   kubectl -n kubelings get svc web -o jsonpath='{.spec.selector}{"\n"}'
   ```
   The pods are `app=web`; the Service selects `app=webserver`.

3. Fix the Service's selector to match the pods:
   ```sh
   kubectl -n kubelings patch svc web --type=merge -p '{"spec":{"selector":{"app":"web"}}}'
   ```
   or `kubectl -n kubelings edit svc web` and change `webserver` → `web`,
   or apply `solution.yaml`.
