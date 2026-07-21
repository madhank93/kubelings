> **☁ iximiuz Labs only.** Wiring encryption means editing the kube-apiserver
> static-pod manifest as root on a control-plane node — outside the kubectl
> sandbox (M6.8 explains the confinement), so this one can't run on your local
> `kind` cluster. Here it runs on a real, disposable control-plane VM: read the
> runbook, then do it for real at the bottom. This is the deep dive behind
> survey §1 of `control-plane-hardening`, and etcd paths follow M7.6's backup
> runbook.

## The threat, restated in one command

On an unencrypted cluster, from a control-plane node:

```sh
ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  get /registry/secrets/kubelings/db-creds | hexdump -C | head
# …  s3cure-NEW-9917  ← your Secret, plaintext, in every etcd snapshot ever taken
```

RBAC guards the API; it does not guard the *disk*. Every backup (M7.6) is a
copy of every Secret unless the apiserver encrypts before writing.

## 1 · The EncryptionConfiguration file

`/etc/kubernetes/enc/enc.yaml` on each control-plane node (mode 0600,
root-only):

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources: [secrets]
    providers:
      - aescbc:
          keys:
            - name: key1
              secret: <base64 of 32 random bytes>   # head -c 32 /dev/urandom | base64
      - identity: {}
```

**Provider order is the whole semantics:**

- **writes** always use the *first* provider — new Secrets become ciphertext
- **reads** try providers *in order* — `identity: {}` last means old
  plaintext Secrets stay readable during migration

Get the order wrong (`identity` first) and you've deployed encryption that
encrypts nothing — writes go through `identity`, i.e. plaintext.

Provider choice: `aescbc`/`secretbox` keep the key **on the node** — etcd
backups are now safe, but the node's filesystem is the new prize. **KMS v2**
is the production answer: envelope encryption, key lives in cloud
KMS/Vault, apiserver holds only a connection to a KMS plugin socket.

## 2 · Wire the apiserver

Edit `/etc/kubernetes/manifests/kube-apiserver.yaml` (static pod — the
kubelet restarts it on save, ~30s of API unavailability per node):

```yaml
spec:
  containers:
    - command:
        - kube-apiserver
        - --encryption-provider-config=/etc/kubernetes/enc/enc.yaml
      volumeMounts:
        - {name: enc, mountPath: /etc/kubernetes/enc, readOnly: true}
  volumes:
    - name: enc
      hostPath: {path: /etc/kubernetes/enc, type: DirectoryOrCreate}
```

Watch it come back: `crictl ps | grep apiserver`, then `kubectl get
--raw=/readyz`. On HA clusters (M7.11-to-be), roll one control-plane node at
a time — same file, same key, every node.

## 3 · Prove it

```sh
kubectl -n kubelings create secret generic enc-canary --from-literal=k=v
ETCDCTL_API=3 etcdctl … get /registry/secrets/kubelings/enc-canary | hexdump -C | head
# 00000000  2f 72 65 67 …  k8s:enc:aescbc:v1:key1:…  ← ciphertext + header
```

The `k8s:enc:aescbc:v1:key1` prefix names the provider and key that
encrypted this row — that's how reads pick a decryption path, and how you
audit migration progress.

## 4 · The migration everyone forgets

Enabling encryption touches **new writes only**. Existing Secrets sit in
plaintext until rewritten:

```sh
kubectl get secrets -A -o json | kubectl replace -f -
```

Run it after every key change too. Audit completeness by scanning etcd for
rows missing the `k8s:enc:` prefix.

## 5 · Key rotation (the lockout-free order)

1. Add `key2` **behind** `key1` (reads work for both, writes still `key1`);
   restart all apiservers.
2. Move `key2` first (writes now `key2`); restart all apiservers.
3. `kubectl get secrets -A -o json | kubectl replace -f -` — everything
   re-encrypts under `key2`.
4. Remove `key1`. Restart. Done.

Skip a step on an HA cluster and an apiserver that only knows `key1` meets a
Secret written with `key2`: unreadable Secrets and a very bad afternoon.
**Never** remove a key that any row in etcd might still be encrypted with —
that's cryptographic data loss, no force-flag can help.

## Takeaway

- Provider *order* is policy: first = write key, list = read set, `identity`
  last during migration only.
- Local keys (`aescbc`) move the secret from etcd to the node; KMS v2 moves
  it off the machine entirely — prefer it anywhere real.
- Three commands to memorize: the etcdctl spot-check, the `replace -f -`
  migration, the prefix audit.
- CKS asks exactly this flow: config file → apiserver flag → etcdctl proof
  → migration. Do it once for real and it's yours.

## Your turn

`init` created **`kubelings/db-creds`** with the password `s3cure-NEW-9917`,
and showed you that string sitting in etcd in the clear.

Encrypt it at rest, on this control plane:

1. Write `/etc/kubernetes/enc/enc.yaml` with an `aescbc` provider first and
   `identity: {}` last (§1).
2. Wire `--encryption-provider-config` into the kube-apiserver static pod,
   with the hostPath volume and mount (§2). Wait for the apiserver to come back.
3. Re-encrypt what already exists (§4) — enabling encryption does **not**
   rewrite old rows.

The check requires all three: the Secret must still read back as
`s3cure-NEW-9917` through the API, **and** its raw etcd row must carry the
`k8s:enc:` prefix with no plaintext left in it.
