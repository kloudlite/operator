package helm_controller

import (
	"context"
	"embed"
	"fmt"
	"path"
	"time"

	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	"github.com/kloudlite/operator/operators/helm-charts/internal/env"
	"github.com/kloudlite/operator/pkg/constants"
	fn "github.com/kloudlite/operator/pkg/functions"
	"github.com/kloudlite/operator/pkg/helm"
	job_manager "github.com/kloudlite/operator/pkg/job-manager"
	"github.com/kloudlite/operator/pkg/kubectl"
	"github.com/kloudlite/operator/pkg/logging"
	rApi "github.com/kloudlite/operator/pkg/operator"
	stepResult "github.com/kloudlite/operator/pkg/operator/step-result"
	"github.com/kloudlite/operator/pkg/templates"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Env        *env.Env
	logger     logging.Logger
	Name       string
	yamlClient kubectl.YAMLClient
	helmClient helm.Client

	templateInstallOrUpgradeJob []byte
	templateUninstallJob        []byte
}

func (r *Reconciler) GetName() string {
	return r.Name
}

var (
	ChartReleaseInstalledOrUpgraded string = "chart-release-installed-or-upgraded"
	ChartRepoAdded                  string = "chart-repo-added"
)

// checks
const (
	installOrUpgradeJob       string = "install-or-upgrade-job"
	uninstallJob              string = "uninstall-job"
	checkJobStatus            string = "check-job-status"
	waitForPrevJobsToComplete string = "wait-for-prev-jobs-to-complete"
)

func getJobName(resName string) string {
	return fmt.Sprintf("helm-job-%s", resName)
}

//go:embed templates/*
var templatesDir embed.FS

// +kubebuilder:rbac:groups=helm.kloudlite.io,resources=helmcharts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.kloudlite.io,resources=helmcharts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.kloudlite.io,resources=helmcharts/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(context.WithValue(ctx, "logger", r.logger), r.Client, request.NamespacedName, &crdsv1.HelmChart{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	req.LogPreReconcile()
	defer req.LogPostReconcile()

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}
		return ctrl.Result{}, nil
	}

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(constants.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	// if step := r.addChartRepo(req); !step.ShouldProceed() {
	// 	return step.ReconcilerResponse()
	// }
	//
	// if step := r.installOrUpgradeChartRelease(req); !step.ShouldProceed() {
	// 	return step.ReconcilerResponse()
	// }

	if step := r.startInstallJob(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	// if step := r.checkInstallJobStatus(req); !step.ShouldProceed() {
	// 	return step.ReconcilerResponse()
	// }

	req.Object.Status.IsReady = true
	if step := req.UpdateStatus(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}
	return ctrl.Result{RequeueAfter: r.Env.ReconcilePeriod}, nil
}

func (r *Reconciler) finalize(req *rApi.Request[*crdsv1.HelmChart]) stepResult.Result {
	// ctx, obj := req.Context(), req.Object
	_, obj := req.Context(), req.Object

	checkName := "finalize"

	req.LogPreCheck(checkName)
	defer req.LogPostCheck(checkName)

	// check := rApi.Check{Generation: obj.Generation}

	// r.helmClient.UninstallRelease(ctx, obj.Namespace, obj.Name)

	if step := r.startUninstallJob(req); !step.ShouldProceed() {
		return step
	}

	// jobNs := "kl-init-operators"
	//
	// job := &batchv1.Job{}
	// if err := r.Get(ctx, fn.NN(jobNs, getJobName(obj.Name)), job); err != nil {
	// 	return req.CheckFailed(checkJobStatus, check, "failed to find corresponding job")
	// }
	//
	// p, err := GetLatestPodForJob(ctx, r.Client, jobNs, getJobName(obj.Name))
	// if err != nil {
	// 	return req.CheckFailed(checkJobStatus, check, "failed to find latest pod for job")
	// }
	//
	// js := checkIfJobHasFinished(ctx, job)
	// if js != JobStatusSuccess {
	// 	check.Status = false
	// 	check.Message = GetTerminationLogFromPod(ctx, p)
	//
	// 	if check != obj.Status.Checks[checkJobStatus] {
	// 		obj.Status.Checks[checkJobStatus] = check
	// 		req.UpdateStatus()
	// 	}
	// 	return req.CheckFailed(checkName, check, "waiting for job to succeed")
	// }
	//
	// if err := r.helmClient.UninstallRelease(ctx, obj.Namespace, obj.Name); err != nil {
	// 	return req.CheckFailed(checkName, check, err.Error())
	// }

	for i := range obj.Status.Resources {
		res := obj.Status.Resources[i]
		resObj := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": res.APIVersion,
				"kind":       res.Kind,
				"metadata": map[string]any{
					"name":      res.Name,
					"namespace": res.Namespace,
				},
			},
		}
		_ = resObj
		req.Logger.Infof("deleting child resource: apiVersion: %s, kind: %s, %s/%s", res.APIVersion, res.Kind, res.Namespace, res.Name)
		// if err := r.Delete(ctx, &resObj); err != nil {
		// 	if !errors.IsNotFound(err) {
		// 		return req.CheckFailed(checkName, check, err.Error())
		// 	}
		// }
	}

	return req.Finalize()
}

