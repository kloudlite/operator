package acluser

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	redpandaMsvcv1 "operators.kloudlite.io/apis/redpanda.msvc/v1"
	"operators.kloudlite.io/env"
	"operators.kloudlite.io/lib/constants"
	"operators.kloudlite.io/lib/errors"
	fn "operators.kloudlite.io/lib/functions"
	"operators.kloudlite.io/lib/harbor"
	"operators.kloudlite.io/lib/logging"
	rApi "operators.kloudlite.io/lib/operator"
	stepResult "operators.kloudlite.io/lib/operator/step-result"
	"operators.kloudlite.io/lib/redpanda"
	"operators.kloudlite.io/lib/templates"
	"operators.kloudlite.io/operators/msvc.redpanda/internal/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	env       *env.Env
	harborCli *harbor.Client
	logger    logging.Logger
	Name      string
}

func (r *Reconciler) GetName() string {
	return r.Name
}

const (
	AccessCredsReady  string = "access-creds-ready"
	RedpandaUserReady string = "redpanda-user-ready"
)

const (
	KeyAccessCreds string = "access-creds"
)

// +kubebuilder:rbac:groups=redpanda.msvc.kloudlite.io,resources=aclusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redpanda.msvc.kloudlite.io,resources=aclusers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redpanda.msvc.kloudlite.io,resources=aclusers/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(context.WithValue(ctx, "logger", r.logger), r.Client, request.NamespacedName, &redpandaMsvcv1.ACLUser{})
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

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.RestartIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	// TODO: initialize all checks here
	if step := req.EnsureChecks(AccessCredsReady, RedpandaUserReady); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.ForegroundFinalizer, constants.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.reconAccessCreds(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.reconACLUser(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	req.Logger.Infof("RECONCILATION COMPLETE")
	return ctrl.Result{RequeueAfter: r.env.ReconcilePeriod * time.Second}, r.Status().Update(ctx, req.Object)
}

func (r *Reconciler) finalize(req *rApi.Request[*redpandaMsvcv1.ACLUser]) stepResult.Result {
	return req.Finalize()
}

func (r *Reconciler) reconAccessCreds(req *rApi.Request[*redpandaMsvcv1.ACLUser]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	scrt, err := rApi.Get(ctx, r.Client, fn.NN(obj.Namespace, obj.Name), &corev1.Secret{})
	if err != nil {
		req.Logger.Infof("secret (%s) does not exist, will be creating it shortly...", fn.NN(obj.Namespace, obj.Name).String())
	}

	if scrt == nil {
		b, err := templates.Parse(
			templates.Secret, map[string]any{
				"name":      "msvc-" + obj.Name,
				"namespace": obj.Namespace,
				"string-data": types.ACLUserCreds{
					KafkaBrokers: "",
					Username:     obj.Name,
					Password:     fn.CleanerNanoid(40),
				},
			},
		)

		if err != nil {
			return req.CheckFailed(AccessCredsReady, check, err.Error()).Err(nil)
		}

		if err := fn.KubectlApplyExec(ctx, b); err != nil {
			return req.CheckFailed(AccessCredsReady, check, err.Error()).Err(nil)
		}
	}

	check.Status = true
	if check != checks[AccessCredsReady] {
		checks[AccessCredsReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) reconACLUser(req *rApi.Request[*redpandaMsvcv1.ACLUser]) stepResult.Result {
	ctx, obj, checks := req.Context(), req.Object, req.Object.Status.Checks
	check := rApi.Check{Generation: obj.Generation}

	adminSecretNs := func() string {
		if obj.Spec.AdminSecretRef.Namespace != "" {
			return obj.Spec.AdminSecretRef.Namespace
		}
		return obj.Namespace
	}()

	adminScrt, err := rApi.Get(ctx, r.Client, fn.NN(adminSecretNs, obj.Spec.AdminSecretRef.Name), &corev1.Secret{})
	if err != nil {
		return req.CheckFailed(RedpandaUserReady, check, err.Error()).Err(nil)
	}

	adminCreds, err := fn.ParseFromSecret[types.AdminUserCreds](adminScrt)
	if err != nil {
		return req.CheckFailed(RedpandaUserReady, check, err.Error()).Err(nil)
	}

	aclUserCreds, ok := rApi.GetLocal[types.ACLUserCreds](req, KeyAccessCreds)
	if !ok {
		return req.CheckFailed(RedpandaUserReady, check, errors.NotInLocals(KeyAccessCreds).Error()).Err(nil)
	}

	adminCli := redpanda.NewAdminClient(adminCreds.Username, adminCreds.Password, adminCreds.KafkaBrokers, adminCreds.AdminEndpoint)
	if err := adminCli.CreateUser(aclUserCreds.Username, aclUserCreds.Password); err != nil {
		req.Logger.Error(err)
		// return nil
	}

	if err := adminCli.AllowUserOnTopics(aclUserCreds.Username, obj.Spec.Topics...); err != nil {
		req.Logger.Error(err)
		// return nil
	}

	check.Status = true
	if check != checks[RedpandaUserReady] {
		checks[RedpandaUserReady] = check
		return req.UpdateStatus()
	}
	return req.Next()
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, envVars *env.Env, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.logger = logger.WithName(r.Name)
	r.env = envVars

	builder := ctrl.NewControllerManagedBy(mgr).For(&redpandaMsvcv1.ACLUser{})
	return builder.Complete(r)
}
