package main

import (
	"time"

	_ "github.com/brandond/playground-zap/pkg/nonblock"
	"go.etcd.io/etcd/client/pkg/v3/logutil"
)

func main() {
	cfg := logutil.DefaultZapLoggerConfig
	// note: Both Output AND ErrorOutput must be nonblocking.
	// If writes to Output fail, an error will be logged to ErrorOutput.
	cfg.OutputPaths = []string{"nonblock:stdout?deadline=5ms"}
	cfg.ErrorOutputPaths = []string{"nonblock:stderr?deadline=1ms"}
	lg, err := cfg.Build()
	if err != nil {
		panic("Failed to build logger config: " + err.Error())
	}
	for range time.Tick(time.Millisecond * 500) {
		lg.Info("Ping...")
		lg.Error("Pong...")
	}
}
