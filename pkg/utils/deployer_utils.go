// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"regexp"
	"strings"

	appv1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	prulev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
)

const (
	dns1035regex     = "[a-z]([-a-z0-9]*[a-z0-9])?"
	illegalCharRegex = ":"
)

func IsInClusterDeployer(deployer *prulev1alpha1.Deployer) bool {
	incluster := true

	annotations := deployer.GetAnnotations()
	if annotations != nil {
		if in, ok := annotations[appv1alpha1.DeployerInCluster]; ok && in == "false" {
			incluster = false
		}
	}

	return incluster
}

func SetInClusterDeployer(deployer *prulev1alpha1.Deployer) {
	annotations := deployer.GetAnnotations()
	annotations[appv1alpha1.DeployerInCluster] = "true"
	deployer.SetAnnotations(annotations)
}

func SetRemoteDeployer(deployer *prulev1alpha1.Deployer) {
	annotations := deployer.GetAnnotations()
	annotations[appv1alpha1.DeployerInCluster] = "false"
	deployer.SetAnnotations(annotations)
}

// StripVersion removes the version part of a GV
func StripVersion(gv string) string {
	if gv == "" {
		return gv
	}

	re := regexp.MustCompile(`^[vV][0-9].*`)
	// If it begins with only version, (group is nil), return empty string which maps to core group
	if re.MatchString(gv) {
		return ""
	}

	return strings.Split(gv, "/")[0]
}

func StripGroup(gv string) string {
	re := regexp.MustCompile(`^[vV][0-9].*`)
	// If it begins with only version, (group is nil), return empty string which maps to core group
	if re.MatchString(gv) {
		return gv
	}

	return strings.Split(gv, "/")[1]
}

func IsNamespaceScoped(deployer *prulev1alpha1.Deployer) bool {
	return deployer.Spec.Scope == "" || deployer.Spec.Scope == apiextensions.NamespaceScoped
}

func TruncateString(str string, num int) string {
	truncated := str
	r, _ := regexp.Compile(illegalCharRegex)
	truncated = r.ReplaceAllString(truncated, "-")
	if len(str) > num {
		truncated = str[0:num]
		r, _ := regexp.Compile(dns1035regex)
		truncated = r.FindString(truncated)
	}
	return truncated
}
