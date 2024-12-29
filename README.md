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

## cmd/groundwork

This code base demonstrates the usage of middlewares for access logging and panic recovery as well as demonstrating how teh graceful shutdown works.  To gracefully handle the shutdown of an HTTP server you either need to run the shutdown functionality in a goroutine or the server listening in a goroutine.  The first article referenced below demonstrates the latter scenario, the code provided demonstrates the former setup.

We introduce using a `Makefile` to build the project and demonstrate how to inject build information into the generated binary. The build information is displayed as part of the startup log messages.

We introduce using a logger to generate structured logs using `log/slog` instead of writing text string to standard output and standard error.

We introduce our first external dependency here the `felixge/httpsnoop` package used in the request logging middleware.

### Building using the Makefile

The `Makefile` contains useful commands for simplifying the building and testing the programs in this project and contains a built in `help` functionality (the default if no command specified)

To build one of the command binaries like `groundwork`, run the below

```sh
make groundwork
```

To build all binaries from this project use
```sh
make build
```

### Starting up the server

To run:

```sh
./bin/groundwork
```

### Testing the panic handlers

In a terminal session curl the panic endpoint which does not have an appropriate recovery middleware on it
```sh
$ curl -v http://localhost:4000/example/panic
* Host localhost:4000 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:4000...
* Connected to localhost (::1) port 4000
> GET /example/panic HTTP/1.1
> Host: localhost:4000
> User-Agent: curl/8.5.0
> Accept: */*
> 
* Empty reply from server
* Closing connection
curl: (52) Empty reply from server
```
Which results on the server side an error message that starts similar to the below and contains an entire traceback in the message.
```
time=2024-12-29T21:01:13.826+13:00 level=ERROR msg="http: panic serving [::1]:41138: example\n```
```

To test the panic handler with the recovery middleware on it

```sh
$ curl -v http://localhost:4000/example/recoveredpanic
* Host localhost:4000 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:4000...
* Connected to localhost (::1) port 4000
> GET /example/recoveredpanic HTTP/1.1
> Host: localhost:4000
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/1.1 500 Internal Server Error
< Date: Sun, 29 Dec 2024 08:01:25 GMT
< Content-Length: 0
< 
* Connection #0 to host localhost left intact
```
Which results on the server side with
```
time=2024-12-29T21:30:16.684+13:00 level=ERROR msg=panic reason=example
time=2024-12-29T21:30:16.684+13:00 level=INFO msg=served host=localhost:4000 username="" received=2024-12-29T21:30:16.684+13:00 method=GET uri=/example/recoveredpanic proto=HTTP/1.1 status=500 size=0 duration=43.531µs
```

### Testing the graceful shutdown process

The `slow` endpoint is used to test the graceful shutdown and is affected by the `httpReadTimeout`, `httpWriteTimeout`, and the `gracePeriod` timeout.  You'll need multiple terminal sessions to test this.

When you execute the query to read from the `slow` endpoint the server will display a message instructing to press Ctrl-C on the server session.
```sh
$ curl http://localhost:4000/example/slow
```
Server shutdown message
```
time=2024-12-29T21:06:13.847+13:00 level=INFO msg="press Ctrl-C now on this server to trigger the shutdown"
```
Once you've pressed Ctrl-C to shut down the process you'll get the following:
```
time=2024-12-29T21:06:13.847+13:00 level=INFO msg="press Ctrl-C now on this server to trigger the shutdown"
```

If you then attempt to run another API query of any kind you'll get something like the below
```sh
$ curl http://localhost:4000/hello/world
curl: (7) Failed to connect to localhost port 4000 after 0 ms: Couldn't connect to server
```

In the meantime the first curl command to the `slow` endpoint is still executing, and when that completes you'll get the below log entries
```
time=2024-12-29T21:12:57.912+13:00 level=INFO msg="handler completed before httpWriteTimeout completed"
time=2024-12-29T21:12:57.913+13:00 level=INFO msg=served host=localhost:4000 username="" received=2024-12-29T21:12:42.912+13:00 method=GET uri=/example/slow proto=HTTP/1.1 status=200 size=21 duration=15.000601827s
time=2024-12-29T21:12:58.206+13:00 level=INFO msg="Shutdown completed."
```

Using the `never` endpoint we can see what the behaviour is if the request does not complete during the graceful shutdown period and we get an empty response.

```sh
$ curl -v http://localhost:4000/example/never
* Host localhost:4000 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:4000...
* Connected to localhost (::1) port 4000
> GET /example/never HTTP/1.1
> Host: localhost:4000
> User-Agent: curl/8.5.0
> Accept: */*
> 
* Empty reply from server
* Closing connection
curl: (52) Empty reply from server
```


References:

* https://dev.to/mokiat/proper-http-shutdown-in-go-3fji
* https://pkg.go.dev/github.com/felixge/httpsnoop
