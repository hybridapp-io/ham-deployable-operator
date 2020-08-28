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
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	hdplName      = "output"
	hdplNamespace = "output-ns"
	hdplOutputNS  = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hdplNamespace,
		},
	}

	hdDeployable = &appv1alpha1.Deployable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hdplName,
			Namespace: hdplNamespace,
		},
		Spec: appv1alpha1.DeployableSpec{},
	}

	payloadConfigMap = &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "payload",
			Namespace:   fooDeployer.Namespace,
			Labels:      map[string]string{appv1alpha1.OutputOf: hdDeployable.Name},
			Annotations: map[string]string{appv1alpha1.OutputOf: hdDeployable.Namespace + "/" + hdDeployable.Name},
		},
		Data: map[string]string{"myconfig": "foo"},
	}

	payloadEndpoint = &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Endpoints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "payload",
			Namespace:   fooDeployer.Namespace,
			Labels:      map[string]string{appv1alpha1.OutputOf: hdDeployable.Name},
			Annotations: map[string]string{appv1alpha1.OutputOf: hdDeployable.Namespace + "/" + hdDeployable.Name},
		},
	}

	payloadSecret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "payload",
			Namespace:   fooDeployer.Namespace,
			Labels:      map[string]string{appv1alpha1.OutputOf: hdDeployable.Name},
			Annotations: map[string]string{appv1alpha1.OutputOf: hdDeployable.Namespace + "/" + hdDeployable.Name},
		},
	}

	templateConfigMap = appv1alpha1.HybridTemplate{
		DeployerType: "configmap",
		Template: &runtime.RawExtension{
			Object: payloadConfigMap,
		},
	}
	templateEndpoints = appv1alpha1.HybridTemplate{
		DeployerType: "endpoints",
		Template: &runtime.RawExtension{
			Object: payloadEndpoint,
		},
	}
	templateSecret = appv1alpha1.HybridTemplate{
		DeployerType: "secret",
		Template: &runtime.RawExtension{
			Object: payloadSecret,
		},
	}
)

func TestDeployableStatus(t *testing.T) {
	g := NewWithT(t)
	hdDeployable.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateConfigMap,
			templateEndpoints,
			templateSecret,
		},
	}

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)
	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	hdNS := hdplOutputNS.DeepCopy()
	g.Expect(c.Create(context.TODO(), hdNS)).To(Succeed())

	// configmap deployer
	deployerConfigMap := fooDeployer.DeepCopy()
	deployerConfigMap.Name = "configmap"
	deployerConfigMap.Spec.Type = "configmap"

	deployerConfigMap.Spec.Scope = apiextensions.ClusterScoped
	deployerConfigMap.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), deployerConfigMap)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), deployerConfigMap)

	// endpoints deployer
	deployerEndpoints := fooDeployer.DeepCopy()
	deployerEndpoints.Name = "endpoints"
	deployerEndpoints.Spec.Type = "endpoints"
	deployerEndpoints.Spec.Scope = apiextensions.ClusterScoped
	deployerEndpoints.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"endpoints"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), deployerEndpoints)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), deployerEndpoints)

	// secret deployer
	deployerSecret := fooDeployer.DeepCopy()
	deployerSecret.Name = "secret"
	deployerSecret.Spec.Type = "secret"
	deployerSecret.Spec.Scope = apiextensions.ClusterScoped
	deployerSecret.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), deployerSecret)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), deployerSecret)

	//Expect payload is created in payload namespace on hdDeployable create
	hdpl := hdDeployable.DeepCopy()
	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployerConfigMap.Name,
				Namespace: deployerConfigMap.Namespace,
			},
		},
	}

	g.Expect(c.Create(context.TODO(), hdpl)).To(Succeed())

	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name])).ToNot(BeNil())

	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name]).LastUpdateTime).ToNot(BeNil())
	firstUpdateTime := hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].LastUpdateTime

	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].Outputs[0].APIVersion)).To(Equal("v1"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].Outputs[0].Kind)).To(Equal("ConfigMap"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].Outputs[0].Name)).To(Equal("payload"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].Outputs[0].Namespace)).To(Equal(fooDeployer.Namespace))

	// update placement to use the endpoints deployer in addition to the configmap deployer
	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployerConfigMap.Name,
				Namespace: deployerConfigMap.Namespace,
			},
			{
				Name:      deployerEndpoints.Name,
				Namespace: deployerEndpoints.Namespace,
			},
		},
	}
	g.Expect(c.Update(context.TODO(), hdpl)).To(Succeed())
	// wait for create
	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name])).ToNot(BeNil())

	// no change to the configmap hdpl status
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].LastUpdateTime)).To(Equal(firstUpdateTime))

	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployerEndpoints.Name,
				Namespace: deployerEndpoints.Namespace,
			},
		},
	}
	g.Expect(c.Update(context.TODO(), hdpl)).To(Succeed())
	// wait for create
	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())
	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name]).LastUpdateTime).ToNot(BeNil())

	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name].Outputs[0].APIVersion)).To(Equal("v1"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name].Outputs[0].Kind)).To(Equal("Endpoints"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name].Outputs[0].Name)).To(Equal("payload"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name].Outputs[0].Namespace)).To(Equal(fooDeployer.Namespace))

	// update placement to use the secret deployer
	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployerSecret.Name,
				Namespace: deployerSecret.Namespace,
			},
		},
	}
	g.Expect(c.Update(context.TODO(), hdpl)).To(Succeed())
	// wait for create
	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	// secrets  also triggers output mapper reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive())

	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)

	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name])).ToNot(BeNil())
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name].Outputs[0].APIVersion)).To(Equal("v1"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name].Outputs[0].Kind)).To(Equal("Secret"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name].Outputs[0].Name)).To(Equal("payload"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name].Outputs[0].Namespace)).To(Equal(fooDeployer.Namespace))

	c.Delete(context.TODO(), hdpl)
	g.Eventually(requests, timeout, interval).Should(Receive())

}
