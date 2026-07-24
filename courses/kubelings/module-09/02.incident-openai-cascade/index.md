---
kind: lesson
title: 'Incident file — locked out of the control plane (OpenAI, 2024)'
description: |
  Guided study of OpenAI's cited December 2024 outage: a fleet-wide telemetry
  rollout overwhelmed every Kubernetes API server at once, DNS caching hid the
  damage until the rollout was everywhere, and the overloaded control plane
  locked engineers out of the fix. Control plane vs data plane, as a war story.
name: incident-openai-cascade
slug: incident-openai-cascade
source: https://status.openai.com/incidents/ctrsv3lwd797
createdAt: "2026-07-07"
playground:
  name: k8s-omni
---
