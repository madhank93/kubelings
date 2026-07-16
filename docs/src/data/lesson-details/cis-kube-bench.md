## The situation

Everything so far in this module hardened *workloads*. The pen-test report's
first section, though, will be about the *platform*: is the API server
accepting anonymous requests? Are kubelet ports open? Are etcd's files
world-readable? The industry checklist for those questions is the
**CIS Kubernetes Benchmark** — a few hundred audited controls for every
control-plane and node component — and **kube-bench** is the tool that runs
it *as a pod on the cluster it's auditing*.

Why a pod can audit the node at all: the checks read component config files
and process flags, so the Job mounts the node's config directories
(`hostPath`, read-only) and runs with the node's PID namespace. That's a lot
of trust — which is itself the first lesson: **an auditor pod is exactly the
shape of pod your Pod Security policy (6.4) exists to block.** It runs here
because *you*, the admin, deliberately grant it.

## Your task

Run the benchmark as a Job in `kubelings` and read its verdicts:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: kube-bench
  namespace: kubelings
spec:
  backoffLimit: 1
  template:
    spec:
      hostPID: true
      restartPolicy: Never
      containers:
        - name: kube-bench
          image: docker.io/aquasec/kube-bench:latest
          command: ["kube-bench"]
          volumeMounts:
            - {name: var-lib-kubelet, mountPath: /var/lib/kubelet, readOnly: true}
            - {name: etc-systemd, mountPath: /etc/systemd, readOnly: true}
            - {name: etc-kubernetes, mountPath: /etc/kubernetes, readOnly: true}
            - {name: usr-bin, mountPath: /usr/local/mount-from-host/bin, readOnly: true}
      volumes:
        - {name: var-lib-kubelet, hostPath: {path: /var/lib/kubelet}}
        - {name: etc-systemd, hostPath: {path: /etc/systemd}}
        - {name: etc-kubernetes, hostPath: {path: /etc/kubernetes}}
        - {name: usr-bin, hostPath: {path: /usr/bin}}
      tolerations:
        - {key: node-role.kubernetes.io/control-plane, operator: Exists, effect: NoSchedule}
```

Save as `kube-bench-job.yaml`, apply, wait, read:

```sh
kubectl apply -f kube-bench-job.yaml
kubectl -n kubelings wait --for=condition=complete job/kube-bench --timeout=180s
kubectl -n kubelings logs job/kube-bench | less     # the actual deliverable
```
