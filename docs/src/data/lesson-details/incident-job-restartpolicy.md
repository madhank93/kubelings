## The real incident

**Universe** (ticketing platform), 21 July 2017. Search and reporting went down —
and so did the cluster UI, because the cluster itself went unhealthy. The trigger
was a **fix**: a reindex Job, shipped to repair broken search indexing.

Source: [Universe — Search and Reporting Outage](http://status.universe.com/incidents/115n3vxqwzcf)

Their postmortem names the misunderstanding precisely: the Job carried
`restartPolicy: Never`, and the author believed that meant *this runs once and
stops*. It doesn't. **`restartPolicy` is the pod's contract with its kubelet, not
the Job's contract with the cluster.** `Never` only says: when this container
exits, don't restart it *in place*. The Job controller then notices the pod
failed and does its own thing — it creates **another pod**. The reindexer was
resource-hungry; every retry landed somewhere new, ate what it could, and died.
Nodes fell over, the failure cascaded, and the cluster stayed sick until a human
deleted the Job.

Three fields would have made this a boring afternoon. None of them were set.

## This cluster, right now

The same reindex Job is running in `kubelings` — same intent, same defaults:

```sh
kubectl -n kubelings get job search-reindex
kubectl -n kubelings get pods -l job-name=search-reindex   # the graveyard grows
kubectl -n kubelings logs -l job-name=search-reindex --tail=3
```

The container dies on a corrupt document. It will keep dying — that part is a
genuine bug in the reindexer, and **not your job today**. Your job is that this
bug currently has no blast-radius limit at all:

```sh
kubectl -n kubelings get job search-reindex -o jsonpath='{.spec.backoffLimit}{"\n"}'
# 2147483647  ← "forever", spelled in int32
```

<figure class="lesson-diagram">
<img src="/diagrams/incident-job-restartpolicy-retry.svg" alt="incident-job-restartpolicy retry diagram" loading="lazy">
</figure>

## Your task

Make the failure bounded. Keep the reindexer failing — bound what its failure can
do:

1. `backoffLimit` **≤ 4** — the Job gives up instead of grinding.
2. `activeDeadlineSeconds` **≤ 300** — wall-clock has a ceiling too, for the
   failure mode where each attempt hangs rather than exits.
3. `ttlSecondsAfterFinished` set — finished Jobs and their pods collect
   themselves instead of accumulating.
4. `resources.limits` (cpu **and** memory) on the container — a retry storm must
   never be able to take a node with it.
5. The Job must end **Failed** on its own, and no more than `backoffLimit + 1`
   pods may remain.

Job specs are largely **immutable** — you'll delete and recreate, not patch.
