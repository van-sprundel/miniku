# Miniku (Mini Kubernetes)

I wanted to learn a bit more about distributed systems, and after reading more of the [500 lines or less book](https://aosabook.org/en/500L/introduction.html), I wanted to have a crack at a mini version of K8S.

## Starting a ReplicaSet (which in turn starts the desired pods)

```sh
ramones~/g/miniku go run ./cmd/miniku/
```

Then in another terminal

```sh
> curl -X POST 127.0.0.1:8080/replicasets -d '{"name":"nginx","desiredCount":4,"selector":{"app":"nginx"},"template":{"image":"nginx:latest"}}'
{"name":"nginx","desiredCount":4,"currentCount":0,"selector":{"app":"nginx"},"template":{"name":"","image":"nginx:latest"}}

> curl 127.0.0.1:8080/replicasets
[{"name":"nginx","desiredCount":4,"currentCount":4,"selector":{"app":"nginx"},"template":{"name":"","image":"nginx:latest"}}]

> curl 127.0.0.1:8080/pods
[{"spec":{"name":"nginx-d3950207","image":"nginx","node_name":"node-2","labels":{"app":"nginx"}},"status":"Running","containerId":"d803aed838112089a96869d408699c12db28a05e1fd2e17b68c6ebe26366cf52","retry_count":0,"next_retry_at":"0001-01-01T00:00:00Z"},{"spec":{"name":"nginx-3852c037","image":"nginx","node_name":"node-2","labels":{"app":"nginx"}},"status":"Running","containerId":"de1a4328ce058af55d41c11b60aa8a0ca5e572222968f2feec5ae829658d2447","retry_count":0,"next_retry_at":"0001-01-01T00:00:00Z"},{"spec":{"name":"nginx-50e73e92","image":"nginx","node_name":"node-1","labels":{"app":"nginx"}},"status":"Running","containerId":"3ab21beda01dcab02583000d539dd3822b5b711ba5b98cce33dbaac406754641","retry_count":0,"next_retry_at":"0001-01-01T00:00:00Z"},{"spec":{"name":"nginx-76e8a1ac","image":"nginx","node_name":"node-1","labels":{"app":"nginx"}},"status":"Running","containerId":"0b0803bb1ab43c46ac27e5c21598f8fc967bcd85dbff5add58ca1f60d08102e4","retry_count":0,"next_retry_at":"0001-01-01T00:00:00Z"}]

> docker ps
CONTAINER ID   IMAGE     COMMAND                  CREATED          STATUS          PORTS     NAMES
0b0803bb1ab4   nginx     "/docker-entrypoint.…"   23 seconds ago   Up 23 seconds   80/tcp    nginx-76e8a1ac
d803aed83811   nginx     "/docker-entrypoint.…"   23 seconds ago   Up 23 seconds   80/tcp    nginx-d3950207
3ab21beda01d   nginx     "/docker-entrypoint.…"   24 seconds ago   Up 23 seconds   80/tcp    nginx-50e73e92
de1a4328ce05   nginx     "/docker-entrypoint.…"   24 seconds ago   Up 23 seconds   80/tcp    nginx-3852c037
```

Now our binary logs the following:

```sh
2026/02/04 15:27:54 scheduler: assigning pod nginx-d3950207 to node node-2
2026/02/04 15:27:54 scheduler: assigning pod nginx-3852c037 to node node-2
2026/02/04 15:27:54 scheduler: assigning pod nginx-50e73e92 to node node-1
2026/02/04 15:27:54 scheduler: assigning pod nginx-76e8a1ac to node node-1
```

Very nice!

## Core Acceptance

- [x] API server (expose desired state)
- [x] kubelet (afaik this is just an agent/worker that lives on a node)
- [x] controller/manager (reconciliation loops for "uptime guarantees" and all that jazz.)
- [x] rediscover (on restart, find matching label containers and re-assign (docker))
- [x] scheduler (this will require multiple kubelets)
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