func (r *Reconciler) startInstallJob(req *rApi.Request[*crdsv1.HelmChart]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(installOrUpgradeJob)
	defer req.LogPostCheck(installOrUpgradeJob)

	jobNs := "kl-init-operators"

	b, err := templates.ParseBytes(r.templateInstallOrUpgradeJob, map[string]any{
		"job-name":      getJobName(obj.Name),
		"job-namespace": jobNs,
		"labels": map[string]string{
			"kloudlite.io/chart-install-or-upgrade-job": "true",
		},
		// "owner-refs": []metav1.OwnerReference{fn.AsOwner(obj, true)},

		"repo-url":  obj.Spec.ChartRepo.Url,
		"repo-name": obj.Spec.ChartRepo.Name,

		"chart-name":    obj.Spec.ChartName,
		"chart-version": obj.Spec.ChartVersion,

		"release-name":      obj.Name,
		"release-namespace": obj.Namespace,
		"values-yaml":       obj.Spec.ValuesYaml,
	})
	if err != nil {
		return req.CheckFailed(installOrUpgradeJob, check, err.Error()).Err(nil)
	}

	job := &batchv1.Job{}
	if err := r.Get(ctx, fn.NN(jobNs, getJobName(obj.Name)), job); err != nil {
		job = nil
	}

	jobFound := false

	if job != nil {
		// handle job exists
		if job.Generation == obj.Generation && job.Labels["kloudlite.io/chart-install-or-upgrade-job"] == "true" {
			jobFound = true
			// this is our job
			if !job_manager.HasJobFinished(ctx, r.Client, job) {
				return req.CheckFailed(installOrUpgradeJob, check, "waiting for job to finish execution")
			}
			tlog := job_manager.GetTerminationLog(ctx, r.Client, job.Namespace, job.Name)
			check.Message = tlog
		} else {
			// it is someone else's job, wait for it to complete
			if !job_manager.HasJobFinished(ctx, r.Client, job) {
				return req.CheckFailed(installOrUpgradeJob, check, fmt.Sprintf("waiting for previous jobs to finish execution"))
			}

			if err := job_manager.DeleteJob(ctx, r.Client, job.Namespace, job.Name); err != nil {
				return req.CheckFailed(installOrUpgradeJob, check, err.Error())
			}
		}
	}

	if !jobFound {
		rr, err := r.yamlClient.ApplyYAML(ctx, b)
		if err != nil {
			return req.CheckFailed(installOrUpgradeJob, check, err.Error())
		}

		req.AddToOwnedResources(rr...)
		return req.Done().RequeueAfter(1 * time.Second).Err(fmt.Errorf("waiting for job to be created"))
	}

	check.Status = true
	if obj.Status.Checks == nil {
		obj.Status.Checks = map[string]rApi.Check{}
	}
	if check != obj.Status.Checks[installOrUpgradeJob] {
		obj.Status.Checks[installOrUpgradeJob] = check
		if sr := req.UpdateStatus(); !sr.ShouldProceed() {
			return sr
		}
	}

	return req.Next()
}

func (r *Reconciler) checkIfAnyPrevJob(ctx context.Context, jobNamespace string, jobName string) *batchv1.Job {
	job := &batchv1.Job{}
	if err := r.Get(ctx, fn.NN(jobNamespace, jobName), job); err != nil {
		return nil
	}
	return job
}

