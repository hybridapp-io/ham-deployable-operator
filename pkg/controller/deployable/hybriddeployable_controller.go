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
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	appv1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	"github.com/hybridapp-io/ham-deployable-operator/pkg/utils"
	prulev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	dplv1 "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
	placementv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
)

const (
	packageInfoLogLevel   = 3
	packageDetailLogLevel = 5
)

var (
	rhacmEnabled       = false
	rhacmDeployableGVK = schema.GroupVersionKind{
		Group:   "apps.open-cluster-management.io",
		Version: "v1",
		Kind:    "Deployable",
	}
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Deployable Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	reconciler := &ReconcileHybridDeployable{Client: mgr.GetClient()}

	err := reconciler.initRegistry(mgr.GetConfig())
	if _, ok := reconciler.gvkGVRMap[rhacmDeployableGVK]; ok {
		rhacmEnabled = true
		klog.Info("RedHat Advanced Cluster Management(RHACM) is enabled in this environment")
	} else {
		klog.Info("RedHat Advanced Cluster Management(RHACM) is not enabled in this environment")
	}
	if err != nil {
		klog.Error("Failed to initialize hybrid deployable registry with error:", err)
		return nil
	}

	reconciler.eventRecorder, err = utils.NewEventRecorder(mgr.GetConfig(), mgr.GetScheme())
	if err != nil {
		klog.Error("Failed to initialize hybrid deployable registry with error:", err)
		return nil
	}

	return reconciler
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	// Create a new controller
	c, err := controller.New("hybriddeployable-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Deployable
	err = c.Watch(
		&source.Kind{Type: &appv1alpha1.Deployable{}},
		&handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{
			Type: &appv1alpha1.Deployable{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: &hdplstatusMapper{mgr.GetClient()},
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				newpr := e.ObjectNew.(*appv1alpha1.Deployable)
				oldpr := e.ObjectOld.(*appv1alpha1.Deployable)

				return !reflect.DeepEqual(oldpr.Status, newpr.Status)
			},
		},
	)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{
			Type: &prulev1alpha1.DeployerSet{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: &deployersetMapper{mgr.GetClient()},
		},
	)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{
			Type: &prulev1alpha1.Deployer{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: &deployerMapper{mgr.GetClient()},
		},
	)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{
			Type: &prulev1alpha1.PlacementRule{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: &placementruleMapper{mgr.GetClient()},
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				newpr := e.ObjectNew.(*prulev1alpha1.PlacementRule)
				oldpr := e.ObjectOld.(*prulev1alpha1.PlacementRule)

				return !reflect.DeepEqual(oldpr.Status, newpr.Status)
			},
		},
	)

	if err != nil {
		return err
	}

	// watch RHACM placement rules only if RHACM is installed
	if rhacmEnabled {
		err = c.Watch(
			&source.Kind{
				Type: &placementv1.PlacementRule{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: &placementruleMapper{mgr.GetClient()},
			},
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					newpr := e.ObjectNew.(*placementv1.PlacementRule)
					oldpr := e.ObjectOld.(*placementv1.PlacementRule)

					return !reflect.DeepEqual(oldpr.Status, newpr.Status)
				},
			},
		)

		if err != nil {
			return err
		}

		// watch on hybrid-discovery annotation of deployables, but only if RHACM is installed
		err = c.Watch(
			&source.Kind{
				Type: &dplv1.Deployable{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: &deployableMapper{mgr.GetClient()},
			},
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					newDeployable := e.ObjectNew.(*dplv1.Deployable)
					oldDeployable := e.ObjectOld.(*dplv1.Deployable)
					if !reflect.DeepEqual(oldDeployable.Status, newDeployable.Status) {
						return true
					}
					// discovery annotation = completed on new
					if _, completedNew := newDeployable.GetAnnotations()[appv1alpha1.AnnotationHybridDiscovery]; completedNew &&
						newDeployable.GetAnnotations()[appv1alpha1.AnnotationHybridDiscovery] == appv1alpha1.HybridDiscoveryCompleted {
						// discovery annotation != completed on old
						if _, completedOld := oldDeployable.GetAnnotations()[appv1alpha1.AnnotationHybridDiscovery]; !completedOld ||
							oldDeployable.GetAnnotations()[appv1alpha1.AnnotationHybridDiscovery] != appv1alpha1.HybridDiscoveryCompleted {
							// hosted deployable
							if _, hostedNew := newDeployable.GetAnnotations()[appv1alpha1.HostingHybridDeployable]; hostedNew {
								return true
							}
						}
					}
					return false
				},
				CreateFunc: func(e event.CreateEvent) bool {
					deployable := e.Object.(*dplv1.Deployable)
					if _, completedNew := deployable.GetAnnotations()[appv1alpha1.AnnotationHybridDiscovery]; completedNew &&
						deployable.GetAnnotations()[appv1alpha1.AnnotationHybridDiscovery] == appv1alpha1.HybridDiscoveryCompleted {
						// hosted deployable
						if _, hostedNew := deployable.GetAnnotations()[appv1alpha1.HostingHybridDeployable]; hostedNew {
							return true
						}
					}
					return false
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					return false
				},
				GenericFunc: func(e event.GenericEvent) bool {
					return false
				},
			},
		)
		if err != nil {
			return err
		}

		err = c.Watch(
			&source.Kind{
				Type: &dplv1.Deployable{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: &outputMapper{mgr.GetClient()},
			},
		)
		if err != nil {
			return err
		}
	}

	err = c.Watch(
		&source.Kind{
			Type: &corev1.Endpoints{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: &outputMapper{mgr.GetClient()},
		},
	)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{
			Type: &corev1.Secret{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: &outputMapper{mgr.GetClient()},
		},
	)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{
			Type: &corev1.ConfigMap{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: &outputMapper{mgr.GetClient()},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileHybridDeployable implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileHybridDeployable{}

type hdplstatusMapper struct {
	client.Client
}

func (mapper *hdplstatusMapper) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request

	hdplList := &appv1alpha1.DeployableList{}

	err := mapper.List(context.TODO(), hdplList, &client.ListOptions{})
	if err != nil {
		klog.Error("Failed to list hybrid deployables for deployerset with error:", err)
		return requests
	}

	depname := obj.Meta.GetName()
	depns := obj.Meta.GetNamespace()

	for _, hdpl := range hdplList.Items {
		if hdpl.Spec.Dependencies == nil {
			continue
		}

		for _, depref := range hdpl.Spec.Dependencies {
			if depref.Name != depname {
				continue
			}

			if depref.Namespace != "" && depref.Namespace != depns {
				continue
			}

			if depref.Namespace == "" && hdpl.Namespace != depns {
				continue
			}

			objkey := types.NamespacedName{
				Name:      hdpl.GetName(),
				Namespace: hdpl.GetNamespace(),
			}

			requests = append(requests, reconcile.Request{NamespacedName: objkey})
		}
	}

	return requests
}

type placementruleMapper struct {
	client.Client
}

func (mapper *placementruleMapper) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request

	hdplList := &appv1alpha1.DeployableList{}

	err := mapper.List(context.TODO(), hdplList, &client.ListOptions{})
	if err != nil {
		klog.Error("Failed to list hybrid deployables for deployerset with error:", err)
		return requests
	}

	prname := obj.Meta.GetName()
	prnamespace := obj.Meta.GetNamespace()

	for _, hdpl := range hdplList.Items {
		if hdpl.Spec.Placement != nil && hdpl.Spec.Placement.PlacementRef != nil && hdpl.Spec.Placement.PlacementRef.Name == prname {
			pref := hdpl.Spec.Placement.PlacementRef
			if pref.Namespace != "" && pref.Namespace != prnamespace {
				continue
			}

			if pref.Namespace == "" && prnamespace != hdpl.Namespace {
				continue
			}

			cp4mcmprgvk := schema.GroupVersionKind{
				Group:   "app.ibm.com",
				Version: "v1alpha1",
				Kind:    "PlacementRule",
			}
			hybridprgvk := schema.GroupVersionKind{
				Group:   "core.hybridapp.io",
				Version: "v1alpha1",
				Kind:    "PlacementRule",
			}
			if !pref.GroupVersionKind().Empty() && pref.GroupVersionKind().String() != cp4mcmprgvk.String() &&
				pref.GroupVersionKind().String() != hybridprgvk.String() {
				continue
			}

			objkey := types.NamespacedName{
				Name:      hdpl.GetName(),
				Namespace: hdpl.GetNamespace(),
			}

			requests = append(requests, reconcile.Request{NamespacedName: objkey})
		}
	}

	return requests
}

type deployersetMapper struct {
	client.Client
}

func (mapper *deployersetMapper) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request

	hdplList := &appv1alpha1.DeployableList{}

	err := mapper.List(context.TODO(), hdplList, &client.ListOptions{})
	if err != nil {
		klog.Error("Failed to list hybrid deployables for deployerset with error:", err)
		return requests
	}

	for _, hdpl := range hdplList.Items {
		// only reconcile with when placement is set and not using ref
		if hdpl.Spec.Placement == nil {
			continue
		}

		if hdpl.Spec.Placement.PlacementRef != nil {
			objkey := types.NamespacedName{
				Name:      hdpl.GetName(),
				Namespace: hdpl.GetNamespace(),
			}

			requests = append(requests, reconcile.Request{NamespacedName: objkey})
		}
	}

	return requests
}

