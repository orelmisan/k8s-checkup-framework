package tests

import (
	"fmt"
	"log"
	"os"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	expect "github.com/google/goexpect"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/config"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/ping"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/tests/console"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/tests/nads"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/tests/preflight"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/tests/vmis"
)

const (
	networkTypeSRIOV     = "sriov"
	networkTypeBridge    = "bridge"
	networkTypeCNVBridge = "cnv-bridge"
)

const (
	sourceMAC = "02:00:00:f9:32:1f"
	targetMAC = "02:00:00:7b:55:76"

	sourceCIDR = "192.168.0.100/24"
	targetCIDR = "192.168.0.200/24"
)

type options struct {
	ResultConfigMapNamespace string
	ResultConfigMapName      string
	workingNamespace         string
	networkNamespace         string
	networkName              string
	sourceNode               string
	targetNode               string
	sampleDuration           time.Duration
	desiredMaxLatency        time.Duration
	sourceMacAddr            string
	targetMacAddr            string
	sourceCIDR               string
	targetCIDR               string
}

type Result struct {
	Error   error
	options options
	latency ping.PingResult
}

const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func CreateLatencyCheckOptions(params map[string]string) (options, error) {
	const errMsgPrefix = "failed to create latency check options"

	sampleDuration, err := time.ParseDuration(params[config.SampleDurationEnvVarName])
	if err != nil {
		return options{}, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	maxLatency, err := time.ParseDuration(params[config.DesiredMaxLatencyEnvVarName])
	if err != nil {
		return options{}, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	workingNamespace, err := os.ReadFile(namespaceFile)
	if err != nil {
		return options{}, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	options := options{
		workingNamespace:         string(workingNamespace),
		ResultConfigMapNamespace: params[config.ResultsConfigMapNamespaceEnvVarName],
		ResultConfigMapName:      params[config.ResultsConfigMapNameEnvVarName],
		networkNamespace:         params[config.NetworkNamespaceEnvVarName],
		networkName:              params[config.NetworkNameEnvVarName],
		sourceNode:               params[config.SourceNodeNameEnvVarName],
		targetNode:               params[config.TargetNodeNameEnvVarName],
		sampleDuration:           sampleDuration,
		desiredMaxLatency:        maxLatency,
	}

	return options, nil
}

func StartNetworkLatencyCheck(virtClient kubecli.KubevirtClient, options options) Result {
	result := Result{}

	if err := runNetworkLatencyPreflightChecks(virtClient, options); err != nil {
		result.Error = err
		return result
	}

	options.sourceMacAddr = sourceMAC
	options.sourceCIDR = sourceCIDR
	options.targetMacAddr = targetMAC
	options.targetCIDR = targetCIDR
	sourceVMI, targetVMI, err := startNetworkLatencyCheckVMIs(virtClient, options)
	if err != nil {
		result.Error = err
		return result
	}

	result, err = runNetworkLatencyCheck(virtClient, options.networkName, sourceVMI, targetVMI, options.sampleDuration)
	if err != nil {
		result.Error = err
		return result
	}

	if result.latency.Max > options.desiredMaxLatency {
		result.Error = fmt.Errorf("max latency is greater than expected: expected: (%v) result: (%v)", options.desiredMaxLatency, result.latency.Max)
	}

	result.options = options

	if err := vmis.DeleteAndWaitVmisDispose(virtClient, sourceVMI, targetVMI); err != nil {
		result.Error = err
		return result
	}

	return result
}

func runNetworkLatencyPreflightChecks(virtClient kubecli.KubevirtClient, options options) error {
	const errMsgPrefix = "not all preflight checks passed"

	log.Println("Starting preflights checks..")

	if err := preflight.VerifyConfigMapExists(virtClient, options.ResultConfigMapNamespace, options.ResultConfigMapName); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := preflight.VerifyKubevirtAvailable(virtClient); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := preflight.VerifyNetworkAttachmentDefinitionExists(virtClient, options.networkNamespace, options.networkName); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	return nil
}

func runNetworkLatencyCheck(virtClient kubecli.KubevirtClient, netwrorkName string, sourceVMI, targetVMI *v1.VirtualMachineInstance, duration time.Duration) (Result, error) {
	const errMsgPrefix = "network latency check failed"

	var result Result

	targetVmiIP, err := vmis.GetVmiNetwrokIPAddress(virtClient, targetVMI.Namespace, targetVMI.Name, netwrorkName)
	if err != nil {
		return result, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	responses, err := pingFromVMConsole(duration, sourceVMI, targetVmiIP)
	if err != nil {
		return result, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	result.latency = ping.ParsePingLatencyResult(responses)

	return result, nil
}

type createLatencyCheckVmiFn func(namespace, networkNamespace, networkName, mac, cidr, node string) *v1.VirtualMachineInstance

func startNetworkLatencyCheckVMIs(virtClient kubecli.KubevirtClient, options options) (*v1.VirtualMachineInstance, *v1.VirtualMachineInstance, error) {
	const errMsgPrefix = "failed to setup netwrok latency check"

	var fn createLatencyCheckVmiFn

	networkType, err := nads.GetNetworkType(virtClient, options.networkNamespace, options.networkName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	switch networkType {
	case networkTypeBridge, networkTypeCNVBridge:
		fn = vmis.NewLatencyCheckVmiWithBridgeIface
	case networkTypeSRIOV:
		fn = vmis.NewLatencyCheckVmiWithSriovIface
	}
	sourceVMI := fn(options.workingNamespace, options.networkNamespace, options.networkName, options.sourceMacAddr, options.sourceCIDR, options.sourceNode)
	targetVMI := fn(options.workingNamespace, options.networkNamespace, options.networkName, options.targetMacAddr, options.targetCIDR, options.targetNode)

	if err := vmis.StartAndWaitVmisReady(virtClient, sourceVMI, targetVMI); err != nil {
		return nil, nil, err
	}

	ipacmd := "ip a"
	_, err = console.RunCommand(sourceVMI, ipacmd, time.Second*15)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	_, err = console.RunCommand(targetVMI, ipacmd, time.Second*15)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	return sourceVMI, targetVMI, nil
}

func pingFromVMConsole(timeout time.Duration, vmi *v1.VirtualMachineInstance, ipAddr string, args ...string) ([]expect.BatchRes, error) {
	resp, err := console.RunCommand(vmi, ping.ComposePingCommand(ipAddr, fmt.Sprintf("-w %d", int(timeout.Seconds()))), timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to ping from VMI %s/%s to %s, error: %v",
			vmi.Namespace, vmi.Name, ipAddr, err)
	}
	return resp, nil
}

const (
	succeededKey      = "status.succeeded"
	failureReasonKey  = "status.failureReason"
	minLatencyKey     = "status.result.minLatency"
	maxLatencyKey     = "status.result.maxLatency"
	averageLatencyKey = "status.result.averageLatency"
	jitterKey         = "status.result.jitter"
)

func ResultToStringsMap(result Result) map[string]string {
	resultMap := map[string]string{}
	resultMap[composeSpecEnvKey(config.NetworkNamespaceEnvVarName)] = result.options.networkNamespace
	resultMap[composeSpecEnvKey(config.NetworkNameEnvVarName)] = result.options.networkName
	resultMap[composeSpecEnvKey(config.SourceNodeNameEnvVarName)] = result.options.sourceNode
	resultMap[composeSpecEnvKey(config.TargetNodeNameEnvVarName)] = result.options.targetNode
	resultMap[composeSpecEnvKey(config.SampleDurationEnvVarName)] = result.options.sampleDuration.String()
	resultMap[composeSpecEnvKey(config.DesiredMaxLatencyEnvVarName)] = result.options.desiredMaxLatency.String()

	var failureReason string
	var succeeded bool
	if result.Error != nil {
		failureReason = result.Error.Error()
		succeeded = false
	} else {
		failureReason = ""
		succeeded = true
	}

	resultMap[succeededKey] = fmt.Sprintf("%v", succeeded)
	resultMap[failureReasonKey] = failureReason
	resultMap[minLatencyKey] = result.latency.Min.String()
	resultMap[maxLatencyKey] = result.latency.Max.String()
	resultMap[averageLatencyKey] = result.latency.Average.String()
	resultMap[jitterKey] = result.latency.Jitter.String()

	return resultMap
}

func composeSpecEnvKey(envVarName string) string {
	return fmt.Sprintf("spec.env.%s", envVarName)
}
