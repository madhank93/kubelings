---
kind: lesson
title: 'HA control plane: three of everything'
description: |
  Reading — what it takes for the control plane to survive losing a node:
  stacked vs external etcd, the --control-plane-endpoint decision, joining
  more control-plane nodes, who load-balances the apiserver, and how the
  singleton controllers pick a leader.
name: ha-control-plane
slug: ha-control-plane
createdAt: "2026-07-13"
playground:
  name: k8s-omni
---
