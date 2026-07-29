---
kind: lesson
title: 'Incident file — the cluster that ran out of IP addresses (loveholidays)'
description: |
  Read-through of loveholidays' cited GKE incident plus its cousins (Neon,
  Veracode): pods stuck Pending, the autoscaler refusing to add nodes, and a
  subnet that was "obviously big enough" — until you count the way Kubernetes
  counts. No tasks; the arithmetic is done at cluster-creation time and cannot be
  patched later, so this lesson trains the sizing instinct.
name: incident-ip-exhaustion
slug: incident-ip-exhaustion
source: https://deploy.live/blog/when-gke-ran-out-of-ip-addresses/
createdAt: "2026-07-27"
playground:
  name: k8s-omni
---
