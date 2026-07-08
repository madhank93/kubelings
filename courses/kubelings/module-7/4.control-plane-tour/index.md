---
kind: lesson
title: 'The control-plane tour: request flow, kubelet, leases'
description: |
  A guided walk through the machinery — what happens between "kubectl apply" and
  a running container, how the kubelet turns a PodSpec into containers via the
  CRI, and how leader election keeps one controller-manager in charge. Reading +
  live pokes on this cluster; no single check to pass.
name: control-plane-tour
slug: control-plane-tour
createdAt: "2026-07-07"
playground:
  name: k8s-omni
---
