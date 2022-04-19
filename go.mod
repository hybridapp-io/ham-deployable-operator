module github.com/hybridapp-io/ham-deployable-operator

go 1.16

require (
	github.com/hybridapp-io/ham-placement v0.0.0-20210225195735-3057d58bb101
	github.com/onsi/gomega v1.10.5
	github.com/open-cluster-management/api v0.0.0-20200610161514-939cead3902c
	github.com/open-cluster-management/multicloud-operators-deployable v0.0.0-20200603180154-d1d17d718c30
	github.com/open-cluster-management/multicloud-operators-placementrule v1.0.1-2020-05-28-18-29-00.0.20200603172904-efde26079087
	github.com/operator-framework/operator-sdk v0.18.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.18.3
	k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v13.0.0+incompatible
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2
)
