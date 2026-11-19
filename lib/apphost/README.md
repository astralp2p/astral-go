# apphost

Client library for the [apphost](../../mod/apphost/README.md) module.

## Basic usage

```go
host := apphost.Connect(apphost.DefaultEndpoint)

fmt.Println("connected to host %v (%v)", host.HostAlias(), host.HostID())

err := host.AuthToken(token)

fmt.Println("authenticated as %v", host.GuestID())

conn, err := host.RouteQuery(query.New(nil, nil, "user.info", nil)
```
## Retrying a failed query

`IsRetryable(err)` is true for a rejection with code 7 (unavailable) or 6
(timeout) and for a timeout, and false for every other error. A caller
retries those with a bounded backoff, discovers the target again on
`route_not_found`, and stops on any other error.
