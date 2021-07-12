module github.com/openshift/odo

go 1.15

require (
	github.com/Netflix/go-expect v0.0.0-20201125194554-85d881c3777e
	github.com/blang/semver v3.5.1+incompatible
	github.com/devfile/api/v2 v2.1.0
	github.com/devfile/library v1.0.0
	github.com/devfile/registry-support/index/generator v0.0.0-20210407161420-cd279527f873
	github.com/devfile/registry-support/registry-library v0.0.0-20210407161420-cd279527f873
	github.com/docker/docker v17.12.0-ce-rc1.0.20200916142827-bd33bbf0497b+incompatible
	github.com/docker/go-connections v0.4.1-0.20200120150455-7dc0a2d6ddce
	github.com/fatih/color v1.10.0
	github.com/frapposelli/wwhrd v0.4.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-git/go-git/v5 v5.2.0
	github.com/gobwas/glob v0.2.3
	github.com/golang/mock v1.5.0
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/kubernetes-sigs/service-catalog v0.3.1
	github.com/kylelemons/godebug v1.1.0
	github.com/mattn/go-colorable v0.1.8
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/openshift/api v0.0.0-20201216151826-78a19e96f9eb
	github.com/openshift/client-go v0.0.0-20201214125552-e615e336eb49
	github.com/openshift/library-go v0.0.0-20210106214821-c4d0b9c8d55f
	github.com/openshift/oc v0.0.0-alpha.0.0.20210325095525-2513fdbb36e2
	github.com/operator-framework/api v0.3.20
	github.com/operator-framework/operator-lifecycle-manager v0.17.0
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/posener/complete v1.1.1
	github.com/redhat-developer/service-binding-operator v0.8.0
	github.com/securego/gosec/v2 v2.8.0
	github.com/segmentio/backo-go v0.0.0-20200129164019-23eae7c10bd3 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.7.5
	github.com/tidwall/sjson v1.1.6
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	github.com/zalando/go-keyring v0.1.1
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/term v0.0.0-20210317153231-de623e64d2a6
	golang.org/x/tools v0.1.2-0.20210512205948-8287d5da45e4 // indirect
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/segmentio/analytics-go.v3 v3.1.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/cli-runtime v0.20.1
	k8s.io/client-go v0.20.1
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.4.0
	k8s.io/kubectl v0.20.1
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/yaml v1.2.0

)

replace (
	bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.7
	github.com/apcera/gssapi => github.com/openshift/gssapi v0.0.0-20161010215902-5fb4217df13b
	github.com/containers/image => github.com/openshift/containers-image v0.0.0-20190130162819-76de87591e9d
	github.com/docker/docker => github.com/docker/docker v1.4.2-0.20191121165722-d1d5f6476656
	k8s.io/apimachinery => github.com/openshift/kubernetes-apimachinery v0.0.0-20210108114224-194a87c5b03a
	k8s.io/cli-runtime => github.com/openshift/kubernetes-cli-runtime v0.0.0-20210108114725-2ff6add1e911
	k8s.io/client-go => github.com/openshift/kubernetes-client-go v0.0.0-20210108114446-0829bdd68114
	k8s.io/kubectl => github.com/openshift/kubernetes-kubectl v0.0.0-20210108115031-c0d78c0aeda3
)
