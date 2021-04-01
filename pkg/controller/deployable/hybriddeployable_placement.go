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

package deployable

import (
	"context"

	genericerrors "errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	hdplutils "github.com/hybridapp-io/ham-deployable-operator/pkg/utils"
	prulev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	placementv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
	placementutils "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/utils"
)

// Top priority: placementRef, ignore others
// Next priority: clusterNames, ignore selector
// Bottomline: Use label selector
func (r *ReconcileHybridDeployable) getDeployersByPlacement(instance *corev1alpha1.Deployable) ([]*prulev1alpha1.Deployer, error) {
	if instance == nil || instance.Spec.Placement == nil {
		return nil, nil
	}

	var deployers []*prulev1alpha1.Deployer

	var err error

	if instance.Spec.Placement.PlacementRef != nil {
		deployers, err = r.getDeployersByPlacementReference(instance)
		return deployers, err
	}

	if instance.Spec.Placement.Deployers != nil {
		for _, dplyref := range instance.Spec.Placement.Deployers {
			deployer := &prulev1alpha1.Deployer{}

			err = r.Get(context.TODO(), types.NamespacedName{Name: dplyref.Name, Namespace: dplyref.Namespace}, deployer)
			if err != nil {
				klog.V(packageDetailLogLevel).Info("Trying to obtain object from deployers list, but got error: ", err)
				continue
			}

			annotations := deployer.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}

			deployer.SetAnnotations(annotations)

			hdplutils.SetInClusterDeployer(deployer)
			deployers = append(deployers, deployer)
		}

		return deployers, err
	}

	if instance.Spec.Placement.DeployerLabels != nil {
		deployers, err = r.getDeployersByLabelSelector(instance.Spec.Placement.DeployerLabels)
		return deployers, err
	}

	return nil, nil
}

func (r *ReconcileHybridDeployable) getDeployersByLabelSelector(deployerLabels *metav1.LabelSelector) ([]*prulev1alpha1.Deployer, error) {
	clSelector, err := placementutils.ConvertLabels(deployerLabels)
	if err != nil {
		return nil, err
	}

	klog.V(packageDetailLogLevel).Info("Using Cluster LabelSelector ", clSelector)

	dplylist := &prulev1alpha1.DeployerList{}

	err = r.List(context.TODO(), dplylist, &client.ListOptions{LabelSelector: clSelector})

	if err != nil && !errors.IsNotFound(err) {
		klog.Error("Listing clusters and found error: ", err)
		return nil, err
	}

	klog.V(packageDetailLogLevel).Info("listed deployers:", dplylist.Items)

	var deployers []*prulev1alpha1.Deployer

	for _, dply := range dplylist.Items {
		deployer := dply.DeepCopy()

		annotations := deployer.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}

		deployer.SetAnnotations(annotations)

		hdplutils.SetInClusterDeployer(deployer)
		deployers = append(deployers, deployer)
	}

	return deployers, err
}

func (r *ReconcileHybridDeployable) getDeployersByPlacementReference(instance *corev1alpha1.Deployable) ([]*prulev1alpha1.Deployer, error) {
	var deployers []*prulev1alpha1.Deployer

	// hpr decisions can be either deployers or clusters for now
	decisions, err := r.getPlacementDecisions(instance)
	if err != nil {
		klog.Error("Failed to retrieve placement rule decisions with error: ", err)
		return nil, err
	}
	if decisions != nil {
		for _, decision := range decisions {
			if decision.Kind == corev1alpha1.DeployerGVK.Kind && decision.APIVersion == corev1alpha1.DeployerGVK.Group+"/"+corev1alpha1.DeployerGVK.Version {
				deployer := &prulev1alpha1.Deployer{}
				key := types.NamespacedName{
					Namespace: decision.Namespace,
					Name:      decision.Name,
				}

				err = r.Get(context.TODO(), key, deployer)
				if err != nil {
					klog.Error("Failed find retrieve deployer ", key.String(), " with error ", err)
					continue
				}
				deployers = append(deployers, deployer)
			} else if decision.Kind == corev1alpha1.ClusterGVK.Kind && decision.APIVersion == corev1alpha1.ClusterGVK.Group+"/"+corev1alpha1.ClusterGVK.Version {
				dset := &prulev1alpha1.DeployerSet{}
				key := types.NamespacedName{
					Namespace: decision.Name,
					Name:      decision.Name,
				}
				deployer := &prulev1alpha1.Deployer{}
				deployer.Name = decision.Name
				deployer.Namespace = decision.Name
				deployer.Spec.Type = corev1alpha1.DefaultDeployerType

				err = r.Get(context.TODO(), key, dset)
				klog.V(packageDetailLogLevel).Info("Got Deployerset for cluster ", decision.Name, "/", decision.Name, " with err:", err, " result: ", dset)

				if err != nil {
					if !errors.IsNotFound(err) {
						continue
					}
				} else {
					if len(dset.Spec.Deployers) > 0 {
						dset.Spec.Deployers[0].Spec.DeepCopyInto(&deployer.Spec)

						if dset.Spec.DefaultDeployer != "" {
							for _, dply := range dset.Spec.Deployers {
								if dply.Key == dset.Spec.DefaultDeployer {
									dply.Spec.DeepCopyInto(&deployer.Spec)
									break
								}
							}
						}
						klog.V(packageDetailLogLevel).Info("Copyied deployer info:", deployer)
					}
				}

				klog.V(packageDetailLogLevel).Info("Adding deployer: ", deployer)

				annotations := deployer.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}

				deployer.SetAnnotations(annotations)
				hdplutils.SetRemoteDeployer(deployer)

				deployers = append(deployers, deployer)
			} else {
				klog.Error("Unsupported decision type {APIVersion: ", decision.APIVersion, " Kind: ", decision.Kind, "}")
				continue
			}
		}
		return deployers, nil
	}
	noPlacementError := genericerrors.New("No placement rule decisions found for hybrid deployable " + instance.Namespace + "/" + instance.Name)
	return nil, noPlacementError
}

