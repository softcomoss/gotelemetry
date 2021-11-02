package main

import gotelemetry "github.com/softcomoss/gotelemtry"

func main() {
	tlm := gotelemetry.NewServerTelemetry("example-service", "production")

	tlm.Info("fishcobite")

	server := tlm.UseInterceptedGRPCServer()

	server.S
}
