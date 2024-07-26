package main

import (
	"flag"
	"net/http"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	targetURL     string
	checkInterval time.Duration
	maxFailures   int
	logger        *zap.Logger
)

func init() {
	flag.StringVar(&targetURL, "url", "http://www.google.com/generate_204", "Target URL")
	flag.DurationVar(&checkInterval, "interval", 5*time.Second, "Check interval(5s, 5m, 1h)")
	flag.IntVar(&maxFailures, "max-failures", 3, "Allow maximum failures before disable ICMP response")

	flag.Parse()

	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var err error
	logger, err = config.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
}

func main() {
	failureCount := 0

	logger.Info("Start monitoring",
		zap.String("targetURL", targetURL),
		zap.Duration("checkInterval", checkInterval),
		zap.Int("maxFailures", maxFailures))

	for {
		if !checkConnection() {
			failureCount++
			logger.Info("Connection fail", zap.Int("failure_count", failureCount))

			if failureCount >= maxFailures {
				disableICMP()
				logger.Warn("Reach max failures, disable ICMP response")
				break
			}
		} else {
			failureCount = 0
			logger.Info("Connection success")
		}

		time.Sleep(checkInterval)
	}
}

func checkConnection() bool {
	resp, err := http.Get(targetURL)
	if err != nil {
		logger.Error("Get HTTP fail", zap.Error(err))
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode <= 299
}

func disableICMP() {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.icmp_echo_ignore_all=1")
	err := cmd.Run()
	if err != nil {
		logger.Error("Close IPv4 ICMP response fail", zap.Error(err))
		os.Exit(1)
	}
	cmd = exec.Command("sysctl", "-w", "net.ipv6.icmp_echo_ignore_all=1")
	err = cmd.Run()
	if err != nil {
		logger.Error("Close IPv6 ICMP response fail", zap.Error(err))
		os.Exit(1)
	}
}
