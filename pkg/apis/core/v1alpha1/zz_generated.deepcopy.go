// +build !ignore_autogenerated

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

// Code generated by operator-sdk. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Deployable) DeepCopyInto(out *Deployable) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Deployable.
func (in *Deployable) DeepCopy() *Deployable {
	if in == nil {
		return nil
	}
	out := new(Deployable)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Deployable) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeployableList) DeepCopyInto(out *DeployableList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Deployable, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeployableList.
func (in *DeployableList) DeepCopy() *DeployableList {
	if in == nil {
		return nil
	}
	out := new(DeployableList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DeployableList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeployableSpec) DeepCopyInto(out *DeployableSpec) {
	*out = *in
	if in.HybridTemplates != nil {
		in, out := &in.HybridTemplates, &out.HybridTemplates
		*out = make([]HybridTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Placement != nil {
		in, out := &in.Placement, &out.Placement
		*out = new(HybridPlacement)
		(*in).DeepCopyInto(*out)
	}
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make([]v1.ObjectReference, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeployableSpec.
func (in *DeployableSpec) DeepCopy() *DeployableSpec {
	if in == nil {
		return nil
	}
	out := new(DeployableSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeployableStatus) DeepCopyInto(out *DeployableStatus) {
	*out = *in
	if in.PerDeployerStatus != nil {
		in, out := &in.PerDeployerStatus, &out.PerDeployerStatus
		*out = make(map[string]PerDeployerStatus, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeployableStatus.
func (in *DeployableStatus) DeepCopy() *DeployableStatus {
	if in == nil {
		return nil
	}
	out := new(DeployableStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybridPlacement) DeepCopyInto(out *HybridPlacement) {
	*out = *in
	if in.Deployers != nil {
		in, out := &in.Deployers, &out.Deployers
		*out = make([]v1.ObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.DeployerLabels != nil {
		in, out := &in.DeployerLabels, &out.DeployerLabels
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.PlacementRef != nil {
		in, out := &in.PlacementRef, &out.PlacementRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybridPlacement.
func (in *HybridPlacement) DeepCopy() *HybridPlacement {
	if in == nil {
		return nil
	}
	out := new(HybridPlacement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybridTemplate) DeepCopyInto(out *HybridTemplate) {
	*out = *in
	if in.Template != nil {
		in, out := &in.Template, &out.Template
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybridTemplate.
func (in *HybridTemplate) DeepCopy() *HybridTemplate {
	if in == nil {
		return nil
	}
	out := new(HybridTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PerDeployerStatus) DeepCopyInto(out *PerDeployerStatus) {
	*out = *in
	in.ResourceUnitStatus.DeepCopyInto(&out.ResourceUnitStatus)
	if in.Outputs != nil {
		in, out := &in.Outputs, &out.Outputs
		*out = make([]v1.ObjectReference, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PerDeployerStatus.
func (in *PerDeployerStatus) DeepCopy() *PerDeployerStatus {
	if in == nil {
		return nil
	}
	out := new(PerDeployerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceUnitStatus) DeepCopyInto(out *ResourceUnitStatus) {
	*out = *in
	if in.LastUpdateTime != nil {
		in, out := &in.LastUpdateTime, &out.LastUpdateTime
		*out = (*in).DeepCopy()
	}
	if in.ResourceStatus != nil {
		in, out := &in.ResourceStatus, &out.ResourceStatus
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceUnitStatus.
func (in *ResourceUnitStatus) DeepCopy() *ResourceUnitStatus {
	if in == nil {
		return nil
	}
	out := new(ResourceUnitStatus)
	in.DeepCopyInto(out)
	return out
}