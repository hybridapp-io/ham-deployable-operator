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
	"testing"

	appv1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	hdplv1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	. "github.com/onsi/gomega"
	dplv1 "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	CONFIGMAP = "configmap"
	ENDPOINT  = "endpoint"
	RHACM     = "kubernetes"
)

var (
	hdplDepName            = "dependency"
	hdplDepNamespace       = "dependency-ns"
	applicationName        = "wordpress-01"
	appLabelSelector       = "app.kubernetes.io/name"
	mcServiceName          = "my-svc"
	mcName                 = "dependency-ns"
	hdplDependentName      = "dependent"
	hdplDependentNamespace = "dependent-ns"

	hdplDepNS = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hdplDepNamespace,
		},
	}

	hdplDependentNS = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hdplDependentNamespace,
		},
	}

	hdplDependency = &appv1alpha1.Deployable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hdplDepName,
			Namespace: hdplDepNamespace,
		},
		Spec: appv1alpha1.DeployableSpec{},
	}
	mcService = &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        mcServiceName,
			Namespace:   fooDeployer.Namespace,
			Annotations: map[string]string{appLabelSelector: applicationName + "2"},
			Labels:      map[string]string{appLabelSelector: applicationName + "2"},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port: 3306,
				},
			},
		},
	}
	mcService2 = &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        mcServiceName + "2",
			Namespace:   fooDeployer.Namespace,
			Annotations: map[string]string{appLabelSelector: applicationName},
			Labels:      map[string]string{appLabelSelector: applicationName},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port: 3307,
				},
			},
		},
	}
	svcDeployable = &dplv1.Deployable{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployable",
			APIVersion: "apps.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcServiceName,
			Namespace: fooDeployer.Namespace,
			Annotations: map[string]string{
				hdplv1alpha1.AnnotationHybridDiscovery: hdplv1alpha1.HybridDiscoveryEnabled,
			},
			Labels: map[string]string{
				hdplv1alpha1.AnnotationHybridDiscovery: hdplv1alpha1.HybridDiscoveryEnabled,
			},
		},
		Spec: dplv1.DeployableSpec{
			Template: &runtime.RawExtension{
				Object: mcService,
			},
		},
	}
	svcDeployable2 = &dplv1.Deployable{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployable",
			APIVersion: "apps.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcServiceName + "2",
			Namespace: fooDeployer.Namespace,
			Annotations: map[string]string{
				hdplv1alpha1.AnnotationHybridDiscovery: hdplv1alpha1.HybridDiscoveryEnabled,
			},
			Labels: map[string]string{
				hdplv1alpha1.AnnotationHybridDiscovery: hdplv1alpha1.HybridDiscoveryEnabled,
			},
		},
		Spec: dplv1.DeployableSpec{
			Template: &runtime.RawExtension{
				Object: mcService2,
			},
		},
	}
	svcDeployableRef = &corev1.ObjectReference{
		Name:       mcServiceName,
		Kind:       "Deployable",
		APIVersion: "apps.open-cluster-management.io/v1",
	}
	svcDeployableRef2 = &corev1.ObjectReference{
		Name:       mcServiceName + "2",
		Namespace:  fooDeployer.Namespace,
		Kind:       "Deployable",
		APIVersion: "apps.open-cluster-management.io/v1",
	}

	templateRHACM = appv1alpha1.HybridTemplate{
		DeployerType: RHACM,
		Template: &runtime.RawExtension{
			Object: svcDeployable,
		},
	}
	templateRHACM2 = appv1alpha1.HybridTemplate{
		DeployerType: RHACM,
		Template: &runtime.RawExtension{
			Object: svcDeployable2,
		},
	}
	cm = &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cm",
			Namespace: fooDeployer.Namespace,
		},
		Data: map[string]string{"myconfig": "foo"},
	}

	ep = &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Endpoints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ep",
			Namespace: hdplDepNamespace,
		},
	}

	epRef = &corev1.ObjectReference{
		Name:       "ep",
		Kind:       "Endpoints",
		APIVersion: "v1",
	}

	hdplRef = &corev1.ObjectReference{
		Name:       hdplDependentName,
		Kind:       hybriddeployableGVK.Kind,
		APIVersion: hybriddeployableGVK.Group + "/" + hybriddeployableGVK.Version,
	}

	templateCM = appv1alpha1.HybridTemplate{
		DeployerType: CONFIGMAP,
		Template: &runtime.RawExtension{
			Object: cm,
		},
	}

	dependentEP = &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Endpoints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ep",
			Namespace: hdplDependentNamespace,
		},
	}

	templateEndpoint = appv1alpha1.HybridTemplate{
		DeployerType: ENDPOINT,
		Template: &runtime.RawExtension{
			Object: dependentEP,
		},
	}
)

