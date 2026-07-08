---
kind: lesson
title: 'Incident file — the OS under the cluster (Datadog, 2023)'
description: |
  Guided study of Datadog's cited 2023 outage: a routine systemd security update
  auto-applied across the fleet restarted systemd-networkd, which silently
  deleted the network routes Cilium had installed — and tens of thousands of
  nodes across five regions dropped off the network in the same hour. The layer
  below Kubernetes can take down everything above it.
name: incident-datadog-cilium
slug: incident-datadog-cilium
source: https://www.datadoghq.com/blog/2023-03-08-multiregion-infrastructure-connectivity-issue/
createdAt: "2026-07-07"
playground:
  name: k8s-omni
---
