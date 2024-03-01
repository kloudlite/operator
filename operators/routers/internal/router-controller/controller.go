package router_controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	"github.com/kloudlite/operator/operators/routers/internal/env"
	"github.com/kloudlite/operator/operators/routers/internal/templates"
	"github.com/kloudlite/operator/pkg/constants"
	fn "github.com/kloudlite/operator/pkg/functions"
	"github.com/kloudlite/operator/pkg/kubectl"
	"github.com/kloudlite/operator/pkg/logging"
	rApi "github.com/kloudlite/operator/pkg/operator"
	stepResult "github.com/kloudlite/operator/pkg/operator/step-result"
	apiLabels "k8s.io/apimachinery/pkg/labels"
)

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	logger     logging.Logger
	Name       string
	Env        *env.Env
	yamlClient kubectl.YAMLClient

	templateIngress []byte
}

func (r *Reconciler) GetName() string {
	return r.Name
}

const (
	IngressReady    string = "ingress-ready"
	BasicAuthReady  string = "basic-auth-ready"
	DefaultsPatched string = "patch-defaults"

	Finalizing         string = "finalizing"
	CheckHttpsCerteady string = "https-cert-ready"
)

// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=crds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=crds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=crds/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(rApi.NewReconcilerCtx(ctx, r.logger), r.Client, request.NamespacedName, &crdsv1.Router{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !req.ShouldReconcile() {
		return ctrl.Result{}, nil
	}

	req.PreReconcile()
	defer req.PostReconcile()

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}
		return ctrl.Result{}, nil
	}

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureChecks(IngressReady); !step.ShouldProceed() {
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

	if step := r.ensuringTLSCertSecretsAreReplicated(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.reconBasicAuth(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.ensureIngresses(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.checkHttpsCert(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	return ctrl.Result{}, nil
}

func (r *Reconciler) patchDefaults(req *rApi.Request[*crdsv1.Router]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(DefaultsPatched)
	defer req.LogPostCheck(DefaultsPatched)

	hasUpdated := false

	if obj.Spec.BasicAuth != nil && obj.Spec.BasicAuth.Enabled && obj.Spec.BasicAuth.SecretName == "" {
		hasUpdated = true
		obj.Spec.BasicAuth.SecretName = obj.Name + "-basic-auth"
	}

	if hasUpdated {
		if err := r.Update(ctx, obj); err != nil {
			return req.CheckFailed(DefaultsPatched, check, err.Error()).Err(nil)
		}
		return req.Done().RequeueAfter(1 * time.Second)
	}

	check.Status = true
	if check != checks[DefaultsPatched] {
		checks[DefaultsPatched] = check
		if sr := req.UpdateStatus(); !sr.ShouldProceed() {
			return sr
		}
	}
	return req.Next()
}

func (r *Reconciler) finalize(req *rApi.Request[*crdsv1.Router]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	if controllerutil.RemoveFinalizer(obj, constants.ForegroundFinalizer) {
		if err := r.Update(ctx, obj); err != nil {
			return req.CheckFailed(Finalizing, check, err.Error()).Err(nil)
		}
		return req.Done().RequeueAfter(1 * time.Second)
	}

	var ingList networkingv1.IngressList
	if err := r.List(ctx, &ingList, &client.ListOptions{
		LabelSelector: apiLabels.SelectorFromValidatedSet(obj.GetEnsuredLabels()),
		Namespace:     obj.Namespace,
	}); err != nil {
		return req.CheckFailed(Finalizing, check, err.Error()).Err(nil)
	}

	for i := range ingList.Items {
		if err := r.Delete(ctx, &ingList.Items[i]); err != nil {
			return req.CheckFailed(Finalizing, check, err.Error()).Err(nil)
		}
	}

	if len(ingList.Items) != 0 {
		return req.CheckFailed(Finalizing, check, "waiting for k8s ingress resources to be deleted")
	}

	if controllerutil.RemoveFinalizer(obj, constants.CommonFinalizer) {
		if err := r.Update(ctx, obj); err != nil {
			return req.CheckFailed(Finalizing, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[Finalizing] {
		checks[Finalizing] = check
		if sr := req.UpdateStatus(); !sr.ShouldProceed() {
			return sr
		}
	}
	return req.Next()
}

func (r *Reconciler) isInProjectNamespace(ctx context.Context, obj client.Object) bool {
	n, err := rApi.Get(context.TODO(), r.Client, fn.NN("", obj.GetNamespace()), &corev1.Namespace{})
	if err != nil {
		return false
	}

	if _, ok := n.Labels[constants.EnvironmentNameKey]; ok {
		return false
	}

	if _, ok := n.Labels[constants.ProjectNameKey]; ok {
		return true
	}
	return false
}

func (r *Reconciler) reconBasicAuth(req *rApi.Request[*crdsv1.Router]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(BasicAuthReady)
	defer req.LogPostCheck(BasicAuthReady)

	if obj.Spec.BasicAuth != nil && obj.Spec.BasicAuth.Enabled {
		if len(obj.Spec.BasicAuth.Username) == 0 {
			return req.CheckFailed(BasicAuthReady, check, ".spec.basicAuth.username must be defined when .spec.basicAuth.enabled is set to true").Err(nil)
		}

		basicAuthScrt := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: obj.Spec.BasicAuth.SecretName, Namespace: obj.Namespace}, Type: "Opaque"}
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, basicAuthScrt, func() error {
			basicAuthScrt.SetOwnerReferences([]metav1.OwnerReference{fn.AsOwner(obj, true)})
			if _, ok := basicAuthScrt.Data["password"]; ok {
				return nil
			}

			password := fn.CleanerNanoid(48)
			ePass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			basicAuthScrt.Data = map[string][]byte{
				"auth":     []byte(fmt.Sprintf("%s:%s", obj.Spec.BasicAuth.Username, ePass)),
				"username": []byte(obj.Spec.BasicAuth.Username),
				"password": []byte(password),
			}
			return nil
		}); err != nil {
			return req.CheckFailed(BasicAuthReady, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[BasicAuthReady] {
		checks[BasicAuthReady] = check
		if sr := req.UpdateStatus(); !sr.ShouldProceed() {
			return sr
		}
	}
	return req.Next()
}

func (r *Reconciler) parseAndExtractDomains(req *rApi.Request[*crdsv1.Router]) ([]string, []string, error) {
	ctx, obj := req.Context(), req.Object

	var wildcardPatterns []string

	if obj.Spec.Https != nil && obj.Spec.Https.Enabled {
		issuerName := obj.Spec.Https.ClusterIssuer
		if issuerName == "" {
			issuerName = r.Env.DefaultClusterIssuer
		}

		if issuerName == "" {
			return nil, nil, fmt.Errorf("no cluster issuer found, could not proceed, when https is enabled")
		}

		clusterIssuer, err := rApi.Get(ctx, r.Client, fn.NN("", issuerName), &certmanagerv1.ClusterIssuer{})
		if err != nil {
			if !apiErrors.IsNotFound(err) {
				return nil, nil, err
			}
			clusterIssuer = nil
		}

		if clusterIssuer != nil {
			// var clusterIssuer certmanagerv1.ClusterIssuer
			// b, err := json.Marshal(cIssuer.Object)
			// if err != nil {
			// 	return nil, nil, err
			// }
			// if err := json.Unmarshal(b, &clusterIssuer); err != nil {
			// 	return nil, nil, err
			// }

			for _, solver := range clusterIssuer.Spec.ACME.Solvers {
				if solver.DNS01 != nil {
					wildcardPatterns = solver.Selector.DNSNames
				}
			}
		}
	}

	wildcardDomains, nonWildcardDomains := FilterDomains(wildcardPatterns, obj.Spec.Domains)
	return wildcardDomains, nonWildcardDomains, nil
}

func (r *Reconciler) ensureIngresses(req *rApi.Request[*crdsv1.Router]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(IngressReady)
	defer req.LogPostCheck(IngressReady)

	if len(obj.Spec.Routes) == 0 {
		check.Status = true
		fn.MapSet(&obj.Status.Checks, IngressReady, check)
		if err := r.Status().Update(ctx, obj); err != nil {
			return req.CheckFailed(IngressReady, check, err.Error())
		}
	}

	wcDomains, nonWcDomains, err := r.parseAndExtractDomains(req)
	if err != nil {
		return req.CheckFailed(IngressReady, check, err.Error())
	}

	annotations := make(map[string]string)
	annotations["nginx.ingress.kubernetes.io/preserve-trailing-slash"] = "true"
	annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/$1"
	annotations["nginx.ingress.kubernetes.io/from-to-www-redirect"] = "true"

	if obj.Spec.MaxBodySizeInMB != nil {
		annotations["nginx.ingress.kubernetes.io/proxy-body-size"] = fmt.Sprintf("%vm", *obj.Spec.MaxBodySizeInMB)
	}

	if obj.Spec.Https != nil && obj.Spec.Https.Enabled {
		annotations["nginx.kubernetes.io/ssl-redirect"] = "true"
		annotations["nginx.ingress.kubernetes.io/force-ssl-redirect"] = fmt.Sprintf("%v", obj.Spec.Https.ForceRedirect)

		// cert-manager.io annotations, [read more](https://cert-manager.io/docs/usage/ingress/#supported-annotations)
		annotations["cert-manager.io/cluster-issuer"] = r.Env.DefaultClusterIssuer
		annotations["cert-manager.io/renew-before"] = "168h" // renew certificates a week before expiry
		annotations["acme.cert-manager.io/http01-ingress-class"] = r.Env.DefaultIngressClass
	}

	if obj.Spec.RateLimit != nil && obj.Spec.RateLimit.Enabled {
		if obj.Spec.RateLimit.Rps > 0 {
			annotations["nginx.ingress.kubernetes.io/limit-rps"] = fmt.Sprintf("%v", obj.Spec.RateLimit.Rps)
		}
		if obj.Spec.RateLimit.Rpm > 0 {
			annotations["nginx.ingress.kubernetes.io/limit-rpm"] = fmt.Sprintf("%v", obj.Spec.RateLimit.Rpm)
		}
		if obj.Spec.RateLimit.Connections > 0 {
			annotations["nginx.ingress.kubernetes.io/limit-connections"] = fmt.Sprintf("%v", obj.Spec.RateLimit.Connections)
		}
	}

	if obj.Spec.Cors != nil && obj.Spec.Cors.Enabled {
		annotations["nginx.ingress.kubernetes.io/enable-cors"] = "true"
		annotations["nginx.ingress.kubernetes.io/cors-allow-origin"] = strings.Join(obj.Spec.Cors.Origins, ",")
		annotations["nginx.ingress.kubernetes.io/cors-allow-credentials"] = fmt.Sprintf("%v", obj.Spec.Cors.AllowCredentials)
	}

	if obj.Spec.BackendProtocol != nil {
		annotations["nginx.ingress.kubernetes.io/backend-protocol"] = *obj.Spec.BackendProtocol
	}

	if obj.Spec.BasicAuth != nil && obj.Spec.BasicAuth.Enabled {
		annotations["nginx.ingress.kubernetes.io/auth-type"] = "basic"
		annotations["nginx.ingress.kubernetes.io/auth-secret"] = obj.Spec.BasicAuth.SecretName
		annotations["nginx.ingress.kubernetes.io/auth-realm"] = "route is protected by basic auth"
	}

	// lambdaGroups := map[string][]crdsv1.Route{}
	var appRoutes []crdsv1.Route

	for _, route := range obj.Spec.Routes {
		// if route.Lambda != "" {
		// 	if _, ok := lambdaGroups[route.Lambda]; !ok {
		// 		lambdaGroups[route.Lambda] = []crdsv1.Route{}
		// 	}
		// 	lambdaGroups[route.Lambda] = append(lambdaGroups[route.Lambda], route)
		// 	annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = fmt.Sprintf("%s.%s", route.Lambda, obj.Namespace)
		// }

		if route.App != "" {
			if r.isInProjectNamespace(ctx, obj) {
				route.App = r.Env.WorkspaceRouteSwitcherService
				route.Port = r.Env.WorkspaceRouteSwitcherPort
				appRoutes = append(appRoutes, route)
				continue
			}
			appRoutes = append(appRoutes, route)
		}
	}

	var kubeYamls [][]byte

	// for lName, lRoutes := range lambdaGroups {
	// 	ingName := fmt.Sprintf("r-%s-lambda-%s", obj.Name, lName)
	//
	// 	vals := map[string]any{
	// 		"name":        ingName,
	// 		"namespace":   obj.Namespace,
	// 		"owner-refs":  []metav1.OwnerReference{fn.AsOwner(obj, true)},
	// 		"labels":      obj.GetLabels(),
	// 		"annotations": annotations,
	//
	// 		"domains":          nonWcDomains,
	// 		"wildcard-domains": wcDomains,
	//
	// 		"router-ref":       obj,
	// 		"routes":           lRoutes,
	// 		"virtual-hostname": fmt.Sprintf("%s.%s", lName, obj.Namespace),
	//
	// 		"is-in-project-namespace": r.isInProjectNamespace(ctx, obj),
	// 		"ingress-class":           getIngressClassName(obj),
	// 		"cluster-issuer":          getClusterIssuer(obj),
	// 	}
	//
	// 	b, err := templates.Parse(templates.CoreV1.Ingress, vals)
	// 	if err != nil {
	// 		return req.CheckFailed(IngressReady, check, "cloud not parse ingress template").Err(nil)
	// 	}
	// 	kubeYamls = append(kubeYamls, b)
	// }

	if len(appRoutes) > 0 {
		b, err := templates.ParseBytes(
			r.templateIngress, map[string]any{
				"name":      obj.Name,
				"namespace": obj.Namespace,

				"owner-refs":  []metav1.OwnerReference{fn.AsOwner(obj, true)},
				"labels":      obj.GetLabels(),
				"annotations": annotations,

				"non-wildcard-domains": nonWcDomains,
				"wildcard-domains":     wcDomains,
				"router-domains":       obj.Spec.Domains,

				"ingress-class": func() string {
					if obj.Spec.IngressClass != "" {
						return obj.Spec.IngressClass
					}
					return r.Env.DefaultIngressClass
				}(),
				"cluster-issuer": func() string {
					if obj.Spec.Https != nil && obj.Spec.Https.ClusterIssuer != "" {
						return obj.Spec.Https.ClusterIssuer
					}
					return r.Env.DefaultClusterIssuer
				}(),

				"routes": appRoutes,

				"is-https-enabled": obj.Spec.Https != nil && obj.Spec.Https.Enabled,
			},
		)
		if err != nil {
			return req.CheckFailed(IngressReady, check, err.Error()).Err(nil)
		}

		kubeYamls = append(kubeYamls, b)
	}

	rr, err := r.yamlClient.ApplyYAML(ctx, kubeYamls...)
	if err != nil {
		return req.CheckFailed(IngressReady, check, err.Error()).Err(nil)
	}

	req.AddToOwnedResources(rr...)

	check.Status = true
	if check != checks[IngressReady] {
		checks[IngressReady] = check
		if sr := req.UpdateStatus(); !sr.ShouldProceed() {
			return sr
		}
	}

	return req.Next()
}

func (r *Reconciler) ensuringTLSCertSecretsAreReplicated(req *rApi.Request[*crdsv1.Router]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	checkName := "tls-secrets-are-replicated"

	req.LogPreCheck(checkName)
	defer req.LogPostCheck(checkName)

	fail := func(err error) stepResult.Result {
		return req.CheckFailed(checkName, check, err.Error())
	}

	scrtlist, err := r.yamlClient.Client().CoreV1().Secrets(obj.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("type=%s", corev1.SecretTypeTLS),
	})
	if err != nil {
		return fail(err)
	}

	for _, scrt := range scrtlist.Items {
		if v, ok := scrt.Annotations[constants.ReplicationEnableKey]; !ok || v != constants.ReplicationEnableValueTrue {
			if scrt.Annotations == nil {
				scrt.Annotations = make(map[string]string, 1)
			}
			scrt.Annotations[constants.ReplicationEnableKey] = constants.ReplicationEnableValueTrue
			if err := r.Update(ctx, &scrt); err != nil {
				return fail(err).RequeueAfter(100 * time.Millisecond)
			}
		}
	}

	check.Status = true
	if check != obj.Status.Checks[checkName] {
		fn.MapSet(&obj.Status.Checks, checkName, check)
		if sr := req.UpdateStatus(); !sr.ShouldProceed() {
			return sr
		}
	}

	return req.Next()
}

func (r *Reconciler) checkHttpsCert(req *rApi.Request[*crdsv1.Router]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	checkName := CheckHttpsCerteady

	req.LogPreCheck(checkName)
	defer req.LogPostCheck(checkName)

	if obj.Spec.Https == nil || !obj.Spec.Https.Enabled {
		return req.Done()
	}

	fail := func(err error) stepResult.Result {
		return req.CheckFailed(checkName, check, err.Error())
	}

	var certlist certmanagerv1.CertificateList
	if err := r.List(ctx, &certlist, &client.ListOptions{
		LabelSelector: apiLabels.SelectorFromValidatedSet(map[string]string{
			constants.RouterNameKey: obj.GetName(),
		}),
		Namespace: obj.GetNamespace(),
	}); err != nil {
		return fail(err)
	}

	for _, cert := range certlist.Items {
		errmsg := ""
		for _, cond := range cert.Status.Conditions {
			if cond.Type == certmanagerv1.CertificateConditionReady {
				check.Status = metav1.ConditionStatus(cond.Status) == metav1.ConditionTrue
				if !check.Status {
					errmsg = fmt.Sprintf("HTTPS Certificate is not ready yet. cert-manager says '%s'.", cond.Message)
				}
			}
		}

		if !check.Status {
			for _, cond := range cert.Status.Conditions {
				if cond.Type == certmanagerv1.CertificateConditionIssuing {
					errmsg = fmt.Sprintf("%s It also says '%s'", errmsg, check.Message)
				}
			}

			return fail(fmt.Errorf(errmsg))
		}
	}

	check.Status = true
	if check != obj.Status.Checks[checkName] {
		fn.MapSet(&obj.Status.Checks, checkName, check)
		if sr := req.UpdateStatus(); !sr.ShouldProceed() {
			return sr
		}
	}

	return req.Next()
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.logger = logger.WithName(r.Name)
	r.yamlClient = kubectl.NewYAMLClientOrDie(mgr.GetConfig(), kubectl.YAMLClientOpts{Logger: r.logger})

	var err error
	r.templateIngress, err = templates.ReadIngressTemplate()
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).For(&crdsv1.Router{})
	builder.Owns(&networkingv1.Ingress{})

	builder.Watches(&certmanagerv1.Certificate{}, handler.EnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
		if v, ok := obj.GetLabels()[constants.RouterNameKey]; ok {
			return []reconcile.Request{{fn.NN(obj.GetNamespace(), v)}}
		}

		return nil
	}))

	builder.WithOptions(controller.Options{MaxConcurrentReconciles: r.Env.MaxConcurrentReconciles})
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}