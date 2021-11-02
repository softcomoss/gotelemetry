package main

import "github.com/softcomoss/gotelemetry"

func main() {
	tlm := gotelemetry.NewServerTelemetry("example-service", "production")

	tlm.Info("fishcobite")

	server := tlm.UseInterceptedGRPCServer()

	server.S
}
