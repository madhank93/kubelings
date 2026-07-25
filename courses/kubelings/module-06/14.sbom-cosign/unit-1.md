---
kind: unit
title: "Signatures and SBOMs: trust, but verify with cosign"
name: sbom-cosign-unit
---


> **Reading + light lab.** The concepts are the reading; the `cosign verify`
> is real and checked. cosign v3.1.1 is installed by init; work in
> `/tmp/kubelings-cosign/`.

## What a signature actually proves

M6.13 ended with a digest: *these exact bytes had zero CRITICALs*. One
question remains: **who built those bytes?** A digest proves integrity, not
origin — a perfectly pinned, perfectly scanned image can still be an
attacker's build if your pipeline pulled from the wrong place.

An image **signature** binds a digest to an *identity*. Classic signing used
long-lived private keys (which get leaked, rotated badly, and stuffed in
CI variables). **Sigstore keyless** replaced the key with an identity chain:

1. The signer (a CI job, a release bot) authenticates to an OIDC provider —
   "I am `keyless@distroless.iam.gserviceaccount.com`".
2. Sigstore's CA (Fulcio) issues a certificate binding that identity to a
   throwaway keypair valid for ~10 minutes.
3. The signature + certificate land in a public transparency log (Rekor).
   The key is discarded; the *log entry* is the durable proof.

Verification therefore asks: *is there a transparency-log entry where
identity X, via issuer Y, signed digest Z?* No keys to manage, and the log
makes signing events publicly auditable.

## Your task (checked)

Google's distroless images (M2.15 debugged one) are signed keyless. Verify:

```sh
cosign verify gcr.io/distroless/static:latest \
  --certificate-oidc-issuer https://accounts.google.com \
  --certificate-identity keyless@distroless.iam.gserviceaccount.com -o json
```

The JSON's `critical.image."docker-manifest-digest"` is the digest the
signature vouches for. Record exactly that value:

```sh
cosign verify gcr.io/distroless/static:latest \
  --certificate-oidc-issuer https://accounts.google.com \
  --certificate-identity keyless@distroless.iam.gserviceaccount.com \
  -o json 2>/dev/null \
  | grep -o '"docker-manifest-digest":"sha256:[a-f0-9]*"' | head -1 | grep -o 'sha256:[a-f0-9]*' \
  > /tmp/kubelings-cosign/verified-digest.txt
cat /tmp/kubelings-cosign/verified-digest.txt   # sha256:…
```

Try breaking it, too — verification *must* fail when the identity is wrong:

```sh
cosign verify gcr.io/distroless/static:latest \
  --certificate-oidc-issuer https://accounts.google.com \
  --certificate-identity somebody-else@example.com -o json
# Error: no matching signatures
```

That error is the entire point: the flags are the *policy*. Omit them and
you're asking "is this signed by anyone?" — nearly worthless.

## SBOM: the ingredient list

A **Software Bill of Materials** enumerates every package inside an image —
the answer to "are we affected?" on CVE-announcement day, *without* pulling
and scanning everything (log4shell made this a boardroom acronym). The
tooling triangle:

- **syft** generates SBOMs (`syft nginx:1.27-alpine -o spdx-json`)
- **trivy** (already installed for M6.13) both generates and *consumes*
  them: `trivy image --format spdx-json --output sbom.json <img>`, then
  later `trivy sbom sbom.json` — re-scan the manifest as new CVEs drop,
  no registry pull
- **cosign** attaches SBOMs to images as signed *attestations*:
  `cosign attest --predicate sbom.json --type spdxjson <img@digest>` — the
  SBOM travels with the image and is itself tamper-evident

SBOM answers *what's inside*; signature answers *who built it*; digest
answers *is it the same bytes*. Supply-chain security is those three
questions asked at admission time.

## Closing the loop at admission

Verification that lives only in a runbook doesn't protect anything. In a
real cluster it goes where M6.11/6.12 live:

```yaml
# Kyverno ClusterPolicy snippet — the cluster-side enforcement
rules:
  - name: require-distroless-signature
    match:
      any: [{resources: {kinds: ["Pod"]}}]
    verifyImages:
      - imageReferences: ["gcr.io/distroless/*"]
        attestors:
          - entries:
              - keyless:
                  subject: "keyless@distroless.iam.gserviceaccount.com"
                  issuer: "https://accounts.google.com"
```

Same identity policy you just typed, enforced on every pull. Gatekeeper
pairs with `ratify` for the equivalent.

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
<summary>Hint</summary>

The check reads one file: `/tmp/kubelings-cosign/verified-digest.txt`. It must
hold a **bare** digest — `sha256:…` and nothing else. A whole JSON blob, a
quoted string, or a trailing `"` all fail the format check.

Two failure modes to tell apart:

- *"no digest recorded"* — the file is missing or empty. The `> ` redirect
  never ran, or the `grep` chain matched nothing.
- *"does not match what the signature vouches for"* — you recorded a digest,
  but not the one this signature covers. Don't copy it from `docker pull` or
  the registry UI; take it from `critical.image."docker-manifest-digest"` in
  the `cosign verify -o json` output.

Check what you actually wrote before re-running the verify:

```sh
cat /tmp/kubelings-cosign/verified-digest.txt
```

Verification needs network access to the Sigstore transparency log — an
offline box fails here for reasons that have nothing to do with your commands.

</details>

<details>
<summary>Solution</summary>


```sh
cosign verify gcr.io/distroless/static:latest \
  --certificate-oidc-issuer https://accounts.google.com \
  --certificate-identity keyless@distroless.iam.gserviceaccount.com \
  -o json 2>/dev/null \
  | grep -o '"docker-manifest-digest":"sha256:[a-f0-9]*"' | head -1 | grep -o 'sha256:[a-f0-9]*' \
  > /tmp/kubelings-cosign/verified-digest.txt
```

## Takeaway

- Digest = same bytes; signature = known builder; SBOM = known contents.
  All three or you're trusting, not verifying.
- Keyless > keys for CI: nothing to leak, and Rekor's log makes every
  signing event auditable.
- The `--certificate-identity` / `--certificate-oidc-issuer` pair IS the
  security decision — pin them to exact identities, never wildcards you
  don't understand.
- Enforcement belongs in admission (Kyverno `verifyImages` / Gatekeeper +
  ratify), not in runbooks.

</details>
