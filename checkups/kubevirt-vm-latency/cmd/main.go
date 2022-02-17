package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/reporter"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/tests"
)

const (
	WorkingNamespaceEnvVarName          = "POD_NAMESPACE"
	CheckupConfigMapNamespaceEnvVarName = "RESULT_CM_NAMESPACE"
	CheckupConfigMapNameEnvVarName      = "RESULT_CM_NAME"
	NetworkNamespaceEnvVarName          = "NETWORK_NAMESPACE"
	NetworkNameEnvVarName               = "NETWORK_NAME"
	LatencyCheckDurationEnvVarName      = "DURATION"
	SourceNodeNameEnvVarName            = "SOURCE_NODE"
	TargetNodeNameEnvVarName            = "TARGET_NODE"
)

var (
	WorkingNamespace          string
	CheckuoConfigMapNamespace string
	CheckupConfigMapName      string
	NetworkNamespace          string
	NetworkName               string
	LatencyCheckDuration      time.Duration
	SourceNodeName            string
	TargetNodeName            string
)

func main() {
	if err := loadEnvVars(); err != nil {
		log.Fatal(err)
	}

	options := tests.Options{
		ResultConfigMapNamespace: CheckuoConfigMapNamespace,
		ResultConfigMapName:      CheckupConfigMapName,
		WorkingNamespace:         WorkingNamespace,
		NetworkNamespace:         NetworkNamespace,
		NetworkName:              NetworkName,
		SourceNode:               SourceNodeName,
		TargetNode:               TargetNodeName,
		Duration:                 LatencyCheckDuration,
	}
	result := tests.StartNetworkLatencyCheck(options)

	reporter := reporter.NewConfigMapReporter(CheckupConfigMapName, CheckuoConfigMapNamespace)
	if err := reporter.WriteToConfigMap(result); err != nil {
		log.Fatal(err)
	}

	if result.Error != nil {
		log.Fatalf("Kubevirt network latency check failed:\n\t%v", result.Error)
	}
}

func loadEnvVars() error {
	var exists bool

	errMsgFormat := "Failed to load %s environment variable"

	WorkingNamespace, exists = os.LookupEnv(WorkingNamespaceEnvVarName)
	if !exists {
		return fmt.Errorf(errMsgFormat, WorkingNamespaceEnvVarName)
	}
	CheckuoConfigMapNamespace, exists = os.LookupEnv(CheckupConfigMapNamespaceEnvVarName)
	if !exists {
		return fmt.Errorf(errMsgFormat, WorkingNamespaceEnvVarName)
	}
	CheckupConfigMapName, exists = os.LookupEnv(CheckupConfigMapNameEnvVarName)
	if !exists {
		return fmt.Errorf(errMsgFormat, WorkingNamespaceEnvVarName)
	}
	NetworkNamespace, exists = os.LookupEnv(NetworkNamespaceEnvVarName)
	if !exists {
		return fmt.Errorf(errMsgFormat, WorkingNamespaceEnvVarName)
	}
	NetworkName, exists = os.LookupEnv(NetworkNameEnvVarName)
	if !exists {
		return fmt.Errorf(errMsgFormat, WorkingNamespaceEnvVarName)
	}
	SourceNodeName, exists = os.LookupEnv(TargetNodeNameEnvVarName)
	if !exists {
		return fmt.Errorf(errMsgFormat, WorkingNamespaceEnvVarName)
	}
	TargetNodeName, exists = os.LookupEnv(SourceNodeNameEnvVarName)
	if !exists {
		return fmt.Errorf(errMsgFormat, WorkingNamespaceEnvVarName)
	}

	var err error
	LatencyCheckDuration, err = lookupEnvAsDuration(LatencyCheckDurationEnvVarName)
	if err != nil {
		return fmt.Errorf("Failed to load %s environment variable: %v", LatencyCheckDurationEnvVarName, err)
	}

	return nil
}

func lookupEnvAsDuration(varName string) (time.Duration, error) {
	duration := time.Duration(0)
	varValue, ok := os.LookupEnv(varName)
	if !ok {
		return duration, fmt.Errorf("Failed to load %s from environment", varName)
	}

	duration, err := time.ParseDuration(varValue)
	if err != nil {
		return duration, fmt.Errorf("Failed to convert %s value to time.Duration", varName)
	}
	return duration, nil
}
