---
kind: lesson
title: 'Incident file — the identity sidecar that added a zero to latency (Adevinta)'
description: |
  Read-through of Adevinta's cited migration incident: moving to Kubernetes made
  p99 latency 10× worse, and the culprit was the machinery that gives pods cloud
  credentials — a metadata-intercepting agent in the path of every AWS call,
  amplified by DNS. No tasks; the failure needs a cloud IAM plane, so this lesson
  trains how pod identity actually works and where it hurts.
name: incident-iam-sidecar
slug: incident-iam-sidecar
source: https://srvaroa.github.io/kubernetes/migration/latency/dns/java/aws/microservices/2019/10/22/kubernetes-added-a-0-to-my-latency.html
createdAt: "2026-07-27"
playground:
  name: k8s-omni
---
