---
kind: lesson
title: 'Incident file — how we failed to integrate Istio (Exponea)'
description: |
  Read-through of Exponea's cited write-up: a service-mesh rollout abandoned
  after the sidecar broke Jobs, StatefulSets, gRPC load balancing and graceful
  shutdown one after another. No tasks; this is the platform-engineering
  judgement call — what a mesh actually costs, which of your workloads it
  changes, and how to decide whether to adopt one at all.
name: incident-istio-integration
slug: incident-istio-integration
source: https://medium.com/@jakubkulich/sailing-with-the-istio-through-the-shallow-water-8ae81668381e
createdAt: "2026-07-27"
playground:
  name: k8s-omni
---
