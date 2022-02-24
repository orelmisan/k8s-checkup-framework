package ping

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	expect "github.com/google/goexpect"

	"k8s.io/utils/net"
)

const (
	ping  = "ping"
	ping6 = "ping -6"
)

func ComposePingCommand(ipAddr string, args ...string) string {
	pingString := ping
	if net.IsIPv6String(ipAddr) {
		pingString = ping6
	}

	if len(args) == 0 {
		args = []string{"-c 5 -w 10"}
	}
	args = append([]string{pingString, ipAddr}, args...)

	return strings.Join(args, " ")
}

type PingResult struct {
	Min     time.Duration `json:"min,omitempty"`
	Max     time.Duration `json:"max,omitempty"`
	Average time.Duration `json:"average,omitempty"`
	Jitter  time.Duration `json:"jitter,omitempty"`
}

func ParsePingLatencyResult(pingResult []expect.BatchRes) PingResult {
	var result PingResult
	latencyPattern := regexp.MustCompile(`(round-trip|rtt)\s+\S+\s*=\s*([0-9.]+)/([0-9.]+)/([0-9.]+)/([0-9.]+)\s*ms`)

	for _, response := range pingResult {
		matches := latencyPattern.FindAllStringSubmatch(response.Output, -1)
		for _, item := range matches {
			min, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[2])))
			if err != nil {
				log.Printf("failed to parse min latency from result: %v", err)
			}
			result.Min = min

			average, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[3])))
			if err != nil {
				log.Printf("failed to parse average jitter from result: %v", err)
			}
			result.Average = average

			max, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[4])))
			if err != nil {
				log.Printf("failed to parse max latency from result: %v", err)
			}
			result.Max = max

			jitter, _ := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[5])))
			if err != nil {
				log.Printf("failed to parse jitter from result: %v", err)
			}
			result.Jitter = jitter
		}
	}

	return result
}
