# Introduction

This repository contains a set of evolutions for building a web server using
Go, these are intended as educational examples.

## cmd/minimal - An initial webserver using only the standard library

This code base is more advanced that the absolute trivial configuration since it incorporates design decisions to make things far more testable, include signal handlers to trigger when the code needs to shut down, and how to deal with graceful shutdowns of in-flight requests to allow them to finish before dropping them if able. The signal handler setup used here is recommended when deploying the workload using an orchestrator like kubernetes which provides means for graceful shutdowns of the workload.

References:

* https://pkg.go.dev/net/http
* https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/
* https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/
* https://cloud.google.com/blog/products/containers-kubernetes/kubernetes-best-practices-terminating-with-grace