func (r *ReconcileHybridDeployable) getPlacementDecisions(instance *corev1alpha1.Deployable) ([]corev1.ObjectReference, error) {
	if instance == nil || instance.Spec.Placement == nil || instance.Spec.Placement.PlacementRef == nil {
		return nil, nil
	}

	pref := instance.Spec.Placement.PlacementRef
	if pref != nil {
		// get placementpolicy resource
		pp := &prulev1alpha1.PlacementRule{}
		pkey := types.NamespacedName{
			Name:      pref.Name,
			Namespace: pref.Namespace,
		}

		if pref.Namespace == "" {
			pkey.Namespace = instance.Namespace
		}

		err := r.Get(context.TODO(), pkey, pp)
		if err != nil {
			oldpr := &placementv1.PlacementRule{}
			err := r.Get(context.TODO(), pkey, oldpr)
			if err != nil {
				if errors.IsNotFound(err) {
					klog.Warning("Failed to locate placement reference ", pref.Namespace+"/"+pref.Name)
					return nil, nil
				}
				return nil, err
			}
			clusters := make([]corev1.ObjectReference, len(oldpr.Status.Decisions))
			for index, decision := range oldpr.Status.Decisions {
				cluster := &corev1.ObjectReference{}
				cluster.Name = decision.ClusterName
				cluster.Namespace = decision.ClusterNamespace
				cluster.Kind = corev1alpha1.ClusterGVK.Kind
				cluster.APIVersion = corev1alpha1.ClusterGVK.Group + "/" + corev1alpha1.ClusterGVK.Version
				clusters[index] = *cluster
			}
			return clusters, nil
		}
		return pp.Status.Decisions, nil
	}
	klog.Info("No placement references found for deployable ", instance.Namespace+"/"+instance.Name)
	return nil, nil
}

func (r *ReconcileHybridDeployable) getChildren(request types.NamespacedName) (map[schema.GroupVersionResource]gvrChildrenMap, error) {
	children := make(map[schema.GroupVersionResource]gvrChildrenMap)

	nameLabel := map[string]string{
		corev1alpha1.HostingHybridDeployable: request.Name,
	}

	for gvr := range r.activeGVRMap {
		objlist, err := r.dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{LabelSelector: labels.Set(nameLabel).String()})
		if err != nil {
			return nil, err
		}

		klog.V(packageDetailLogLevel).Info("Processing active gvr ", gvr, " and got list: ", objlist.Items)

		gvkchildren := make(map[types.NamespacedName]metav1.Object)

		for i := range objlist.Items {
			obj := objlist.Items[i]
			annotations := obj.GetAnnotations()
			if annotations == nil {
				continue
			}

			if host, ok := annotations[corev1alpha1.HostingHybridDeployable]; ok {
				if host == request.String() {
					key := r.genObjectIdentifier(&obj)

					if _, ok := gvkchildren[key]; ok {
						klog.Info("Deleting redundant deployed object", obj.GetNamespace(), "/", obj.GetName())

						deletePolicy := metav1.DeletePropagationBackground

						_ = r.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Delete(context.TODO(),
							obj.GetName(), metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
					} else {
						gvkchildren[key] = obj.DeepCopy()
					}

					klog.V(packageDetailLogLevel).Info("Adding gvk child with key:", key)
				}
			}
		}

		children[gvr] = gvkchildren
	}

	return children, nil
}
