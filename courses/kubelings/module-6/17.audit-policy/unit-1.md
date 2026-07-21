---
kind: unit
title: "Audit policy: who touched that Secret?"
name: audit-policy-unit
---


> **☁ iximiuz Labs only.** Audit policy is a kube-apiserver flag plus a file on
> the control-plane node — host-level wiring, outside the kubectl sandbox
> (M6.8's confinement note), so this one can't run on your local `kind`
> cluster. Here it runs on a real, disposable control-plane VM: read the
> runbook, then wire it for real at the bottom. Deep dive behind survey §2 of
> `control-plane-hardening`.

## The question you can't answer right now

The cryptominer incident (M6.2) ended with a compromised ServiceAccount
token. The response's first question — *what did that token read?* — has no
answer on a default cluster: **the apiserver logs nothing about requests.**
No record of who read which Secret, who exec'd into which pod, who deleted
the namespace. Audit logging is opt-in, and the opt-in is a policy file.

## 1 · The policy language: four levels, first match wins

```yaml
apiVersion: audit.k8s.io/v1
kind: Policy
```

| Level | Records | Cost |
|---|---|---|
| `None` | nothing | free — this is your noise filter |
| `Metadata` | who, what, when, verb, resource — not the payload | cheap; the default workhorse |
| `Request` | Metadata + the request body | payloads can hold secrets — careful |
| `RequestResponse` | both bodies | expensive; reserve for write-audits on sensitive types |

Rules are evaluated **top-down; first match wins** — the entire craft is
rule order: mute the noise first, then escalate the sensitive paths, then a
Metadata catch-all.

## 2 · A production-shaped policy

`/etc/kubernetes/audit/policy.yaml`:

```yaml
apiVersion: audit.k8s.io/v1
kind: Policy
# don't record the payload of the noise floor
omitStages: ["RequestReceived"]
rules:
  # 1 · mute the firehose: node heartbeats, lease renewals, self-reads
  - level: None
    users: ["system:kube-proxy", "system:apiserver"]
  - level: None
    userGroups: ["system:nodes"]
    verbs: ["get", "update"]
    resources:
      - group: "coordination.k8s.io"
        resources: ["leases"]
  - level: None
    nonResourceURLs: ["/healthz*", "/readyz*", "/livez*", "/metrics"]

  # 2 · the crown jewels: every Secret/ConfigMap touch, with payloads on writes
  - level: Metadata
    verbs: ["get", "list", "watch"]
    resources:
      - group: ""
        resources: ["secrets", "configmaps"]
  - level: RequestResponse
    verbs: ["create", "update", "patch", "delete"]
    resources:
      - group: ""
        resources: ["secrets", "configmaps"]

  # 3 · interactive access: exec/attach/portforward — record what was asked
  - level: Request
    resources:
      - group: ""
        resources: ["pods/exec", "pods/attach", "pods/portforward"]

  # 4 · RBAC changes are always a story
  - level: RequestResponse
    resources:
      - group: "rbac.authorization.k8s.io"
        resources: ["*"]

  # 5 · everything else: who did what, no payloads
  - level: Metadata
```

Why Metadata (not more) for Secret *reads*: a `get secret` response body IS
the secret — logging it at `RequestResponse` would copy every credential
into the audit log, creating the leak you're auditing for.

## 3 · Wire the apiserver

`/etc/kubernetes/manifests/kube-apiserver.yaml` (static pod; kubelet
restarts it on save):

```yaml
- --audit-policy-file=/etc/kubernetes/audit/policy.yaml
- --audit-log-path=/var/log/kubernetes/audit/audit.log
- --audit-log-maxsize=100      # MB per file
- --audit-log-maxbackup=5
- --audit-log-maxage=30        # days
```

…plus hostPath mounts for both the policy dir (readOnly) and the log dir.
The log is JSON-lines; each event carries `auditID`, `user.username`,
`verb`, `objectRef`, `responseStatus`, and RBAC's verdict in
`annotations."authorization.k8s.io/reason"` — the *why* behind every allow.

## 4 · Interrogate it

```sh
# who read a specific secret?
grep '"resource":"secrets"' audit.log | grep '"name":"db-creds"' \
  | grep '"verb":"get"' | head
# who exec'd into anything today, into what?
grep '"subresource":"exec"' audit.log \
  | grep -o '"username":"[^"]*"\|"namespace":"[^"]*"\|"name":"[^"]*"' | paste - - -
# who deleted the namespace? (the on-call classic)
grep '"verb":"delete"' audit.log | grep '"resource":"namespaces"'
```

Ship the file off-node (fluent-bit DaemonSet or `--audit-webhook-config-file`
to a collector): logs that share fate with the node they audit disappear
exactly when needed — Datadog's lesson (M8.6), applied to forensics.

## Takeaway

- Default cluster = zero request history. Audit policy is opt-in; opting in
  is one file + one flag.
- First-match-wins means **order is the policy**: None-rules first,
  escalations second, Metadata catch-all last.
- Secret reads at `Metadata`, Secret writes at `RequestResponse`, exec at
  `Request` — the trio that answers real incident questions.
- CKS asks for exactly this: write a policy for given requirements, wire the
  flags, find an event in the log. The grep patterns above are the exam and
  the incident, same skill.

## Your turn

`init` created **`kubelings/db-creds`** and two empty directories. Right now
the cluster cannot tell you who has read that Secret. Fix that:

1. Write `/etc/kubernetes/audit/policy.yaml` (§2 is a working starting point).
2. Wire the `--audit-policy-file` and `--audit-log-path` flags into the
   kube-apiserver static pod, with hostPath mounts for **both** directories —
   the policy dir read-only, the log dir writable (§3).
3. Once the apiserver is back, read the Secret so there's an event to find:
   `kubectl -n kubelings get secret db-creds -o yaml`

The check looks for a recorded `get` of `db-creds` in the audit log — **and**
verifies the event does *not* contain the Secret's value. Getting the level
right is the point, not just getting logging on.

<details>
<summary>Hint</summary>

Two ways this fails.

**Nothing in the log at all.** Usually one of: the log dir is mounted
`readOnly: true` (the apiserver can't write), only one of the two flags is
set, or an earlier `level: None` rule is swallowing the event. Rules are
first-match-wins, so a broad `None` near the top mutes everything below it.

**The event is there but contains `s3cure-AUDIT-4242`.** You logged Secret
reads at `Request` or `RequestResponse`. For a `get`, the response body *is*
the Secret — so the audit log now holds a copy of every credential anyone
reads. Reads go at `Metadata`; `RequestResponse` is for writes.

If the apiserver doesn't come back at all:

```sh
crictl ps -a | grep apiserver
crictl logs $(crictl ps -a --name kube-apiserver -q | head -1) 2>&1 | tail -20
```

A malformed policy file is a startup error, not a warning — the apiserver
refuses to start rather than audit nothing silently.

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


## 1 · The policy

```sh
mkdir -p /etc/kubernetes/audit /var/log/kubernetes/audit
cat >/etc/kubernetes/audit/policy.yaml <<'YAML'
apiVersion: audit.k8s.io/v1
kind: Policy
omitStages: ["RequestReceived"]
rules:
  # mute the firehose first
  - level: None
    users: ["system:kube-proxy", "system:apiserver"]
  - level: None
    userGroups: ["system:nodes"]
    verbs: ["get", "update"]
    resources:
      - group: "coordination.k8s.io"
        resources: ["leases"]
  - level: None
    nonResourceURLs: ["/healthz*", "/readyz*", "/livez*", "/metrics"]

  # Secret/ConfigMap reads: WHO and WHAT, never the payload
  - level: Metadata
    verbs: ["get", "list", "watch"]
    resources:
      - group: ""
        resources: ["secrets", "configmaps"]
  # writes: full bodies are worth the cost
  - level: RequestResponse
    verbs: ["create", "update", "patch", "delete"]
    resources:
      - group: ""
        resources: ["secrets", "configmaps"]

  - level: Request
    resources:
      - group: ""
        resources: ["pods/exec", "pods/attach", "pods/portforward"]
  - level: RequestResponse
    resources:
      - group: "rbac.authorization.k8s.io"
        resources: ["*"]
  - level: Metadata
YAML
```

## 2 · Wire the apiserver

In `/etc/kubernetes/manifests/kube-apiserver.yaml`, add to the command:

```yaml
- --audit-policy-file=/etc/kubernetes/audit/policy.yaml
- --audit-log-path=/var/log/kubernetes/audit/audit.log
- --audit-log-maxsize=100
- --audit-log-maxbackup=5
- --audit-log-maxage=30
```

and the mounts — note the asymmetry, policy read-only, logs writable:

```yaml
      volumeMounts:
        - {name: audit-policy, mountPath: /etc/kubernetes/audit, readOnly: true}
        - {name: audit-logs,   mountPath: /var/log/kubernetes/audit}
  volumes:
    - name: audit-policy
      hostPath: {path: /etc/kubernetes/audit, type: DirectoryOrCreate}
    - name: audit-logs
      hostPath: {path: /var/log/kubernetes/audit, type: DirectoryOrCreate}
```

Wait for it:

```sh
until kubectl get --raw=/readyz >/dev/null 2>&1; do sleep 2; done
```

## 3 · Generate and find the event

```sh
kubectl -n kubelings get secret db-creds -o yaml

grep '"resource":"secrets"' /var/log/kubernetes/audit/audit.log \
  | grep '"name":"db-creds"' | grep '"verb":"get"' | tail -1 | python3 -m json.tool
```

You'll see `user.username`, `verb`, `objectRef`, `responseStatus`, and RBAC's
verdict in `annotations."authorization.k8s.io/reason"` — the *why* behind the
allow. What you won't see is `s3cure-AUDIT-4242`, because the rule that
matched was `Metadata`.

## Root cause, restated

A default cluster has no request history at all. That's not a gap in the
logs — it's the absence of logs. The first question of every incident
response ("what did that token touch?") is unanswerable, permanently and
retroactively, unless you turned this on *before* the incident.

And the level matters as much as the switch. An audit log that records
Secret reads at `RequestResponse` is a single file containing every
credential in the cluster, usually with weaker access control than etcd and
often shipped off-node to a log aggregator many people can read. Auditing
done carelessly is a bigger exposure than not auditing at all.

</details>
