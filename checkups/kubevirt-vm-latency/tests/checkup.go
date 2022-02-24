package tests

import (
	"fmt"
	"log"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	expect "github.com/google/goexpect"
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

type Options struct {
	ResultConfigMapNamespace string        `json:"-"`
	ResultConfigMapName      string        `json:"-"`
	WorkingNamespace         string        `json:"workingNamespace"`
	NetworkNamespace         string        `json:"networkNamespace"`
	NetworkName              string        `json:"networkName"`
	SourceNode               string        `json:"sourceNode"`
	TargetNode               string        `json:"targetNode"`
	Duration                 time.Duration `json:"-"`
	SourceMacAddr            string        `json:"-"`
	TargetMacAddr            string        `json:"-"`
	SourceCIDR               string        `json:"-"`
	TargetCIDR               string        `json:"-"`
}

type Result struct {
	Options  Options         `json:",inline"`
	Duration string          `json:"duration,omitempty"`
	Error    error           `json:"failureReason"`
	Latency  ping.PingResult `json:"latency"`
}

func StartNetworkLatencyCheck(options Options) Result {
	result := Result{}

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		result.Error = fmt.Errorf("failed to obtain KubeVirt client: %v\n", err)
		return result
	}

	if err := runNetworkLatencyPreflightChecks(virtClient, options); err != nil {
		result.Error = err
		return result
	}

	options.SourceMacAddr = sourceMAC
	options.SourceCIDR = sourceCIDR
	options.TargetMacAddr = targetMAC
	options.TargetCIDR = targetCIDR
	sourceVMI, targetVMI, err := startNetworkLatencyCheckVMIs(virtClient, options)
	if err != nil {
		result.Error = err
		return result
	}

	result, err = runNetworkLatencyCheck(virtClient, options.NetworkName, sourceVMI, targetVMI, options.Duration)
	if err != nil {
		result.Error = err
		return result
	}
	result.Duration = options.Duration.String()
	result.Options = options

	if err := vmis.DeleteAndWaitVmisDispose(virtClient, sourceVMI, targetVMI); err != nil {
		result.Error = err
		return result
	}

	return result
}

func runNetworkLatencyPreflightChecks(virtClient kubecli.KubevirtClient, options Options) error {
	const errMsgPrefix = "not all preflight checks passed"

	log.Println("Starting preflights checks..")

	if err := preflight.VerifyConfigMapExists(virtClient, options.ResultConfigMapNamespace, options.ResultConfigMapName); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := preflight.VerifyKubevirtAvailable(virtClient); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := preflight.VerifyNetworkAttachmentDefinitionExists(virtClient, options.NetworkNamespace, options.NetworkName); err != nil {
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
	result.Latency = ping.ParsePingLatencyResult(responses)

	return result, nil
}

type createLatencyCheckVmiFn func(namespace, networkNamespace, networkName, mac, cidr, node string) *v1.VirtualMachineInstance

func startNetworkLatencyCheckVMIs(virtClient kubecli.KubevirtClient, options Options) (*v1.VirtualMachineInstance, *v1.VirtualMachineInstance, error) {
	const errMsgPrefix = "failed to setup netwrok latency check"

	var fn createLatencyCheckVmiFn

	networkType, err := nads.GetNetworkType(virtClient, options.NetworkNamespace, options.NetworkName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	switch networkType {
	case networkTypeBridge, networkTypeCNVBridge:
		fn = vmis.NewLatencyCheckVmiWithBridgeIface
	case networkTypeSRIOV:
		fn = vmis.NewLatencyCheckVmiWithSriovIface
	}
	sourceVMI := fn(options.WorkingNamespace, options.NetworkNamespace, options.NetworkName, options.SourceMacAddr, options.SourceCIDR, options.SourceNode)
	targetVMI := fn(options.WorkingNamespace, options.NetworkNamespace, options.NetworkName, options.TargetMacAddr, options.TargetCIDR, options.TargetNode)

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