func TestDependency(t *testing.T) {
	g := NewWithT(t)
	hdplDependency.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateCM,
		},
	}

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests, _ := SetupTestReconcile(rec)
	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	hdNS := hdplDepNS.DeepCopy()
	g.Expect(c.Create(context.TODO(), hdNS)).To(Succeed())

	// configmap deployer
	deployer := fooDeployer.DeepCopy()
	deployer.Name = CONFIGMAP
	deployer.Spec.Type = CONFIGMAP

	deployer.Spec.Scope = apiextensions.ClusterScoped
	deployer.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), deployer)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), deployer)

	endpoint := ep.DeepCopy()
	g.Expect(c.Create(context.TODO(), endpoint)).To(Succeed())
	defer c.Delete(context.TODO(), endpoint)

	hdpl := hdplDependency.DeepCopy()
	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployer.Name,
				Namespace: deployer.Namespace,
			},
		},
	}
	hdpl.Spec.Dependencies = []corev1.ObjectReference{
		*epRef,
	}

	g.Expect(c.Create(context.TODO(), hdpl)).To(Succeed())

	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)

	// expect the dependency to be created
	depEndpoint := &corev1.Endpoints{}
	g.Expect(c.Get(context.TODO(), types.NamespacedName{
		Name:      "ep",
		Namespace: deployer.Namespace,
	}, depEndpoint)).To(Succeed())

	c.Delete(context.TODO(), hdpl)
	g.Eventually(requests, timeout, interval).Should(Receive())

}

func TestHybridDeployableDependency(t *testing.T) {
	g := NewWithT(t)

	hdplDependent := &appv1alpha1.Deployable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hdplDependentName,
			Namespace: hdplDependentNamespace,
		},
		Spec: appv1alpha1.DeployableSpec{
			HybridTemplates: []appv1alpha1.HybridTemplate{
				templateEndpoint,
			},
		},
	}

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests, _ := SetupTestReconcile(rec)
	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	hdNS := hdplDependentNS.DeepCopy()
	g.Expect(c.Create(context.TODO(), hdNS)).To(Succeed())

	dependentDeployer := fooDeployer.DeepCopy()
	dependentDeployer.Name = ENDPOINT
	dependentDeployer.Namespace = hdplDependentNamespace
	dependentDeployer.Spec.Type = ENDPOINT

	dependentDeployer.Spec.Scope = apiextensions.ClusterScoped
	dependentDeployer.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"endpoints"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), dependentDeployer)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), dependentDeployer)

	dependentHDPL := hdplDependent.DeepCopy()
	g.Expect(c.Create(context.TODO(), dependentHDPL)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	dependencyHDPL := hdplDependency.DeepCopy()
	dependencyHDPL.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateEndpoint,
		},
	}
	dependencyHDPL.Namespace = hdplDependentNamespace
	dependencyHDPL.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      dependentDeployer.Name,
				Namespace: dependentDeployer.Namespace,
			},
		},
	}
	dependencyHDPL.Spec.Dependencies = []corev1.ObjectReference{
		*hdplRef,
	}

	g.Expect(c.Create(context.TODO(), dependencyHDPL)).To(Succeed())

	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	c.Get(context.TODO(), types.NamespacedName{Name: dependencyHDPL.Name, Namespace: dependencyHDPL.Namespace}, dependencyHDPL)

	// expect the dependency to be created
	depHDPL := &appv1alpha1.Deployable{}
	g.Expect(c.Get(context.TODO(), types.NamespacedName{
		Name:      dependentHDPL.Name,
		Namespace: dependentHDPL.Namespace,
	}, depHDPL)).To(Succeed())

	c.Delete(context.TODO(), dependentHDPL)
	c.Delete(context.TODO(), dependencyHDPL)
	g.Eventually(requests, timeout, interval).Should(Receive())

}

