# MiniKu (Mini Kubernetes)

I wanted to learn a bit more about distributed systems, and after reading more of the [500 lines or less book](https://aosabook.org/en/500L/introduction.html), I wanted to have a crack at a mini version of K8S.

## Coverage

<!-- coverage-start -->

| Package          | Coverage  |
| ---------------- | --------- |
| `pkg/api`        | 88.0%     |
| `pkg/client`     | 79.5%     |
| `pkg/controller` | 61.2%     |
| `pkg/kubelet`    | 60.8%     |
| `pkg/scheduler`  | 60.5%     |
| `pkg/store`      | 87.7%     |
| **Total**        | **74.4%** |

<!-- coverage-end -->

## Benchmarks

<!-- bench-start -->

| Benchmark                  |   ns/op |   B/op | allocs/op |
| -------------------------- | ------: | -----: | --------: |
| MatchesSelector            |   77.65 |      0 |         0 |
| GetMatchingPods/pods=100   |  705280 | 194626 |      1367 |
| Reconcile                  | 1231754 |  95980 |       800 |
| PickNode/nodes=10          |  144928 |  13441 |       123 |
| ScheduleOne                |  411754 |  29407 |       265 |
| GetAvailableNodes/nodes=10 |  143687 |  12417 |       122 |
| MemStorePut                |   449.5 |    171 |         2 |
| MemStoreGet                |   108.4 |     13 |         1 |
| MemStoreList/size=100      |    1335 |   2688 |         1 |
| MemStoreDelete             |   233.7 |     27 |         3 |

<!-- bench-end -->

## Architecture

```
  cmd/apiserver      cmd/scheduler      cmd/controller      cmd/kubelet
  (stores live       (HTTP client)      (HTTP client)       (HTTP client +
   here only)                                                namespace runtime)
       |                  |                   |                   |
       +------ HTTP ------+------- HTTP ------+------- HTTP ------+
```

All components communicate through the API server over HTTP. `cmd/miniku` runs everything in a single process for convenience.

## Container Runtime

Miniku uses a **Linux namespace-based container runtime**. This way containers are isolated using kernel namespaces (PID, MNT, UTS) with `pivot_root` into an Alpine rootfs.

Requires `sudo` to run (namespace creation needs `CAP_SYS_ADMIN`).

**Note:** `PodSpec.Image` currently maps to Alpine minirootfs regardless of the image name specified.

## Starting a ReplicaSet (which in turn starts the desired pods)

```sh
> sudo go run ./cmd/miniku/
```

Then in another terminal

```sh
> curl -X POST 127.0.0.1:8080/replicasets -d '{"name":"test","desiredCount":4,"selector":{"app":"test"},"template":{"image":"alpine","command":["/bin/sh","-c","while true; do sleep 1; done"]}}'
{"name":"test","desiredCount":4,"currentCount":0,"selector":{"app":"test"},"template":{"name":"","image":"alpine","command":["/bin/sh","-c","while true; do sleep 1; done"]}}

> curl 127.0.0.1:8080/replicasets
[{"name":"test","desiredCount":4,"currentCount":4,"selector":{"app":"test"},"template":{"name":"","image":"alpine","command":["/bin/sh","-c","while true; do sleep 1; done"]}}]

> curl 127.0.0.1:8080/pods
# pods with status "Running", each with a container ID
```

Now our binary logs the following:

```sh
2026/02/04 15:27:54 scheduler: assigning pod test-d3950207 to node node-2
2026/02/04 15:27:54 scheduler: assigning pod test-3852c037 to node node-2
2026/02/04 15:27:54 scheduler: assigning pod test-50e73e92 to node node-1
2026/02/04 15:27:54 scheduler: assigning pod test-76e8a1ac to node node-1
```

Very nice!

# Core Acceptance

- [x] API server (expose desired state)
- [x] kubelet (afaik this is just an agent/worker that lives on a node)
- [x] controller/manager (reconciliation loops for "uptime guarantees" and all that jazz.)
- [x] rediscover (on restart, find matching containers and re-assign)
- [x] scheduler (this will require multiple kubelets)

# Core Spec

## API

TODO

## Kubelet Action

| Pod Status | Container State | Action                                       |
| ---------- | --------------- | -------------------------------------------- |
| Pending    | doesn't exist   | Create container, update pod to Running      |
| Pending    | running         | Just update pod to Running (recovered?)      |
| Running    | running         | Nothing - we're converged                    |
| Running    | exited          | Update pod to Failed                         |
| Running    | doesn't exist   | Weird state - maybe re-create or mark Failed |

```
           (heartbeat)
  Kubelet ─────────────>  NodeStore
                              |
                              V
                          NodeController ("is heartbeat stale?")
                              |
                              V
                          NotReady if stale
```

# Disclaimer

This project is purely for my own education. That means **no** LLM's, which also means it's not going to be production-ready code. ~~The Pod spec is purposefully simple (name, img, state) because I do not need anything else for my goals.~~ Turns out that was a lie
