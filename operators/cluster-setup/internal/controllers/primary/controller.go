package primary

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagerMetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"io"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"net/http"
	crdsv1 "operators.kloudlite.io/apis/crds/v1"
	"sigs.k8s.io/yaml"
	"strings"
	"time"

	"github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"operators.kloudlite.io/apis/cluster-setup/v1"
	lc "operators.kloudlite.io/operators/cluster-setup/internal/constants"
	"operators.kloudlite.io/operators/cluster-setup/internal/env"
	"operators.kloudlite.io/operators/cluster-setup/internal/templates"
	"operators.kloudlite.io/pkg/constants"
	fn "operators.kloudlite.io/pkg/functions"
	"operators.kloudlite.io/pkg/harbor"
	kHttp "operators.kloudlite.io/pkg/http"
	"operators.kloudlite.io/pkg/kubectl"
	"operators.kloudlite.io/pkg/logging"
	rApi "operators.kloudlite.io/pkg/operator"
	stepResult "operators.kloudlite.io/pkg/operator/step-result"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	harborCli  *harbor.Client
	logger     logging.Logger
	Name       string
	yamlClient *kubectl.YAMLClient
	restConfig *rest.Config
	Env        *env.Env
}

func (r *Reconciler) GetName() string {
	return r.Name
}

func newHelmClient(config *rest.Config, namespace string) (helmclient.Client, error) {
	return helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace: namespace,
		},
		RestConfig: config,
	})
}

func areHelmValuesEqual(releaseValues map[string]any, templateValues []byte) bool {
	b, err := json.Marshal(releaseValues)
	if err != nil {
		return false
	}

	tv, err := yaml.YAMLToJSON(templateValues)
	if err != nil {
		return false
	}

	if len(b) != len(tv) || bytes.Compare(b, tv) != 0 {
		return false
	}
	return true
}

const (
	NamespacesReady    string = "namespaces-ready"
	SvcAccountsReady   string = "svc-accounts-ready"
	LokiReady          string = "loki-ready"
	GrafanaReady       string = "grafana-ready"
	PrometheusReady    string = "prometheus-ready"
	CertManagerReady   string = "cert-manager-ready"
	CertIssuerReady    string = "cert-issuer-ready"
	IngressReady       string = "ingress-ready"
	OperatorCRDsReady  string = "operator-crds-ready"
	MsvcAndMresReady   string = "msvc-and-mres-ready"
	OperatorsEnvReady  string = "operator-env-ready"
	KloudliteAPIsReady string = "kloudlite-apis-ready"
	KloudliteWebReady  string = "kloudite-web-ready"
	DefaultsPatched    string = "defaults-patched"
)

var (
	namespacesList = []string{lc.NsCore, lc.NsRedpanda, lc.NsCertManager, lc.NsMonitoring, lc.NsIngress, lc.NsOperators}
)