type deployerMapper struct {
	client.Client
}

func (mapper *deployerMapper) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request

	hdplList := &appv1alpha1.DeployableList{}
	err := mapper.List(context.TODO(), hdplList, &client.ListOptions{})

	if err != nil {
		klog.Error("Failed to list hybrid deployables for deployerset with error:", err)
		return requests
	}

	dplyname := obj.Meta.GetName()
	dplynamespace := obj.Meta.GetNamespace()

	for _, hdpl := range hdplList.Items {
		// only reconcile with when placement is set and not using ref
		if hdpl.Spec.Placement == nil {
			continue
		}

		if hdpl.Spec.Placement.PlacementRef == nil && hdpl.Spec.Placement.Deployers != nil {
			matched := false

			for _, cn := range hdpl.Spec.Placement.Deployers {
				if cn.Name == dplyname && cn.Namespace == dplynamespace {
					matched = true
				}
			}

			if !matched {
				continue
			}
		}

		objkey := types.NamespacedName{
			Name:      hdpl.GetName(),
			Namespace: hdpl.GetNamespace(),
		}

		requests = append(requests, reconcile.Request{NamespacedName: objkey})
	}

	return requests
}

type deployableMapper struct {
	client.Client
}

func (mapper *deployableMapper) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request

	if hdpl, ok := obj.Meta.GetAnnotations()[appv1alpha1.HostingHybridDeployable]; ok {
		if len(strings.Split(hdpl, "/")) == 2 {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: strings.Split(hdpl, "/")[0],
					Name:      strings.Split(hdpl, "/")[1]}})
		}
	}

	return requests
}

