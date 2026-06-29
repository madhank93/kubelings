#!/usr/bin/env bash
# Passes when the 'web' Service has at least one READY endpoint — i.e. its
# selector matches the running pods. Polls, because endpoints are populated
# asynchronously after a selector change (k8s is eventually consistent).
set -uo pipefail

NS="${KUBELINGS_NS:-kubelings}"
TIMEOUT="${KUBELINGS_TIMEOUT:-60}"
deadline=$(( $(date +%s) + TIMEOUT ))

ready_endpoints() {
  # grep -c prints the count and exits non-zero on 0 matches; `|| true` keeps
  # the function quiet and returns just the number.
  kubectl -n "$NS" get endpointslices \
    -l "kubernetes.io/service-name=web" \
    -o jsonpath='{range .items[*].endpoints[*]}{.conditions.ready}{"\n"}{end}' \
    2>/dev/null | grep -c "true" || true
}

while :; do
  n="$(ready_endpoints)"
  if [ "${n:-0}" -ge 1 ]; then
    echo "PASS ✅  Service 'web' has ${n} ready endpoint(s) — traffic will reach the pods."
    exit 0
  fi
  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "FAIL ❌  Service 'web' still has no ready endpoints after ${TIMEOUT}s."
    echo "Hint: compare the Service selector with the pod labels:"
    echo "  kubectl -n $NS get pods --show-labels"
    echo "  kubectl -n $NS get svc web -o jsonpath='{.spec.selector}{\"\\n\"}'"
    exit 1
  fi
  sleep 2
done
