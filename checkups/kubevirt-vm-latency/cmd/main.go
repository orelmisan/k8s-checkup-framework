package main

import (
	"log"

	"kubevirt.io/client-go/kubecli"

	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/config"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/reporter"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/tests"
)

func main() {
	const errMsgPrefix = "Kubevirt network latency check failed"

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		log.Fatalf("%s: failed to obtain KubeVirt client %v", errMsgPrefix, err)
	}

	envVars, err := config.LoadEnvVars()
	if err != nil {
		log.Fatalf("%s: %v", errMsgPrefix, err)
	}

	options, err := tests.CreateLatencyCheckOptions(envVars)
	if err != nil {
		log.Fatalf("%s: %v", errMsgPrefix, err)
	}
	result := tests.StartNetworkLatencyCheck(virtClient, options)
	resultData := tests.ResultToStringsMap(result)

	if err := reporter.WriteToStdout(resultData); err != nil {
		log.Fatalf("%s: %v", errMsgPrefix, err)
	}

	if err := reporter.WriteToConfigMap(virtClient, options.ResultConfigMapNamespace, options.ResultConfigMapName, resultData); err != nil {
		log.Fatalf("%s: %v", errMsgPrefix, err)
	}

	if result.Error != nil {
		log.Fatalf("%s: %v", errMsgPrefix, result.Error)
	}
}
