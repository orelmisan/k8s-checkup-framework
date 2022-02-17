package ping

import (
	"log"
	"regexp"
	"strconv"
	"strings"

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

type PingLatencyResult struct {
	Min     float64 `json:"min,omitempty"`
	Max     float64 `json:"max,omitempty"`
	Average float64 `json:"average,omitempty"`
	Jitter  float64 `json:"jitter,omitempty"`
}

func ParsePingLatencyResult(pingResult []expect.BatchRes) PingLatencyResult {
	var result PingLatencyResult
	latencyPattern := regexp.MustCompile(`(round-trip|rtt)\s+\S+\s*=\s*([0-9.]+)/([0-9.]+)/([0-9.]+)/([0-9.]+)\s*ms`)

	for _, response := range pingResult {
		matches := latencyPattern.FindAllStringSubmatch(response.Output, -1)
		for _, item := range matches {
			min, err := strconv.ParseFloat(strings.TrimSpace(item[2]), 64)
			if err != nil {
				log.Printf("failed to parse min latency from result: %v", err)
			}
			result.Min = min

			avg, _ := strconv.ParseFloat(strings.TrimSpace(item[3]), 64)
			if err != nil {
				log.Printf("failed to parse average jitter from result: %v", err)
			}
			result.Average = avg

			max, err := strconv.ParseFloat(strings.TrimSpace(item[4]), 64)
			if err != nil {
				log.Printf("failed to parse max latency from result: %v", err)
			}
			result.Max = max

			jitter, _ := strconv.ParseFloat(strings.TrimSpace(item[5]), 64)
			if err != nil {
				log.Printf("failed to parse jitter from result: %v", err)
			}
			result.Jitter = jitter
		}
	}

	return result
}
