---
title: Incident Library
description: Real, cited Kubernetes production postmortems mapped to the Kubelings curriculum, plus clearly-labeled synthetic failure patterns.
---

Real production failures are the best teachers Kubernetes has. This library maps
**cited, public postmortems** to the Kubelings module that teaches the underlying
concept — and, where a failure is reproducible on `kind`, to a runnable lesson.

Two kinds of entries, honestly labeled:

- **`[REAL]`** — a verifiable public postmortem or conference talk from a named
  company, with the source linked. Nothing paraphrased beyond what the source says.
- **`[PATTERN]`** — a synthetic composite of a failure mode seen across many
  clusters. No company attached, because attaching one would be fiction.

All source links below were reachability-checked on 2026-07-08. (Medium links may
show a bot-check to crawlers but open fine in a browser.)

## `[REAL]` incidents

| Company | Incident | Teaches | Module | Source |
|---|---|---|---|---|
| Zalando | Total DNS outage — ndots:5 amplification OOMKills all CoreDNS | DNS internals, resource limits, shared-fate monitoring | M4 · runnable lesson `incident-dns-ndots` · [case study](/incidents/zalando-dns-outage/) | [postmortem](https://github.com/zalando-incubator/kubernetes-on-aws/blob/dev/docs/postmortems/jan-2019-dns-outage.md) |
| Reddit | Pi-Day outage — Calico/CNI drift during upgrade | upgrades, CNI, config drift | M9 capstone · reading `incident-reddit-piday` · [case study](/incidents/reddit-piday/) | [postmortem](https://www.reddit.com/r/RedditEng/comments/11xx5o0/you_broke_reddit_the_piday_outage/) |
| OpenAI | Dec 2024 — telemetry rollout overwhelmed control plane, broke DNS discovery | control-plane overload, DNS discovery, blast radius | M9 capstone · reading `incident-openai-cascade` · [case study](/incidents/openai-telemetry-cascade/) | [status postmortem](https://status.openai.com/incidents/ctrsv3lwd797) |
| Datadog | 2023-03-08 — systemd-networkd wiped Cilium routes, 24h multi-region outage | CNI, node networking, host-OS interaction | M8 · reading `incident-datadog-cilium` · [case study](/incidents/datadog-cilium-routes/) | [postmortem](https://www.datadoghq.com/blog/2023-03-08-multiregion-infrastructure-connectivity-issue/) |
| Monzo | Current-account payments fail — etcd + Linkerd cascade | cascading failure, endpoints, service mesh | M9 capstone · reading `incident-monzo-cascade` · [case study](/incidents/monzo-cascade/) | [public incident thread](https://community.monzo.com/t/resolved-current-account-payments-may-fail-major-outage-27-10-2017/26296/95) |
| Monzo | Anatomy of a Production Kubernetes Outage (talk) | same cascade, deep-dive | M9 capstone | [KubeCon talk](https://www.youtube.com/watch?v=OUYTNywPk-s) |
| Spotify | Accidentally deleted ALL kube clusters — no user impact | infra-as-code blast radius, DR | M8 | [KubeCon talk](https://www.youtube.com/watch?v=ix0Tw8uinWs) |
| Grafana Labs | Production outage caused by Pod Priorities | priority & preemption | M5 · runnable `incident-priority-preemption` · [case study](/incidents/grafana-priority-preemption/) | [blog](https://grafana.com/blog/2019/07/24/how-a-production-outage-was-caused-using-kubernetes-pod-priorities/) |
| Jetstack | Simple admission webhook → cluster outage | admission webhooks, failurePolicy | M6 · runnable `incident-webhook-outage` · [case study](/incidents/jetstack-webhook-outage/) | [blog](https://blog.jetstack.io/blog/gke-webhook-outage) |
| JW Player | Cryptominer on internal clusters via exposed dashboard | attack surface, RBAC, exposure | M6 · runnable `incident-cryptominer` · [case study](/incidents/jwplayer-cryptominer/) | [blog](https://medium.com/jw-player-engineering/how-a-cryptocurrency-miner-made-its-way-onto-our-internal-kubernetes-clusters-9b09c4704205) |
| Moonlight | All pods scheduled to the same failing host | anti-affinity, spread | M5 · runnable `incident-same-node` · [case study](/incidents/moonlight-same-node/) | [postmortem](https://updates.moonlightwork.com/outage-post-mortem-87370) |
| Target | Cascading failure of distributed systems (Kafka/Consul/Docker) | cascade thinking, resource stampedes | M9 capstone | [blog](https://medium.com/@daniel.p.woods/on-infrastructure-at-scale-a-cascading-failure-of-distributed-systems-7cff2a3cd2df) |
| Datadog | 10 ways to shoot yourself in the foot (ndots, IPVS, DaemonSets) | DNS, kube-proxy modes, images | M4/M7 | [KubeCon talk](https://www.youtube.com/watch?v=QKI-JRs2RIE) |
| Zalando | Kubelet `--kube-api-qps` starves CD platform builds | kubelet flags, API throughput | M7 | [postmortem](https://github.com/zalando-incubator/kubernetes-on-aws/blob/dev/docs/postmortems/jun-2019-kubelet-qps.md) |
| Zalando | A million ways to crash your cluster (Ingress/etcd/CronJob/CPU) | broad ops survey | M8 | [slides](https://www.slideshare.net/try_except_/running-kubernetes-in-production-a-million-ways-to-crash-your-cluster-devopscon-munich-2018) |
| Zalando | Let's talk about failures (NotReady nodes, ELB, CoreDNS) | node lifecycle, ingress | M8 | [slides](https://www.slideshare.net/try_except_/lets-talk-about-failures-with-kubernetes-hamburg-meetup) |
| Zalando | How to crash your cluster (talk) | IAM, kubelet QPS, OOMKill | M7/M8 | [talk](https://www.youtube.com/watch?v=LpFApeaGv7A) |
| Zalando | Kubernetes failure stories (talk) | Ingress, etcd, CPU throttling | M8 | [talk](https://www.youtube.com/watch?v=6sDTB4eV4F8) |
| Airbnb | 10 weird ways to blow up your Kubernetes | sidecars, DaemonSets, JVM+HPA | M2/M5 | [talk](https://www.youtube.com/watch?v=FrQ8Lwm9_j8) |
| Airbnb | 10 MORE weird ways (webhooks, CPU limits, kube2iam) | admission, throttling | M2/M6 | [talk](https://www.youtube.com/watch?v=4CT0cI62YHk) |
| Airbnb | Did Kubernetes make my p95s worse? | CPU throttling, DNS latency | M2/M4 | [talk](https://www.youtube.com/watch?v=QXApVwRBeys) |
| Skyscanner | A couple of characters brought down our site | GitOps templating, namespace deletion | M8 | [blog](https://medium.com/@SkyscannerEng/how-a-couple-of-characters-brought-down-our-site-356ccaf1fbc3) |
| Skyscanner | One templating line, clusters in pain | Service VIPs, templating | M4 | [blog](https://medium.com/@SkyscannerEng/misunderstanding-the-behaviour-of-one-templating-line-and-the-pain-it-caused-our-k8s-clusters-a420f30a99f1) |
| Adevinta | Kubernetes made my latency 10× higher | DNS, KIAM, per-request costs | M4 | [blog](https://srvaroa.github.io/kubernetes/migration/latency/dns/java/aws/microservices/2019/10/22/kubernetes-added-a-0-to-my-latency.html) |
| Preply | DNS public postmortem #1 — conntrack races | conntrack, CoreDNS scaling | M4 | [blog](https://medium.com/preply-engineering/dns-postmortem-e169efd45afd) |
| loveholidays | Conntrack table exhaustion networking failures | conntrack limits | M4 · reading `incident-conntrack` · [case study](/incidents/conntrack-exhaustion/) | [blog](https://deploy.live/blog/kubernetes-networking-problems-due-to-the-conntrack/) |
| loveholidays | When GKE ran out of IP addresses | IP planning, autoscaling ceilings | M4 | [blog](https://deploy.live/blog/when-gke-ran-out-of-ip-addresses/) |
| loveholidays | The shipwreck of GKE cluster upgrade | upgrades, pod availability | M8 | [blog](https://deploy.live/blog/the-shipwreck-of-gke-cluster-upgrade/) |
| Omio | CPU limits and aggressive throttling | CFS throttling, limits | M2 · runnable `incident-cpu-throttling` | [blog](https://medium.com/omio-engineering/cpu-limits-and-aggressive-throttling-in-kubernetes-c5b20bd8a718) |
| Buffer | Faster services by removing CPU limits | CPU limits trade-offs | M2 · runnable `incident-cpu-throttling` | [blog](https://erickhun.com/posts/kubernetes-faster-services-no-cpu-limits/) |
| MindTickle | The case of the missing packet (EKS CNI) | AWS CNI, packet tracing | M4 | [blog](https://yashmehrotra.com/post/2020-03-16-case-of-missing-packet/) |
| MindTickle | Intermittent delays — conntrack DNAT races, musl vs libc | conntrack, resolver behavior | M4 | [blog](https://medium.com/techmindtickle/intermittent-delays-in-kubernetes-e9de8239e2fa) |
| Ravelin | Kubernetes' dirty endpoint secret and Ingress | graceful shutdown, endpoints lag | M4 · runnable `incident-graceful-shutdown` · [case study](/incidents/ravelin-graceful-shutdown/) | [blog](https://philpearl.github.io/post/k8s_ingress/) |
| Blue Matador | Node OOM postmortem — no resource limits | SystemOOM, eviction | M8 · runnable `incident-node-oom` · [case study](/incidents/bluematador-node-oom/) | [blog](https://www.bluematador.com/blog/post-mortem-kubernetes-node-oom) |
| Civis Analytics | How we broke (and fixed) our K8s cluster | batch jobs vs API server | M8 | [blog](https://medium.com/civis-analytics/https-medium-com-civis-analytics-breaking-kubernetes-how-we-broke-and-fixed-our-k8s-cluster-adfa6fbade61) |
| Xing | Moving to Kubernetes: the bad and the ugly | Ingress, conntrack, PLEG | M8 | [talk](https://www.youtube.com/watch?v=MoIdU0J0f0E) |
| Nordstrom | 101 ways to break and recover a cluster | NotReady, eviction, etcd splits | M7/M8 | [talk](https://www.youtube.com/watch?v=xZO9nx6GBu0) |
| Algolia | Killing the dashboard during Black Friday (Jobs overload) | Jobs, overload control | M9 capstone · reading `incident-black-friday` · [case study](/incidents/algolia-black-friday/) | [talk](https://www.youtube.com/watch?v=Fjyg7cxRZQs) |
| FREE NOW | New K8s workers unable to join cluster | node bootstrap, spot instances | M8 | [postmortem PDF](https://github.com/freenowtech/postmortems/blob/master/2019-09-19%20-%20New%20K8s%20workers%20unable%20to%20join%20cluster.pdf) |
| Tinder | Move to K8s at scale — 250k rps DNS, conntrack races, ARP cache outage (Jan 2019) | DNS at scale, conntrack, node kernel limits | M4 · [case study](/incidents/tinder-scale-migration/) | [blog](https://medium.com/tinder/tinders-move-to-kubernetes-cda2a6372f44) |
| CircleCI | 2023-03-14 — kubelet/kube-proxy version skew corrupted iptables mid-upgrade; 7h+ outage, two follow-on incidents | version skew, kube-proxy sync, upgrade staging | M8 · [case study](/incidents/circleci-version-skew/) | [incident report](https://discuss.circleci.com/t/incident-report-2023-03-14-delays-starting-jobs/47555) |
| Heroku | 2025-06-10 — unattended system update flushed network routes fleet-wide; ~24h platform outage (Datadog's 2023 failure mode, recurring) | immutable infra, auto-updates, host-OS × CNI | M8 · see `incident-datadog-cilium` | [official summary](https://www.heroku.com/blog/summary-of-june-10-outage/) · [status](https://status.heroku.com/incidents/2822) |
| Neon | 2025-05 — IP exhaustion in K8s subnets (AWS CNI) + control-plane overload; repeat incident from a remediation regression | IP planning, CNI limits, change-induced repeats | M4 | [postmortem](https://neon.com/blog/postmortem-delayed-start-compute-operations) |
| Chick-fil-A | Bare-metal k3s in every restaurant — 2,800+ edge clusters, field-failure lessons | edge k8s, fleet ops, node observability gaps | M8 | [blog](https://medium.com/chick-fil-atech/bare-metal-k8s-clustering-at-chick-fil-a-scale-929a0e6d29e5) · [evolution](https://medium.com/chick-fil-atech/how-our-edge-kubernetes-platform-has-evolved-12609006bc92) |

*More cited incidents are added as each source is verified — the upstream index is
[kubernetes-failure-stories](https://codeberg.org/hjacobs/kubernetes-failure-stories) (formerly k8s.af).*

## `[PATTERN]` scenarios (synthetic, labeled)

| Pattern | Teaches | Module | Case study |
|---|---|---|---|
| PVC stuck `Terminating` | finalizers, storage protection | M3/M7 | [read](/incidents/pattern-pvc-terminating/) |
| Noisy-neighbor CPU throttling | requests/limits, QoS | M2/M5 | coming soon |
| Readiness probe flaps under load | probes, endpoint churn | M2 | coming soon |
| Zombie CronJobs pile up | Job history limits, TTL | M2 | coming soon |
| Ghost endpoints after scale-down | graceful shutdown | M4 | coming soon |
| Secret rotated, pods never noticed | mounts vs env, reload | M3 | coming soon |

## How incidents become lessons

Reproducible incidents ship as **runnable lessons inside the module that teaches
the concept** (e.g. Zalando's ndots amplifier → Module 4, lesson
`incident-dns-ndots`). Multi-concept cascades (Monzo, Target) become **Module 9
capstone labs**. Every runnable incident lesson links back to its cited source —
you fix the same class of failure the original team fixed.
