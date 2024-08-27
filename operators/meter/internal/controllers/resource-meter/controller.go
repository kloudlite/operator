package resourcemeter

import (
	"context"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	apiLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mv1 "github.com/kloudlite/operator/apis/meter/v1"
	"github.com/kloudlite/operator/operators/meter/internal/env"
	"github.com/kloudlite/operator/pkg/constants"
	fn "github.com/kloudlite/operator/pkg/functions"
	"github.com/kloudlite/operator/pkg/kubectl"
	"github.com/kloudlite/operator/pkg/logging"
	rApi "github.com/kloudlite/operator/pkg/operator"
	stepResult "github.com/kloudlite/operator/pkg/operator/step-result"
)

const (
	DEVICE_KEY_PREFIX     = "wg-keys-"
	DEVICE_CONFIG_PREFIX  = "wg-configs-"
	WG_SERVER_NAME_PREFIX = "wg-server-"
	DNS_NAME_PREFIX       = "wg-dns-"
)

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	logger     logging.Logger
	Name       string
	yamlClient kubectl.YAMLClient
	Env        *env.Env
}

func (r *Reconciler) GetName() string {
	return r.Name
}

const (
	DnsConfigReady string = "dns-config-ready"

	KeysAndSecretReady string = "keys-and-secret-ready"
	ServerSvcReady     string = "server-svc-ready"
	ConfigReady        string = "config-ready"
	ServicesSynced     string = "services-synced"
	ServerReady        string = "server-ready"

	RMeterDeleted string = "device-deleted"
)

var (
	RM_CHECKLIST = []rApi.CheckMeta{
		{Name: ServerReady, Title: "Ensuring server is ready"},
	}

	RM_DESTROY_CHECKLIST = []rApi.CheckMeta{
		{Name: RMeterDeleted, Title: "Cleaning up resources"},
	}
)

// +kubebuilder:rbac:groups=meter.kloudlite.io,resources=resourcemeters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meter.kloudlite.io,resources=resourcemeters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=meter.kloudlite.io,resources=resourcemeters/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(rApi.NewReconcilerCtx(ctx, r.logger), r.Client, request.NamespacedName, &mv1.ResourceMeter{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}

		return ctrl.Result{}, nil
	}

	req.PreReconcile()
	defer req.PostReconcile()

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureCheckList(RM_CHECKLIST); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.ForegroundFinalizer, constants.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	return ctrl.Result{}, nil
}

func (r *Reconciler) finalize(req *rApi.Request[*mv1.ResourceMeter]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.NewRunningCheck(RMeterDeleted, req)

	if !slices.Equal(obj.Status.CheckList, RM_DESTROY_CHECKLIST) {
		req.Object.Status.CheckList = RM_DESTROY_CHECKLIST
		if step := req.UpdateStatus(); !step.ShouldProceed() {
			return step
		}
	}

	var services corev1.ServiceList
	if err := r.List(ctx, &services, &client.ListOptions{
		LabelSelector: apiLabels.SelectorFromValidatedSet(map[string]string{constants.WGDeviceNameKey: obj.Name, "kloudlite.io/wg-svc-type": "external"}),
	}); err != nil {
		if !apiErrors.IsNotFound(err) {
			return check.Failed(err)
		}
	} else {
		for _, svc := range services.Items {
			if err := r.Delete(ctx, &svc); err != nil {
				return check.Failed(err)
			}
		}
	}

	return req.Finalize()
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.logger = logger.WithName(r.Name)
	r.yamlClient = kubectl.NewYAMLClientOrDie(mgr.GetConfig(), kubectl.YAMLClientOpts{Logger: r.logger})

	builder := ctrl.NewControllerManagedBy(mgr).For(&mv1.ResourceMeter{})

	watchList := []client.Object{
		&corev1.Secret{},
		&corev1.ConfigMap{},
		&corev1.Service{},
		&appsv1.Deployment{},
	}

	for _, object := range watchList {
		builder.Watches(
			object,
			handler.EnqueueRequestsFromMapFunc(
				func(_ context.Context, obj client.Object) []reconcile.Request {
					if dev, ok := obj.GetLabels()[constants.WGDeviceNameKey]; ok {
						return []reconcile.Request{{NamespacedName: fn.NN(obj.GetNamespace(), dev)}}
					}

					return nil
				}),
		)
	}

	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}
