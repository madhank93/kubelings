---
kind: unit
title: "Incident file — how we failed to integrate Istio (Exponea)"
name: incident-istio-integration-unit
---


> **Incident file (guided reading).** No lab: a mesh install plus a fleet to put
> behind it is a platform decision, not a namespace. What transfers is the shape
> of the cost — every failure below is a consequence of *one* architectural fact,
> and recognising that fact is what makes the adopt/don't-adopt call answerable.
>
> Sources:
> [Exponea — Sailing with Istio through the shallow water](https://medium.com/@jakubkulich/sailing-with-the-istio-through-the-shallow-water-8ae81668381e) ·
> [Airbnb — 10 more weird ways to blow up your Kubernetes](https://www.youtube.com/watch?v=4CT0cI62YHk)

## What happened

Exponea (GKE, microservices) adopted Istio for the usual, good reasons: traffic
insight, mTLS between services, retries/timeouts moved out of application code.
They started with GKE's managed Istio, found it configurable only for mTLS mode,
went self-managed, and then met the sidecar — repeatedly:

- **Port names became load-bearing.** Istio routes by protocol inferred from the
  port *name* (`http`, `grpc`, `tcp-…`). A port named wrongly — or not at all —
  silently changed how traffic was handled, and services failed at startup.
- **Jobs stopped completing.** The app container exits; **the sidecar proxy does
  not** — it's a server, it runs forever. The pod never reaches Completed, so the
  Job never finishes and the pipeline waiting on it never moves. (M2
  `incident-job-restartpolicy` is the same organ from the other side: what
  "finished" means for a Job.)
- **StatefulSets misbehaved.** Service discovery didn't track pod IP changes on
  recreation, so connections were routed to addresses that no longer existed.
- **gRPC client-side load balancing fought the mesh.** Headless Services exist so
  clients can balance across endpoints themselves; the sidecar wants to do that
  job, and the two answers to "who balances?" don't compose.
- **Graceful shutdown regressed.** The sidecar exited immediately on SIGTERM,
  cutting off app containers that needed their full grace period to finish work
  in flight. (M4 `incident-graceful-shutdown` — now with a second process whose
  lifecycle you also have to get right.)
- **Upgrades had no good story.** Sidecar version rollout across a fleet, plus
  unsupported multi-tenancy, made production adoption a project rather than a
  step.

They concluded the value of "Istio in a few services" didn't cover the cost, and
postponed the integration.

## The one fact underneath all of it

**A mesh puts a second process inside every pod and a second network hop in every
call.** Every symptom above follows:

| Consequence | Symptom class |
|---|---|
| Two containers, one pod lifecycle | Jobs never complete; shutdown ordering; startup races (app talks before proxy is ready) |
| Traffic is now *parsed*, not just forwarded | protocol/port-name sensitivity, header rules, mTLS handshake failures |
| The mesh owns discovery and balancing | conflicts with headless Services, client-side LB, StatefulSet identity |
| The mesh is a control plane you now operate | upgrades, CRD sprawl, its own outages (a broken sidecar injector = M6 `incident-webhook-outage`) |
| Every request pays proxy CPU/latency | tail-latency budget, per-pod memory floor across the whole fleet |

None of this makes meshes wrong — it makes them *large*. The 2019 write-up
predates a lot of maturation (sidecar `holdApplicationUntilProxyStarts`, native
sidecar containers in Kubernetes 1.29+ which fix the Job-never-completes problem
outright, and **ambient mode**, which removes the per-pod sidecar entirely).
The judgement it teaches is the durable part.

## Concept checks

- Your batch Jobs hang at 1/2 Ready after a mesh rollout. Two fixes exist, one
  modern and one historical — name both. (Modern: Kubernetes **sidecar
  containers** — `initContainers` with `restartPolicy: Always` — so the proxy is
  terminated when the main container exits. Historical: call the proxy's
  `/quitquitquit` endpoint at the end of the job, or exclude Jobs from injection.)
- Why does a service mesh make the *port name* a correctness concern when plain
  Kubernetes does not? (Plain Services forward bytes; a mesh terminates and
  parses the protocol to do routing, retries and telemetry — so it must be told
  which protocol it's holding, and the name is the declaration.)
- What would you buy a mesh for if you already have NetworkPolicy, an ingress
  controller, and OpenTelemetry? (Honest answers: mTLS identity between services
  without app changes, L7 retries/traffic splitting for progressive delivery, and
  uniform golden signals. If none of those are unsolved problems for you, the
  mesh is cost without benefit.)

## What the industry took from it

- **Adopt a mesh workload-class by workload-class, not cluster-wide.** Stateless
  HTTP services first; Jobs, StatefulSets and anything doing client-side
  balancing last, deliberately, with their own tests.
- **Exclude by default, include on purpose** — namespace/pod injection labels are
  the blast-radius control, and they're free until you use them.
- **Budget the tax before the pilot:** per-pod CPU/memory of the proxy × fleet
  size, plus added p99. If the number is uncomfortable at 50 services, it will be
  unacceptable at 500.
- **Prefer the smallest thing that solves your actual problem.** Gateway API
  (M4 `gateway-api`) for north-south routing, NetworkPolicy (M4/M6) for
  segmentation, OTel (M8 `otel-collector-pipeline`) for traces — reach for the
  mesh when you need service identity and L7 policy *between* services and you're
  prepared to run a control plane to get them.

*No check — study, then advance.*

<!-- d2:sidecar -->
```text
   ┌─────────────────┐  
   │sidecar injected │  
   │                 │  
   └─────────────────┘  
           │            
 second process in pod  
           │            
           ▼            
  ┌───────────────────┐ 
  │proxy runs forever │ 
  │                   │ 
  └───────────────────┘ 
           │            
app exits, pod does not 
           │            
           ▼            
  ┌────────────────────┐
  │Job never completes │
  │                    │
  └────────────────────┘
```
<!-- /d2:sidecar -->
