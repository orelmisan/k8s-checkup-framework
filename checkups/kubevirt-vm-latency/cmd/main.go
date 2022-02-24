package main

import (
	"log"

	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/config"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/reporter"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/tests"
)

func main() {
	const errMsgPrefix = "Kubevirt network latency check failed"

	envVars, err := config.LoadEnvVars()
	if err != nil {
		log.Fatalf("%s: %v", errMsgPrefix, err)
	}

	options, err := tests.CreateLatencyCheckOptions(envVars)
	if err != nil {
		log.Fatalf("%s: %v", errMsgPrefix, err)
	}

	result := tests.StartNetworkLatencyCheck(options)

	reporter := reporter.NewConfigMapReporter(options.ResultConfigMapName, options.ResultConfigMapNamespace)
	if err := reporter.WriteToConfigMap(result); err != nil {
		log.Fatal(err)
	}

	if result.Error != nil {
		log.Fatalf("%s: %v", errMsgPrefix, result.Error)
	}
}
