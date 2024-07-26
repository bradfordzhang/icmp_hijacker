package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	TargetURL     string
	CheckInterval time.Duration
	MaxFailures   int
}

var (
	config Config
	logger *zap.Logger
)

func initConfig() {
	flag.StringVar(&config.TargetURL, "url", "http://www.google.com/generate_204", "目标URL")
	flag.DurationVar(&config.CheckInterval, "interval", 5*time.Second, "检查间隔(5s, 5m, 1h)")
	flag.IntVar(&config.MaxFailures, "max-failures", 3, "禁用ICMP响应前允许的最大失败次数")

	flag.Parse()

	zapConfig := zap.NewProductionConfig()
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var err error
	logger, err = zapConfig.Build()
	if err != nil {
		panic(err)
	}
}

func main() {
	initConfig()
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("开始监控",
		zap.String("targetURL", config.TargetURL),
		zap.Duration("checkInterval", config.CheckInterval),
		zap.Int("maxFailures", config.MaxFailures))

	monitorChan := make(chan bool)
	go monitorConnection(ctx, monitorChan)

	// 处理优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		logger.Info("收到关闭信号,正在停止...")
		cancel()
	case <-monitorChan:
		logger.Info("监控完成")
	}
}

func monitorConnection(ctx context.Context, done chan<- bool) {
	failureCount := 0
	icmpDisabled := false

	ticker := time.NewTicker(config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			done <- true
			return
		case <-ticker.C:
			if !checkConnection() {
				failureCount++
				logger.Info("连接失败", zap.Int("失败次数", failureCount))

				if failureCount >= config.MaxFailures && !icmpDisabled {
					if err := disableICMP(); err == nil {
						icmpDisabled = true
						logger.Warn("达到最大失败次数,已禁用ICMP响应")
					}
				}
			} else {
				failureCount = 0
				logger.Info("连接成功")
				if icmpDisabled {
					if err := enableICMP(); err == nil {
						icmpDisabled = false
						logger.Warn("连接恢复,已启用ICMP响应")
					}
				}
			}
		}
	}
}

func checkConnection() bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(config.TargetURL)
	if err != nil {
		logger.Error("HTTP GET失败", zap.Error(err))
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func disableICMP() error {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.icmp_echo_ignore_all=1")
	if err := cmd.Run(); err != nil {
		logger.Error("禁用IPv4 ICMP响应失败", zap.Error(err))
		return err
	}
	return nil
}

func enableICMP() error {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.icmp_echo_ignore_all=0")
	if err := cmd.Run(); err != nil {
		logger.Error("启用IPv4 ICMP响应失败", zap.Error(err))
		return err
	}
	return nil
}
