# Miniku (Mini Kubernetes)

I wanted to learn a bit more about distributed systems, and after reading more of the [500 lines or less book](https://aosabook.org/en/500L/introduction.html), I wanted to have a crack at a mini version of K8S.

```sh
> curl -X POST 127.0.0.1:8080/replicasets -d '{"name":"nginx","desiredCount":3,"selector":{"app":"nginx"},"template":{"image":"nginx:latest"}}'
{"name":"nginx","desiredCount":3,"currentCount":0,"selector":{"app":"nginx"},"template":{"name":"","image":"nginx:latest"}}

> curl 127.0.0.1:8080/replicasets
[{"name":"nginx","desiredCount":3,"currentCount":3,"selector":{"app":"nginx"},"template":{"name":"","image":"nginx:latest"}}]

> curl 127.0.0.1:8080/pods
[{"spec":{"name":"nginx-ca7f8c36","image":"nginx:latest","labels":{"app":"nginx"}},"status":"Pending","retry_count":0,"next_retry_at":"0001-01-01T00:00:00Z"},{"spec":{"name":"nginx-7cfc52b8","image":"nginx:latest","labels":{"app":"nginx"}},"status":"Pending","retry_count":0,"next_retry_at":"0001-01-01T00:00:00Z"},{"spec":{"name":"nginx-161d920c","image":"nginx:latest","labels":{"app":"nginx"}},"status":"Pending","retry_count":0,"next_retry_at":"0001-01-01T00:00:00Z"}]

> docker ps
CONTAINER ID   IMAGE          COMMAND                  CREATED         STATUS         PORTS     NAMES
590cb04088f5   nginx:latest   "/docker-entrypoint.…"   2 seconds ago   Up 2 seconds   80/tcp    nginx-407fd916
d08ef4919f86   nginx:latest   "/docker-entrypoint.…"   3 seconds ago   Up 2 seconds   80/tcp    nginx-6f999539
fe3685e63fa1   nginx:latest   "/docker-entrypoint.…"   3 seconds ago   Up 3 seconds   80/tcp    nginx-9271c514
```

## Core Acceptance

- [x] API server (expose desired state)
- [x] kubelet (afaik this is just an agent/worker that lives on a node)
- [x] controller/manager (reconciliation loops for "uptime guarantees" and all that jazz.)
- [ ] scheduler
- [ ] etcd (k/v store for cluster state, nothing fancy planned)

## Core Spec

### API

TODO

### Kubelet Action

| Pod Status | Container State | Action                                       |
| ---------- | --------------- | -------------------------------------------- |
| Pending    | doesn't exist   | Create container, update pod to Running      |
| Pending    | running         | Just update pod to Running (recovered?)      |
| Running    | running         | Nothing - we're converged                    |
| Running    | exited          | Update pod to Failed                         |
| Running    | doesn't exist   | Weird state - maybe re-create or mark Failed |

## Disclaimer

This project is purely for my own education. That means **no** LLM's, which also means it's not going to be production-ready code. ~~The Pod spec is purposefully simple (name, img, state) because I do not need anything else for my goals.~~ Turns out that was a lie
