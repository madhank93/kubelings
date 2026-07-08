---
kind: lesson
title: 'Incident file — the Pi-Day outage (Reddit, 2023)'
description: |
  Guided study of Reddit's cited Pi-Day outage: a routine Kubernetes upgrade on
  their oldest cluster renamed one node label — and the CNI's route reflectors,
  selecting nodes by the old name, went empty. Pod networking collapsed
  cluster-wide. A war story about deprecations, snowflake clusters, and why
  restores get tested before they're needed.
name: incident-reddit-piday
slug: incident-reddit-piday
source: https://www.reddit.com/r/RedditEng/comments/11xx5o0/you_broke_reddit_the_piday_outage/
createdAt: "2026-07-07"
playground:
  name: k8s-omni
---
