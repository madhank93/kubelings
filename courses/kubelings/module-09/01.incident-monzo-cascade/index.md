---
kind: lesson
title: 'Incident file — the cascade (Monzo''s bank-stopping outage)'
description: |
  Guided study of Monzo's cited outage: a routine change touched etcd, which
  confused a service mesh, which returned empty endpoints, which crashed clients
  on a null pointer — and a bank's payments stopped. Trace how five familiar
  mechanisms compose into one national-scale failure.
name: incident-monzo-cascade
slug: incident-monzo-cascade
source: https://community.monzo.com/t/resolved-current-account-payments-may-fail-major-outage-27-10-2017/26296/95
createdAt: "2026-07-07"
playground:
  name: k8s-omni
---
