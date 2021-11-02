# Softcom Telemetry - Go

---

A telemetry library that supports the following out of the box:

- logging
- tracing
- health checks (wip)

### Dependencies

- [Elastic APM lib](https://go.elastic.co/apm)
- [Logrus]( https://github.com/sirupsen/logrus )
- [gRPC](https://grpc.io)
- [Softcom JetStream Client](https://github.com/softcomoss/jetstreamclient)


### Usage

```go
package main

import "github.com/softcomoss/gotelemtry"

func main()  {

	tlm := gotelemetry.NewServerTelemetry("example-service", "production")

	tlm.Info("fishcobite")
	
	
	tlm.U
}
```