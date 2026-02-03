# Miniku

I wanted to learn a bit more about distributed systems, and after reading more of the [500 lines or less book](https://aosabook.org/en/500L/introduction.html), I wanted to have a crack at a mini version of K8S.

## Core Acceptance

- API server (expose desired state)
- etcd (k/v store for cluster state, nothing fancy planned)
- scheduler
- controller/manager (reconciliation loops for "uptime guarantees" and all that jazz.)
- kubelet (afaik this is just an agent/worker that lives on a node)

## Disclaimer

This project is purely for my own education. That means **no** LLM's, which also means it's not going to be production-ready code. The Pod spec is purposefully simple (name, img, state) because I do not need anything else for my goals.