type githubRelease struct {
	Assets []struct {
		Id   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"assets"`
}

// +kubebuilder:rbac:groups=cluster-setup.kloudlite.io,resources=primaryclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster-setup.kloudlite.io,resources=primaryclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster-setup.kloudlite.io,resources=primaryclusters/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(context.WithValue(ctx, "logger", r.logger), r.Client, request.NamespacedName, &v1.PrimaryCluster{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}
		return ctrl.Result{}, nil
	}

	req.Logger.Infof("NEW RECONCILATION")
	defer func() {
		req.Logger.Infof("RECONCILATION COMPLETE (isReady=%v)", req.Object.Status.IsReady)
	}()

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.RestartIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	// TODO: initialize all checks here
	if step := req.EnsureChecks(NamespacesReady, DefaultsPatched, SvcAccountsReady, LokiReady, GrafanaReady, PrometheusReady, CertManagerReady, CertIssuerReady, IngressReady, OperatorCRDsReady, MsvcAndMresReady, OperatorsEnvReady, KloudliteAPIsReady, KloudliteWebReady); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.ForegroundFinalizer, constants.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.patchDefaults(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureNamespaces(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureSvcAccounts(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureLoki(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensurePrometheus(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureGrafana(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureCertManager(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	//if step := r.ensureCertIssuer(req); !step.ShouldProceed() {
	//	return step.ReconcilerResponse()
	//}

	if step := r.ensureIngressNginx(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureOperatorCRDs(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureOperators(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureMsvcAndMres(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureKloudliteApis(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	req.Object.Status.LastReconcileTime = metav1.Time{Time: time.Now()}
	return ctrl.Result{RequeueAfter: r.Env.ReconcilePeriod}, r.Status().Update(ctx, req.Object)
}

func (r *Reconciler) patchDefaults(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(DefaultsPatched)
	defer req.LogPostCheck(DefaultsPatched)

	if obj.Spec.SharedConstants == nil {
		sharedC := v1.SharedConstants{
			// mongo
			MongoSvcName:  "mongo-svc",
			AuthDbName:    "auth-db",
			ConsoleDbName: "console-db",
			CiDbName:      "ci-db",
			DnsDbName:     "dns-db",
			FinanceDbName: "finance-db",
			IamDbName:     "iam-db",
			CommsDbName:   "comms-db",

			// redis
			RedisSvcName:     "redis-svc",
			AuthRedisName:    "auth-redis",
			ConsoleRedisName: "console-redis",
			CiRedisName:      "ci-redis",
			DnsRedisName:     "dns-redis",
			IamRedisName:     "iam-redis",
			SocketRedisName:  "socket-redis",

			// Apps api
			AppAuthApi:       "auth-api",
			AppConsoleApi:    "console-api",
			AppCiApi:         "ci-api",
			AppFinanceApi:    "finance-api",
			AppCommsApi:      "comms-api",
			AppDnsApi:        "dns-api",
			AppIAMApi:        "iam-api",
			AppJsEvalApi:     "js-eval-api",
			AppGqlGatewayApi: "gateway",
			AppWebhooksApi:   "webhooks",

			// Apps web
			AppAuthWeb:     "auth-web",
			AppAccountsWeb: "accounts-web",
			AppConsoleWeb:  "console-web",
			AppSocketWeb:   "socket-web",

			CookieDomain: obj.Spec.Domain,

			// Secrets
			OAuthSecretName: "oauth-secrets",

			// Routers
			AuthWebDomain: fmt.Sprintf("auth.%s", obj.Spec.Domain),
		}

		if *obj.Spec.SharedConstants != sharedC {
			obj.Spec.SharedConstants = &sharedC
			if err := r.Update(ctx, obj); err != nil {
				return req.CheckFailed(DefaultsPatched, check, err.Error())
			}
			return req.Done().RequeueAfter(1 * time.Second)
		}
	}

	check.Status = true
	if check != checks[DefaultsPatched] {
		checks[DefaultsPatched] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) finalize(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	return req.Finalize()
}

func (r *Reconciler) ensureNamespaces(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(NamespacesReady)
	defer req.LogPostCheck(NamespacesReady)

	for i := range namespacesList {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:            namespacesList[i],
			OwnerReferences: []metav1.OwnerReference{fn.AsOwner(obj, true)},
		}}
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
			if ns.Labels == nil {
				ns.Labels = make(map[string]string, 1)
			}
			ns.Labels["kloudlite.io/cluster-installation"] = obj.Name
			return nil
		}); err != nil {
			return req.CheckFailed(NamespacesReady, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[NamespacesReady] {
		checks[NamespacesReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensureSvcAccounts(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(SvcAccountsReady)
	defer req.LogPostCheck(SvcAccountsReady)

	for _, ps := range obj.Spec.ImgPullSecrets {
		pullScrt, err := rApi.Get(ctx, r.Client, fn.NN(ps.Namespace, ps.Name), &corev1.Secret{})
		if err != nil {
			return req.CheckFailed(SvcAccountsReady, check, err.Error()).Err(nil)
		}

		for _, ns := range namespacesList {
			newPullSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: ps.Name, Namespace: ns}, Type: "kubernetes.io/dockerconfigjson"}
			if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, newPullSecret, func() error {
				newPullSecret.Data = pullScrt.Data
				return nil
			}); err != nil {
				return req.CheckFailed(SvcAccountsReady, check, err.Error()).Err(nil)
			}

			normalSvcAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: lc.DefaultSvcAccount, Namespace: ns}}
			if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, normalSvcAccount, func() error {
				normalSvcAccount.OwnerReferences = []metav1.OwnerReference{fn.AsOwner(obj, true)}
				if !fn.ContainsAll(normalSvcAccount.ImagePullSecrets, []corev1.LocalObjectReference{{Name: newPullSecret.Name}}) {
					normalSvcAccount.ImagePullSecrets = append(normalSvcAccount.ImagePullSecrets, corev1.LocalObjectReference{Name: newPullSecret.Name})
				}
				return nil
			}); err != nil {
				return req.CheckFailed(SvcAccountsReady, check, err.Error()).Err(nil)
			}

			clusterSvcAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: lc.ClusterSvcAccount, Namespace: ns}}

			if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, clusterSvcAccount, func() error {
				clusterSvcAccount.OwnerReferences = []metav1.OwnerReference{fn.AsOwner(obj, true)}
				if !fn.ContainsAll(clusterSvcAccount.ImagePullSecrets, []corev1.LocalObjectReference{{Name: newPullSecret.Name}}) {
					clusterSvcAccount.ImagePullSecrets = append(clusterSvcAccount.ImagePullSecrets, corev1.LocalObjectReference{Name: newPullSecret.Name})
				}
				return nil
			}); err != nil {
				return req.CheckFailed(SvcAccountsReady, check, err.Error()).Err(nil)
			}
		}
	}

	// cluster role binding
	clusterRb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: lc.ClusterSvcAccount + "-rb",
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, clusterRb, func() error {
		subjects := make(map[string]bool, len(clusterRb.Subjects))
		for i := range clusterRb.Subjects {
			subjects[clusterRb.Subjects[i].Namespace] = true
		}

		for i := range namespacesList {
			if !subjects[namespacesList[i]] {
				clusterRb.Subjects = append(clusterRb.Subjects, rbacv1.Subject{
					Kind:      "ServiceAccount",
					APIGroup:  "",
					Name:      lc.ClusterSvcAccount,
					Namespace: namespacesList[i],
				})
			}

			clusterRb.RoleRef = rbacv1.RoleRef{
				APIGroup: "",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			}
		}
		return nil
	}); err != nil {
		return req.CheckFailed(SvcAccountsReady, check, err.Error()).Err(nil)
	}

	check.Status = true
	if check != checks[SvcAccountsReady] {
		checks[SvcAccountsReady] = check
		return req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) ensureLoki(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(LokiReady)
	defer req.LogPostCheck(LokiReady)

	const releaseName = "loki"

	helmCli, err := newHelmClient(r.restConfig, lc.NsMonitoring)
	if err != nil {
		return req.CheckFailed(LokiReady, check, err.Error()).Err(nil)
	}

	helmValues, err := helmCli.GetReleaseValues(releaseName, false)
	if err != nil {
		req.Logger.Infof("helm release (%s) not found, will be creating it", releaseName)
	}

	b, err := templates.Parse(templates.LokiValues, map[string]any{"loki-values": obj.Spec.LokiValues})
	if err != nil {
		return req.CheckFailed(LokiReady, check, err.Error()).Err(nil)
	}

	if !areHelmValuesEqual(helmValues, b) {
		if err := helmCli.AddOrUpdateChartRepo(repo.Entry{
			Name: "grafana",
			URL:  "https://grafana.github.io/helm-charts",
		}); err != nil {
			return req.CheckFailed(LokiReady, check, err.Error()).Err(nil)
		}

		if _, err := helmCli.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
			ReleaseName: releaseName,
			ChartName:   "grafana/loki-stack",
			Namespace:   lc.NsMonitoring,
			ValuesYaml:  string(b),
		}, &helmclient.GenericHelmOptions{}); err != nil {
			return req.CheckFailed(LokiReady, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[LokiReady] {
		checks[LokiReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensurePrometheus(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(PrometheusReady)
	defer req.LogPostCheck(PrometheusReady)

	const releaseName = "prometheus"

	helmCli, err := newHelmClient(r.restConfig, lc.NsMonitoring)
	if err != nil {
		return req.CheckFailed(PrometheusReady, check, err.Error()).Err(nil)
	}

	helmValues, err := helmCli.GetReleaseValues(releaseName, false)
	if err != nil {
		req.Logger.Infof("helm release (%s) not found, will be creating it", releaseName)
	}

	b, err := templates.Parse(templates.PrometheusValues, map[string]any{"prometheus-values": obj.Spec.PrometheusValues, "name": releaseName})
	if err != nil {
		return req.CheckFailed(PrometheusReady, check, err.Error()).Err(nil)
	}

	if !areHelmValuesEqual(helmValues, b) {
		if err := helmCli.AddOrUpdateChartRepo(repo.Entry{
			Name: "bitnami",
			URL:  "https://charts.bitnami.com/bitnami",
		}); err != nil {
			return req.CheckFailed(PrometheusReady, check, err.Error()).Err(nil)
		}

		if _, err := helmCli.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
			ReleaseName: releaseName,
			ChartName:   "bitnami/kube-prometheus",
			Namespace:   lc.NsMonitoring,
			ValuesYaml:  string(b),
		}, &helmclient.GenericHelmOptions{}); err != nil {
			req.Logger.Error(err)
			return req.CheckFailed(PrometheusReady, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[PrometheusReady] {
		checks[PrometheusReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensureGrafana(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(GrafanaReady)
	defer req.LogPostCheck(GrafanaReady)

	const releaseName = "grafana"

	helmCli, err := newHelmClient(r.restConfig, lc.NsMonitoring)
	if err != nil {
		return req.CheckFailed(GrafanaReady, check, err.Error()).Err(nil)
	}

	helmValues, err := helmCli.GetReleaseValues(releaseName, false)
	if err != nil {
		req.Logger.Infof("helm release (%s) not found, will be creating it", releaseName)
	}

	b, err := templates.Parse(templates.PrometheusValues, map[string]any{"prometheus-values": obj.Spec.PrometheusValues, "name": releaseName})
	if err != nil {
		return req.CheckFailed(GrafanaReady, check, err.Error()).Err(nil)
	}

	if !areHelmValuesEqual(helmValues, b) {
		if err := helmCli.AddOrUpdateChartRepo(repo.Entry{
			Name: "bitnami",
			URL:  "https://charts.bitnami.com/bitnami",
		}); err != nil {
			return req.CheckFailed(GrafanaReady, check, err.Error()).Err(nil)
		}

		if _, err := helmCli.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
			ReleaseName: releaseName,
			ChartName:   "bitnami/grafana",
			Namespace:   lc.NsMonitoring,
			ValuesYaml:  string(b),
		}, &helmclient.GenericHelmOptions{}); err != nil {
			return nil
		}
	}

	check.Status = true
	if check != checks[GrafanaReady] {
		checks[GrafanaReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensureCertManager(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(CertManagerReady)
	defer req.LogPostCheck(CertManagerReady)

	const releaseName = "cert-manager"

	helmCli, err := newHelmClient(r.restConfig, lc.NsCertManager)
	if err != nil {
		return req.CheckFailed(CertManagerReady, check, err.Error()).Err(nil)
	}

	helmValues, err := helmCli.GetReleaseValues(releaseName, false)
	if err != nil {
		req.Logger.Infof("helm release (%s) not found, will be creating it", releaseName)
	}

	b, err := templates.Parse(templates.CertManagerValues, map[string]any{"cert-manager-values": obj.Spec.CertManagerValues})
	if err != nil {
		return req.CheckFailed(CertManagerReady, check, err.Error()).Err(nil)
	}

	if !areHelmValuesEqual(helmValues, b) {
		if err := helmCli.AddOrUpdateChartRepo(repo.Entry{
			Name: "jetstack",
			URL:  "https://charts.jetstack.io",
		}); err != nil {
			return req.CheckFailed(CertManagerReady, check, err.Error()).Err(nil)
		}

		if _, err := helmCli.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
			ReleaseName: releaseName,
			ChartName:   "jetstack/cert-manager",
			Namespace:   lc.NsCertManager,
			ValuesYaml:  string(b),
		}, &helmclient.GenericHelmOptions{}); err != nil {
			return req.CheckFailed(CertManagerReady, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[CertManagerReady] {
		checks[CertManagerReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensureIngressNginx(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(IngressReady)
	defer req.LogPostCheck(IngressReady)

	const releaseName = "ingress-nginx"

	helmCli, err := newHelmClient(r.restConfig, lc.NsIngress)
	if err != nil {
		return req.CheckFailed(IngressReady, check, err.Error()).Err(nil)
	}

	helmValues, err := helmCli.GetReleaseValues(releaseName, false)
	if err != nil {
		req.Logger.Infof("helm release (%s) not found, will be creating it", releaseName)
	}

	b, err := templates.Parse(templates.IngressNginxValues, map[string]any{"ingress-values": obj.Spec.IngressValues})
	if err != nil {
		return req.CheckFailed(IngressReady, check, err.Error()).Err(nil)
	}

	if !areHelmValuesEqual(helmValues, b) {
		if err := helmCli.AddOrUpdateChartRepo(repo.Entry{
			Name: "ingress-nginx",
			URL:  "https://kubernetes.github.io/ingress-nginx",
		}); err != nil {
			return req.CheckFailed(IngressReady, check, err.Error()).Err(nil)
		}

		if _, err := helmCli.InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
			ReleaseName: releaseName,
			ChartName:   "ingress-nginx/ingress-nginx",
			Namespace:   lc.NsIngress,
			ValuesYaml:  string(b),
		}, &helmclient.GenericHelmOptions{}); err != nil {
			return req.CheckFailed(IngressReady, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[IngressReady] {
		checks[IngressReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensureCertIssuer(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(CertIssuerReady)
	defer req.LogPostCheck(CertIssuerReady)

	clusterIssuer, err := rApi.Get(ctx, r.Client, fn.NN("", obj.Spec.CertManagerValues.ClusterIssuer.Name), &certmanagerv1.ClusterIssuer{})
	if err != nil {
		req.Logger.Infof("cluster issuer (%s) not found, will be creating it", obj.Spec.CertManagerValues.ClusterIssuer.Name)
		clusterIssuer = nil
	}

	if clusterIssuer == nil || check.Generation > checks[CertIssuerReady].Generation {
		b, err := templates.Parse(templates.CertIssuer, map[string]any{"cluster-issuer": obj.Spec.CertManagerValues.ClusterIssuer})
		if err != nil {
			return req.CheckFailed(CertIssuerReady, check, err.Error()).Err(nil)
		}

		if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
			return req.CheckFailed(CertIssuerReady, check, err.Error()).Err(nil)
		}
	}

	// creating wildcard certificates for specified cloudflare domains
	wcert := &certmanagerv1.Certificate{ObjectMeta: metav1.ObjectMeta{
		Name:      obj.Spec.CertManagerValues.ClusterIssuer.Name,
		Namespace: lc.NsCertManager,
	}}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, wcert, func() error {
		wcert.Spec.DNSNames = obj.Spec.CertManagerValues.ClusterIssuer.Cloudflare.DnsNames
		wcert.Spec.SecretName = obj.Spec.CertManagerValues.ClusterIssuer.Name + "-tls"
		wcert.Spec.IssuerRef = certmanagerMetav1.ObjectReference{
			Kind: "ClusterIssuer",
			Name: obj.Spec.CertManagerValues.ClusterIssuer.Name,
		}
		return nil
	}); err != nil {
		return req.CheckFailed(CertIssuerReady, check, err.Error()).Err(nil)
	}

	check.Status = true
	if check != checks[CertIssuerReady] {
		checks[CertIssuerReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensureOperatorCRDs(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(OperatorCRDsReady)
	defer req.LogPostCheck(OperatorCRDsReady)

	if check.Generation > checks[OperatorCRDsReady].Generation || !checks[OperatorCRDsReady].Status {

		for _, ghSource := range obj.Spec.Operators.Manifests {
			artifactsMap := make(map[string]bool, len(ghSource.Artifacts))
			for _, a := range ghSource.Artifacts {
				artifactsMap[a] = true
			}

			artifactIds := make([]int64, 0, len(artifactsMap))

			ghTokenScrt, err := rApi.Get(ctx, r.Client, fn.NN(ghSource.TokenSecret.Namespace, ghSource.TokenSecret.Name), &corev1.Secret{})
			if err != nil {
				return req.CheckFailed(OperatorCRDsReady, check, err.Error()).Err(nil)
			}

			ghToken := string(ghTokenScrt.Data[ghSource.TokenSecret.Key])

			httpReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", ghSource.Repo, ghSource.Tag), nil)
			httpReq.Header.Set("Authorization", fmt.Sprintf("token %s", ghToken))
			if err != nil {
				return req.CheckFailed(OperatorCRDsReady, check, err.Error()).Err(nil)
			}

			ghRelease, _, err := kHttp.Get[githubRelease](httpReq)

			if ghRelease == nil {
				return req.CheckFailed(OperatorCRDsReady, check, "github release not found").Err(nil)
			}

			for i := range ghRelease.Assets {
				if artifactsMap[ghRelease.Assets[i].Name] {
					artifactIds = append(artifactIds, ghRelease.Assets[i].Id)
				}
			}

			for i := range artifactIds {
				dReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/releases/assets/%d", ghSource.Repo, artifactIds[i]), nil)
				dReq.Header.Set("Authorization", fmt.Sprintf("token %s", ghToken))
				if err != nil {
					return req.CheckFailed(OperatorCRDsReady, check, err.Error())
				}
				dReq.Header.Set("Accept", "application/octet-stream")
				resp, err := http.DefaultClient.Do(dReq)
				if err != nil {
					return req.CheckFailed(OperatorCRDsReady, check, err.Error())
				}
				output, err := io.ReadAll(resp.Body)
				if err != nil {
					return req.CheckFailed(OperatorCRDsReady, check, err.Error())
				}

				b, err := templates.ParseBytes(output, map[string]any{
					"Namespace":       lc.NsOperators,
					"SvcAccountName":  lc.ClusterSvcAccount,
					"ImagePullPolicy": "Always",
					"EnvName":         "production",
					"ImageTag":        "v1.0.4",
				})

				if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
					return req.CheckFailed(OperatorCRDsReady, check, err.Error())
				}
			}
		}
	}

	check.Status = true
	if check != checks[OperatorCRDsReady] {
		checks[OperatorCRDsReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensureOperators(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	// operator, helm-operator and internal-operator
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(OperatorsEnvReady)
	defer req.LogPostCheck(OperatorsEnvReady)

	b, err := templates.Parse(templates.InternalOperatorEnv, map[string]any{
		"namespace":       "kl-core",
		"cluster-id":      obj.Spec.ClusterID,
		"wildcard-domain": fmt.Sprintf("*.%s", obj.Spec.Domain),
	})
	if err != nil {
		return req.CheckFailed(OperatorsEnvReady, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
		return req.CheckFailed(OperatorsEnvReady, check, err.Error()).Err(nil)
	}

	cfSecret, err := rApi.Get(ctx, r.Client, fn.NN(obj.Spec.CloudflareCreds.SecretKeyRef.Namespace, obj.Spec.CloudflareCreds.SecretKeyRef.Name), &corev1.Secret{})
	if err != nil {
		return req.CheckFailed(OperatorsEnvReady, check, err.Error()).Err(nil)
	}

	cfCertManagerScrt := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: obj.Spec.CloudflareCreds.SecretKeyRef.Name, Namespace: lc.NsCertManager}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, cfCertManagerScrt, func() error {
		if cfCertManagerScrt.Type != "" {
			cfCertManagerScrt.Type = cfSecret.Type
		}
		cfCertManagerScrt.Data = cfSecret.Data
		return nil
	}); err != nil {
		return req.CheckFailed(OperatorsEnvReady, check, err.Error()).Err(nil)
	}

	b, err = templates.Parse(templates.RouterOperatorEnv, map[string]any{
		"namespace":               lc.NsOperators,
		"cluster-wildcard-domain": fmt.Sprintf("*.%v", obj.Spec.Domain),
		"cloudflare-email":        obj.Spec.CloudflareCreds.Email,
		"cloudflare-secret-name":  cfCertManagerScrt.Name,
	})
	if err != nil {
		return req.CheckFailed(OperatorsEnvReady, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
		return req.CheckFailed(OperatorsEnvReady, check, err.Error())
	}

	b, err = templates.Parse(templates.InternalOperatorEnv, map[string]any{
		"namespace":       lc.NsOperators,
		"cluster-id":      obj.Spec.ClusterID,
		"wildcard-domain": obj.Spec.Domain,
	})
	if err != nil {
		return req.CheckFailed(OperatorsEnvReady, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
		return req.CheckFailed(OperatorsEnvReady, check, err.Error())
	}

	check.Status = true
	if check != checks[OperatorsEnvReady] {
		checks[OperatorsEnvReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) ensureMsvcAndMres(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(MsvcAndMresReady)
	defer req.LogPostCheck(MsvcAndMresReady)

	b, err := templates.Parse(templates.MongoMsvcAndMres, map[string]any{
		"namespace":           lc.NsCore,
		"local-storage-class": obj.Spec.StorageClass,
		"region":              "master",
	})
	if err != nil {
		return req.CheckFailed(MsvcAndMresReady, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
		return req.CheckFailed(MsvcAndMresReady, check, err.Error()).Err(nil)
	}

	b, err = templates.Parse(templates.RedisMsvcAndMres, map[string]any{
		"namespace":           lc.NsCore,
		"local-storage-class": obj.Spec.StorageClass,
		"region":              "master",
	})
	if err != nil {
		return req.CheckFailed(MsvcAndMresReady, check, err.Error()).Err(nil)
	}

	if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
		return req.CheckFailed(MsvcAndMresReady, check, err.Error()).Err(nil)
	}

	check.Status = true
	if check != checks[MsvcAndMresReady] {
		checks[MsvcAndMresReady] = check
		return req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) ensureKloudliteApis(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(KloudliteAPIsReady)
	defer req.LogPostCheck(KloudliteAPIsReady)

	// patch oauth-secrets
	oauthSecrets := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: obj.Spec.OAuthCreds.Name, Namespace: obj.Spec.OAuthCreds.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, oauthSecrets, func() error {
		if oauthSecrets != nil {
			oauthSecrets.Data["GITHUB_CALLBACK_URL"] = []byte(strings.Replace(string(oauthSecrets.Data["GITHUB_CALLBACK_URL"]), "AUTH_WEB_DOMAIN", fmt.Sprintf("auth.%s", obj.Spec.Domain), 1))
			oauthSecrets.Data["GITLAB_CALLBACK_URL"] = []byte(strings.Replace(string(oauthSecrets.Data["GITLAB_CALLBACK_URL"]), "AUTH_WEB_DOMAIN", fmt.Sprintf("auth.%s", obj.Spec.Domain), 1))
			oauthSecrets.Data["GOOGLE_CALLBACK_URL"] = []byte(strings.Replace(string(oauthSecrets.Data["GOOGLE_CALLBACK_URL"]), "AUTH_WEB_DOMAIN", fmt.Sprintf("auth.%s", obj.Spec.Domain), 1))
		}
		return nil
	}); err != nil {
		return req.CheckFailed(KloudliteAPIsReady, check, err.Error()).Err(nil)
	}

	authApi, err := rApi.Get(ctx, r.Client, fn.NN(lc.NsCore, obj.Spec.SharedConstants.AppAuthApi), &crdsv1.App{})
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return req.CheckFailed(KloudliteAPIsReady, check, err.Error()).Err(nil)
		}
	}

	if authApi != nil || check.Generation > checks[KloudliteWebReady].Generation {
		b, err := templates.Parse(templates.AuthApi, map[string]any{
			"namespace":        lc.NsCore,
			"image-auth-api":   fmt.Sprintf("registry.kloudlite.io/kloudlite/production/auth:v1.0.4"),
			"shared-constants": obj.Spec.SharedConstants,
		})
		if err != nil {
			return req.CheckFailed(KloudliteAPIsReady, check, err.Error()).Err(nil)
		}

		if err := r.yamlClient.ApplyYAML(ctx, b); err != nil {
			return req.CheckFailed(KloudliteAPIsReady, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[KloudliteAPIsReady] {
		checks[KloudliteAPIsReady] = check
		return req.UpdateStatus()
	}
	return req.Next()

}

func (r *Reconciler) ensureKloudliteWebs(req *rApi.Request[*v1.PrimaryCluster]) stepResult.Result {
	return req.Next()
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.logger = logger.WithName(r.Name)
	r.yamlClient = kubectl.NewYAMLClientOrDie(mgr.GetConfig())
	r.restConfig = mgr.GetConfig()

	builder := ctrl.NewControllerManagedBy(mgr).For(&v1.PrimaryCluster{})
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}
