---
kind: unit
title: "Audit policy: who touched that Secret?"
name: audit-policy-unit
---


> **Runbook reading.** Audit policy is a kube-apiserver flag plus a file on
> the control-plane node — host-level wiring, outside the kubectl sandbox
> (M6.8's confinement note). Commands are exact; rehearse on a kind node or
> iximiuz VM. Deep dive behind survey §2 of `control-plane-hardening`.

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
