---
kind: lesson
title: 'kube-proxy: there is no proxy'
description: |
  Reading — what actually happens to a packet addressed to a ClusterIP. The
  IP that exists on no interface, the iptables chains that rewrite it, the
  IPVS and nftables alternatives, and where conntrack plugs in. Closes the
  loop on every Service mystery in this module.
name: kube-proxy-dataplane
slug: kube-proxy-dataplane
createdAt: "2026-07-08"
playground:
  name: k8s-omni
---
