## The situation

The new API gateway should have been serving ten minutes ago:

```
NAME                       READY   STATUS              RESTARTS   AGE
gateway-5f8d9b7c6d-mq2xk   0/1     ContainerCreating   0          10m
```

`ContainerCreating` for a few seconds is normal (image pull, volume setup). Ten
*minutes* means a volume can't be assembled. Events tell you which one:

```sh
kubectl -n kubelings describe pod -l app=gateway | tail -6
```

```
MountVolume.SetUp failed for volume "tls" :
  secret "gateway-tls" not found
```

There *is* a TLS secret in the namespace — the cert-rotation job faithfully
maintains `gateway-tls-cert`. The Deployment asks for `gateway-tls`. Two teams,
two naming conventions, zero communication.

Note the design: Kubernetes **refuses to start the container at all** rather
than start it without its Secret. A gateway that boots without TLS material and
serves plaintext would be a much worse failure than a pod that visibly never
starts. The hang is the safety feature.

## Your task

Get `gateway` Running with the cert actually mounted at `/etc/tls/`:

```sh
kubectl -n kubelings get secrets
kubectl -n kubelings get deploy gateway -o jsonpath='{.spec.template.spec.volumes}'
```

Fix from whichever side is right — but the rotation job *rewrites*
`gateway-tls-cert` on every rotation, so renaming the Secret means fighting a
robot forever. Fix the reference.