type outputMapper struct {
	client.Client
}

func (mapper *outputMapper) Map(obj handler.MapObject) []reconcile.Request {
	var requests []reconcile.Request

	annotations := obj.Meta.GetAnnotations()
	if annotations == nil {
		return nil
	}

	var hdplkey string

	var ok bool

	if hdplkey, ok = annotations[appv1alpha1.OutputOf]; !ok {
		return nil
	}

	nn := strings.Split(hdplkey, "/")
	key := types.NamespacedName{
		Namespace: nn[0],
		Name:      nn[1],
	}

	requests = append(requests, reconcile.Request{NamespacedName: key})
	return requests
}

// ReconcileHybridDeployable reconciles a Deployable object
type ReconcileHybridDeployable struct {
	client.Client
	eventRecorder *utils.EventRecorder
	hybridDeployableRegistry
}

// Reconcile reads that state of the cluster for a Deployable object and makes changes based on the state read
// and what is in the Deployable.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileHybridDeployable) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling Deployable:", request)

	// Fetch the Deployable instance
	instance := &appv1alpha1.Deployable{}

	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			children, err := r.getChildren(request.NamespacedName)
			if err == nil {
				r.purgeChildren(children)
			}

			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	deployers, err := r.getDeployersByPlacement(instance)
	if err != nil {
		klog.Error("Failed to get deployers from hybrid deployable spec placement with error:", err)
		return reconcile.Result{}, nil
	}

	children, err := r.getChildren(request.NamespacedName)
	if err != nil {
		klog.Error("Failed to get existing objects for hybriddeployable with error:", err)
	}

	instance.Status.PerDeployerStatus = nil

	err = r.deployResourceByDeployers(instance, deployers, children)
	if err != nil {
		if len(deployers) > 0 && deployers[0] != nil {
			key := deployers[0].Namespace + "/" + deployers[0].Name
			if instance.Status.PerDeployerStatus[key].LastUpdateTime != nil {
				sleepInterval := 2 * (time.Now().Unix() -
					instance.Status.PerDeployerStatus[deployers[0].Name].LastUpdateTime.Time.Unix())
				time.Sleep(time.Duration(sleepInterval) * time.Second)
			}
		}
		return reconcile.Result{}, err
	}
	r.purgeChildren(children)

	err = r.updateStatus(instance)
	if err != nil {
		klog.Info("Failed to update status with error:", err)
	}

	return reconcile.Result{}, nil
}
