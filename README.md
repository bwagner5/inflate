# inflate

`inflate` creates Kubernetes Deployments for scheduler and node-autoscaler demos.

## Karpenter demo

Point your current kubeconfig context at the demo cluster and run:

```console
inflate demo
```

The foreground loop creates independently randomized single-pod Deployments, grows to 5–20 demo pods, holds, then removes them and repeats. CPU requests range from 250m to 10 cores and memory requests range from 256Mi to 5Gi. Required hostname anti-affinity places each demo pod on a different node, giving Karpenter clear scale-up and scale-down events without creating an unbounded cluster. Press Ctrl-C to remove the demo workloads and stop.

The default `medium` preset is suitable for a varied autoscaling demo. Two other presets bundle useful sizing and timing defaults:

| Preset | Pod/node target | Replicas per Deployment | CPU per pod | Memory per pod | Change interval | Hold |
| --- | --- | --- | --- | --- | --- | --- |
| `low` | 5–10 | 1–3 | 250m–1500m | 256Mi–1536Mi | 15s | 1m |
| `medium` (default) | 5–20 | 1 | 250m–10 cores | 256Mi–5Gi | 15s | 2m |
| `high` | 20–50 | 1 | 250m–10 cores | 256Mi–5Gi | 5s | 1m |

Select a preset with `--preset`. Any explicitly specified sizing or timing flag overrides the selected preset, so this uses the low preset with a longer hold:

```sh
inflate demo --preset low --hold 3m
```

Timing, resource ranges, and node bounds are configurable:

```console
inflate demo --min-nodes 5 --max-nodes 8 --interval 20s --hold 2m
```

The cluster's baseline nodes and Karpenter limits can make the total node count differ from the demo pod target.

## Create one Deployment

```console
inflate create example --replicas 3 --cpu 750m --memory 512Mi
inflate get
inflate delete example
```

Use `inflate create --help` or `inflate demo --help` for all scheduling and kubeconfig options. Release archives for macOS and Linux on amd64 and arm64 are available from [GitHub Releases](https://github.com/bwagner5/inflate/releases).