func (r *Reconciler) checkInstallJobStatus(req *rApi.Request[*crdsv1.HelmChart]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(checkJobStatus)
	defer req.LogPostCheck(checkJobStatus)

	jobNs := "kl-init-operators"

	job := &batchv1.Job{}
	if err := r.Get(ctx, fn.NN(jobNs, getJobName(obj.Name)), job); err != nil {
		return req.CheckFailed(checkJobStatus, check, "failed to find corresponding job")
	}

	p, err := GetLatestPodForJob(ctx, r.Client, jobNs, getJobName(obj.Name))
	if err != nil {
		return req.CheckFailed(checkJobStatus, check, "failed to find latest pod for job")
	}

	js := checkIfJobHasFinished(ctx, job)
	check.Status = js == JobStatusSuccess
	check.Message = GetTerminationLogFromPod(ctx, p)
	if check != obj.Status.Checks[checkJobStatus] {
		obj.Status.Checks[checkJobStatus] = check
		req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) checkIfJobTaskCompleted(ctx context.Context, obj client.Object, jobNamespace string, jobName string, action string) (bool, error) {
	job := &batchv1.Job{}
	if err := r.Get(ctx, fn.NN(jobNamespace, jobName), job); err != nil {
		return false, err
	}

	if job.Generation != obj.GetGeneration() {
		return false, fmt.Errorf("job is of a different generation")
	}

	switch action {
	case "install":
		return job.Labels["kloudlite.io/chart-install-or-upgrade-job"] == "true", nil
	case "uninstall":
		return job.Labels["kloudlite.io/uninstall"] == "true", nil
	}

	js := checkIfJobHasFinished(ctx, job)
	if js == JobStatusSuccess || js == JobStatusFailed {
		return true, nil
	}
	return false, nil
}

func (r *Reconciler) waitForPrevJobsToComplete(ctx context.Context, req *rApi.Request[*crdsv1.HelmChart], jobNamespace string, jobName string) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(waitForPrevJobsToComplete)
	defer req.LogPostCheck(waitForPrevJobsToComplete)

	job := &batchv1.Job{}
	if err := r.Get(ctx, fn.NN(jobNamespace, jobName), job); err != nil {
		if errors.IsNotFound(err) {
			return req.Next()
		}
		return req.CheckFailed(waitForPrevJobsToComplete, check, err.Error())
	}

	js := checkIfJobHasFinished(ctx, job)
	if js == JobStatusSuccess || js == JobStatusFailed {
		// means, job has finished

		if err := r.Delete(ctx, job); err != nil {
			return req.CheckFailed(waitForPrevJobsToComplete, check, err.Error())
		}
	}

	return req.CheckFailed(waitForPrevJobsToComplete, check, fmt.Sprintf("job %s/%s is still running", jobNamespace, jobName))
}

func (r *Reconciler) startUninstallJob(req *rApi.Request[*crdsv1.HelmChart]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(uninstallJob)
	defer req.LogPostCheck(uninstallJob)

	jobNs := "kl-init-operators"

	_ = []metav1.OwnerReference{fn.AsOwner(obj, true)}

	b, err := templates.ParseBytes(r.templateUninstallJob, map[string]any{
		"job-name":      getJobName(obj.Name),
		"job-namespace": jobNs,
		"labels": map[string]string{
			"kloudlite.io/chart-uninstall-job": "true",
		},
		// "owner-refs": []metav1.OwnerReference{fn.AsOwner(obj, true)},

		"release-name":      obj.Name,
		"release-namespace": obj.Namespace,
	})
	if err != nil {
		return req.CheckFailed(uninstallJob, check, err.Error()).Err(nil)
	}

	job := &batchv1.Job{}
	if err := r.Get(ctx, fn.NN(jobNs, getJobName(obj.Name)), job); err != nil {
		job = nil
	}

	jobFound := false
	if job != nil {
		// job exists
		if job.Generation == obj.Generation && job.Labels["kloudlite.io/chart-uninstall-job"] == "true" {
			jobFound = true
			// this is our job
			if !job_manager.HasJobFinished(ctx, r.Client, job) {
				return req.CheckFailed(uninstallJob, check, "waiting for job to finish execution")
			}
			tlog := job_manager.GetTerminationLog(ctx, r.Client, job.Namespace, job.Name)
			check.Message = tlog
		} else {
			// it is someone else's job, wait for it to complete
			if !job_manager.HasJobFinished(ctx, r.Client, job) {
				return req.CheckFailed(uninstallJob, check, fmt.Sprintf("waiting for previous jobs to finish execution"))
			}
			// deleting that job
			if err := job_manager.DeleteJob(ctx, r.Client, job.Namespace, job.Name); err != nil {
				return req.CheckFailed(uninstallJob, check, err.Error())
			}
		}
	}

	if !jobFound {
		rr, err := r.yamlClient.ApplyYAML(ctx, b)
		if err != nil {
			return req.CheckFailed(uninstallJob, check, err.Error()).Err(nil)
		}

		req.AddToOwnedResources(rr...)
	}

	check.Status = true
	// check.Hash = fn.New(fn.HashMD5(b))
	if check != obj.Status.Checks[uninstallJob] {
		obj.Status.Checks[uninstallJob] = check
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
	r.yamlClient = kubectl.NewYAMLClientOrDie(mgr.GetConfig())
	r.helmClient = helm.NewHelmClientOrDie(mgr.GetConfig(), helm.ClientOptions{
		RepositoryCacheDir:   r.Env.HelmRepositoryCacheDir,
		RepositoryConfigFile: path.Join(r.Env.HelmRepositoryCacheDir, ".helm-repo-config.yml"),
		Logger:               r.logger,
	})

	var err error
	r.templateInstallOrUpgradeJob, err = templatesDir.ReadFile("templates/install-or-upgrade-job.yml.tpl")
	if err != nil {
		return err
	}

	r.templateUninstallJob, err = templatesDir.ReadFile("templates/uninstall-job.yml.tpl")
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).For(&crdsv1.HelmChart{})
	builder.Owns(&batchv1.Job{})
	builder.WithOptions(controller.Options{MaxConcurrentReconciles: r.Env.MaxConcurrentReconciles})
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}
