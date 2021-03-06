/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2020 Red Hat, Inc.
 *
 */

package libvmi

import (
	kvirtv1 "kubevirt.io/api/core/v1"
)

// Default VMI values
const (
	DefaultTestGracePeriod int64 = 0
	DefaultVmiName               = "testvmi"
)

const containerDiskFedoraTestTooling = "quay.io/kubevirt/fedora-with-test-tooling-container-disk:v0.49.0"

// NewFedora instantiates a new Fedora based VMI configuration,
// building its extra properties based on the specified With* options.
func NewFedora(opts ...Option) *kvirtv1.VirtualMachineInstance {
	fedoraOptions := []Option{
		WithTerminationGracePeriod(DefaultTestGracePeriod),
		WithResourceMemory("512M"),
		WithRng(),
		WithContainerImage(containerDiskFedoraTestTooling),
	}
	opts = append(fedoraOptions, opts...)
	return New(RandName(DefaultVmiName), opts...)
}
