package workspace

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/record"

	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	"github.com/kloudlite/operator/operators/workmachine/internal/env"
	"github.com/kloudlite/operator/operators/workmachine/internal/templates"
	"github.com/kloudlite/operator/pkg/constants"
	"github.com/kloudlite/operator/toolkit/kubectl"
	rApi "github.com/kloudlite/operator/toolkit/reconciler"
	step_result "github.com/kloudlite/operator/toolkit/reconciler/step-result"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Env    *env.Env

	YAMLClient kubectl.YAMLClient
	recorder   record.EventRecorder

	workspaceDeploymentTemplate []byte
	templateWebhook             []byte
}

func (r *Reconciler) GetName() string {
	return "workspace"
}

const (
	CreateDeployment     string = "create-deployment"
	CreateService        string = "create-service"
	createWorkMachineJob string = "create-work-machine-job"
)

// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(ctx, r.Client, request.NamespacedName, &crdsv1.WorkMachine{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
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

	if step := req.EnsureCheckList([]rApi.CheckMeta{{Name: CreateDeployment}}); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.RestartIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.ForegroundFinalizer, constants.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.createWorkMachineCreationJob(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	return ctrl.Result{}, nil
}

func (r *Reconciler) finalize(req *rApi.Request[*crdsv1.WorkMachine]) step_result.Result {
	if step := req.EnsureCheckList([]rApi.CheckMeta{
		{Name: "uninstall workspace"},
	}); !step.ShouldProceed() {
		return step
	}

	check := rApi.NewRunningCheck("uninstall workspace", req)

	if step := req.CleanupOwnedResources(check); !step.ShouldProceed() {
		return step
	}

	return req.Finalize()
}

func (r *Reconciler) createWorkMachineCreationJob(req *rApi.Request[*crdsv1.WorkMachine]) step_result.Result {
	// ctx, obj := req.Context(), req.Object
	check := rApi.NewRunningCheck(createWorkMachineJob, req)

	return check.Completed()
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()

	if r.YAMLClient == nil {
		return fmt.Errorf("yaml client must be set")
	}

	r.recorder = mgr.GetEventRecorderFor(r.GetName())

	var err error
	r.workspaceDeploymentTemplate, err = templates.Read(templates.WorkspaceTemplate)
	if err != nil {
		return err
	}
	builder := ctrl.NewControllerManagedBy(mgr).For(&crdsv1.WorkMachine{})
	builder.WithOptions(controller.Options{MaxConcurrentReconciles: r.Env.MaxConcurrentReconciles})
	builder.Owns(&appsv1.Deployment{})
	builder.Owns(&corev1.Service{})
	builder.Owns(&crdsv1.Router{})
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}