func TestDeployableDependencyRefGVKEqualsDeployableGVK(t *testing.T) {
	g := NewWithT(t)
	hdplDependent := &appv1alpha1.Deployable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hdplDependentName,
			Namespace: hdplDependentNamespace,
		},
		Spec: appv1alpha1.DeployableSpec{
			HybridTemplates: []appv1alpha1.HybridTemplate{
				templateRHACM,
			},
		},
	}
	var c client.Client
	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())
	c = mgr.GetClient()
	rec := newReconciler(mgr)
	recFn, requests, _ := SetupTestReconcile(rec)
	g.Expect(add(mgr, recFn)).To(Succeed())
	stopMgr, mgrStopped := StartTestManager(mgr, g)
	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()
	dependentDeployer := fooDeployer.DeepCopy()
	dependentDeployer.Name = RHACM
	dependentDeployer.Namespace = fooDeployer.Namespace
	dependentDeployer.Spec.Type = RHACM
	dependentDeployer.Spec.Scope = apiextensions.ClusterScoped
	dependentDeployer.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"deployables"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), dependentDeployer)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), deployer)
	dependentHDPL := hdplDependent.DeepCopy()
	dependentHDPL.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      dependentDeployer.Name,
				Namespace: dependentDeployer.Namespace,
			},
		},
	}

	dependentHDPL.Spec.Dependencies = []corev1.ObjectReference{
		*svcDeployableRef,
	}
	g.Expect(c.Create(context.TODO(), dependentHDPL)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive())
	//  hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())
	// c.Get(context.TODO(), types.NamespacedName{Name: dependentHDPL.Name, Namespace: dependentHDPL.Namespace}, dependentHDPL)
	g.Expect(c.Get(context.TODO(), types.NamespacedName{Name: dependentHDPL.Name, Namespace: dependentHDPL.Namespace}, dependentHDPL)).To(Succeed())
	c.Delete(context.TODO(), dependentHDPL)
	g.Eventually(requests, timeout, interval).Should(Receive())

	hdplDependent2 := &appv1alpha1.Deployable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hdplDependentName + "2",
			Namespace: hdplDependentNamespace,
		},
		Spec: appv1alpha1.DeployableSpec{
			HybridTemplates: []appv1alpha1.HybridTemplate{
				templateRHACM2,
			},
		},
	}

	dependentHDPL2 := hdplDependent2.DeepCopy()

	dependentHDPL2.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      dependentDeployer.Name,
				Namespace: dependentDeployer.Namespace,
			},
		},
	}

	dependentHDPL2.Spec.Dependencies = []corev1.ObjectReference{
		*svcDeployableRef2,
	}
	g.Expect(c.Create(context.TODO(), dependentHDPL2)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive())
	//  hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	g.Expect(c.Get(context.TODO(), types.NamespacedName{Name: dependentHDPL2.Name, Namespace: dependentHDPL2.Namespace}, dependentHDPL2)).To(Succeed())

	c.Delete(context.TODO(), dependentHDPL2)
	g.Eventually(requests, timeout, interval).Should(Receive())
}
