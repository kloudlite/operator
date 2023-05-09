package byoc_client_watcher

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"google.golang.org/grpc"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	clusterv1 "github.com/kloudlite/operator/apis/clusters/v1"
	"github.com/kloudlite/operator/operators/status-n-billing/internal/env"
	"github.com/kloudlite/operator/operators/status-n-billing/types"
	"github.com/kloudlite/operator/pkg/logging"
	rApi "github.com/kloudlite/operator/pkg/operator"
	stepResult "github.com/kloudlite/operator/pkg/operator/step-result"
)

type Reconciler struct {
	client.Client
	Scheme                    *runtime.Scheme
	Logger                    logging.Logger
	Name                      string
	Env                       *env.Env
	GetGrpcConnection         func() (*grpc.ClientConn, error)
	dispatchBYOCClientUpdates func(ctx context.Context, stu types.StatusUpdate) error
}

func (r *Reconciler) GetName() string {
	return r.Name
}

const (
	DefaultsPatched          string = "defaults-patched"
	KafkaTopicExists         string = "kafka-topic-exists"
	HarborProjectExists      string = "harbor-project-exists"
	StorageClassProcessed    string = "storage-class-processed"
	IngressClassProcessed    string = "ingress-class-processed"
	NodesProcessed           string = "nodes-processed"
	HelmDeploymentsProcessed string = "helm-deployments-processed"
)

const byocClientFinalizer = "kloudlite.io/byoc-client-finalizer"

// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=crds.kloudlite.io,resources=apps/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := rApi.NewRequest(rApi.NewReconcilerCtx(ctx, r.Logger), r.Client, request.NamespacedName, &clusterv1.BYOC{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	obj := req.Object

	req.LogPreReconcile()
	defer req.LogPostReconcile()

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	r.processStorageClasses(req)
	r.processIngressClasses(req)
	r.processNodes(req)
	r.processHelmDeployments(req)

	obj.Status.IsReady = true
	if step := req.UpdateStatus(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	return ctrl.Result{RequeueAfter: r.Env.ReconcilePeriod}, nil
}

func (r *Reconciler) processStorageClasses(req *rApi.Request[*clusterv1.BYOC]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(StorageClassProcessed)
	defer req.LogPostCheck(StorageClassProcessed)

	var scl storagev1.StorageClassList
	if err := r.List(ctx, &scl); err != nil {
		return req.Done().Err(client.IgnoreNotFound(err))
	}

	results := make([]string, len(scl.Items))
	for i := range scl.Items {
		results[i] = scl.Items[i].Name
		if scl.Items[i].Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			results[0], results[i] = results[i], results[0]
		}
	}

	if !reflect.DeepEqual(obj.Spec.StorageClasses, results) {
		obj.Spec.StorageClasses = results
		if err := r.Update(ctx, obj); err != nil {
			return req.CheckFailed(StorageClassProcessed, check, err.Error())
		}
		return req.Done().RequeueAfter(100 * time.Millisecond)
	}

	check.Status = true
	if check != obj.Status.Checks[StorageClassProcessed] {
		obj.Status.Checks[StorageClassProcessed] = check
		req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) processIngressClasses(req *rApi.Request[*clusterv1.BYOC]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(IngressClassProcessed)
	defer req.LogPostCheck(IngressClassProcessed)

	var icl networkingv1.IngressClassList
	if err := r.List(ctx, &icl); err != nil {
		return req.Done().Err(client.IgnoreNotFound(err))
	}

	results := make([]string, len(icl.Items))
	for i := range icl.Items {
		results[i] = icl.Items[i].Name
	}

	if !reflect.DeepEqual(obj.Spec.IngressClasses, results) {
		obj.Spec.IngressClasses = results
		if err := r.Update(ctx, obj); err != nil {
			return req.CheckFailed(IngressClassProcessed, check, err.Error())
		}
		return req.Done().RequeueAfter(100 * time.Millisecond)
	}

	check.Status = true
	if check != obj.Status.Checks[IngressClassProcessed] {
		obj.Status.Checks[IngressClassProcessed] = check
		req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) processNodes(req *rApi.Request[*clusterv1.BYOC]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(NodesProcessed)
	defer req.LogPostCheck(NodesProcessed)

	var nl corev1.NodeList
	if err := r.List(ctx, &nl); err != nil {
		return req.Done().Err(client.IgnoreNotFound(err))
	}

	results := make([]string, len(nl.Items))
	for i := range nl.Items {
		for _, address := range nl.Items[i].Status.Addresses {
			if address.Type == "ExternalIP" {
				results[i] = address.Address
			}
		}
	}

	if !reflect.DeepEqual(obj.Spec.PublicIPs, results) {
		obj.Spec.PublicIPs = results
		if err := r.Update(ctx, obj); err != nil {
			return req.CheckFailed(NodesProcessed, check, err.Error())
		}
		return req.Done().RequeueAfter(100 * time.Millisecond)
	}

	check.Status = true
	if check != obj.Status.Checks[NodesProcessed] {
		obj.Status.Checks[NodesProcessed] = check
		req.UpdateStatus()
	}

	return req.Next()
}

func (r *Reconciler) processHelmDeployments(req *rApi.Request[*clusterv1.BYOC]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := rApi.Check{Generation: obj.Generation}

	req.LogPreCheck(HelmDeploymentsProcessed)
	defer req.LogPostCheck(HelmDeploymentsProcessed)

	var dl appsv1.DeploymentList
	if err := r.List(ctx, &dl, &client.ListOptions{
		Namespace: r.Env.OperatorsNamespace,
	}); err != nil {
		return req.Done().Err(client.IgnoreNotFound(err))
	}

	for i := range dl.Items {
		check := rApi.Check{
			Status: dl.Items[i].Status.Replicas == dl.Items[i].Status.ReadyReplicas,
		}

		if !check.Status {
			var podList corev1.PodList
			if err := r.List(
				ctx, &podList, &client.ListOptions{
					LabelSelector: apiLabels.SelectorFromValidatedSet(map[string]string{"app": dl.Items[i].Name}),
					Namespace:     obj.Namespace,
				},
			); err != nil {
				check.Message = err.Error()
			}

			pMessages := rApi.GetMessagesFromPods(podList.Items...)
			bMsg, err := json.Marshal(pMessages)
			if err != nil {
				check.Message = err.Error()
			}
			if bMsg != nil {
				check.Message = string(bMsg)
			}
		}

		obj.Status.Checks[dl.Items[i].Name] = check
	}

	check.Status = true
	if check != obj.Status.Checks[HelmDeploymentsProcessed] {
		obj.Status.Checks[HelmDeploymentsProcessed] = check
		req.UpdateStatus()
	}

	return req.Next()
}

// func (r *Reconciler) sendStatusUpdate(ctx context.Context, obj *clusterv1.BYOC) (ctrl.Result, error) {
// 	obj.SetManagedFields(nil)
// 	b, err := json.Marshal(obj)
// 	if err != nil {
// 		return ctrl.Result{}, err
// 	}
//
// 	var m map[string]any
// 	if err := json.Unmarshal(b, &m); err != nil {
// 		return ctrl.Result{}, err
// 	}
//
// 	if err := r.dispatchBYOCClientUpdates(ctx, types.StatusUpdate{
// 		ClusterName: obj.Name,
// 		AccountName: obj.Spec.AccountName,
// 		Object:      m,
// 	}); err != nil {
// 		return ctrl.Result{}, err
// 	}
//
// 	r.Logger.WithKV("timestamp", time.Now()).Infof("dispatched update to message office api")
//
// 	if obj.GetDeletionTimestamp() != nil {
// 		if controllerutil.ContainsFinalizer(obj, byocClientFinalizer) {
// 			controllerutil.RemoveFinalizer(obj, byocClientFinalizer)
// 			return ctrl.Result{}, r.Update(ctx, obj)
// 		}
// 		return ctrl.Result{}, nil
// 	}
//
// 	if !controllerutil.ContainsFinalizer(obj, byocClientFinalizer) {
// 		controllerutil.AddFinalizer(obj, byocClientFinalizer)
// 		return ctrl.Result{}, r.Update(ctx, obj)
// 	}
// 	return ctrl.Result{}, nil
// }

func (r *Reconciler) finalize(req *rApi.Request[*clusterv1.BYOC]) stepResult.Result {
	return nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.Logger = logger.WithName(r.Name)

	// r.dispatchBYOCClientUpdates = func(ctx context.Context, stu types.StatusUpdate) error {
	// 	return fmt.Errorf("grpc connection not established yet")
	// }

	// go func() {
	// 	handlerCh := make(chan error, 1)
	// 	for {
	// 		logger.Infof("Waiting for grpc connection to setup")
	// 		cc, err := r.GetGrpcConnection()
	// 		if err != nil {
	// 			log.Fatalf("Failed to connect after retries: %v", err)
	// 		}
	//
	// 		logger.Infof("GRPC connection successful")
	//
	// 		msgDispatchCli := messages.NewMessageDispatchServiceClient(cc)
	//
	// 		mds, err := msgDispatchCli.ReceiveBYOCClientUpdates(context.Background())
	// 		// mds, err := msgDispatchCli.ReceiveStatusMessages(context.Background())
	// 		if err != nil {
	// 			logger.Errorf(err, "ReceiveBYOCClientUpdates")
	// 		}
	//
	// 		r.dispatchBYOCClientUpdates = func(_ context.Context, stu types.StatusUpdate) error {
	// 			b, err := json.Marshal(stu)
	// 			if err != nil {
	// 				return err
	// 			}
	//
	// 			if err = mds.Send(&messages.BYOCClientUpdateData{
	// 				AccessToken:             r.Env.AccessToken,
	// 				ClusterName:             r.Env.ClusterName,
	// 				AccountName:             r.Env.AccountName,
	// 				ByocClientUpdateMessage: b,
	// 			}); err != nil {
	// 				handlerCh <- err
	// 				return err
	// 			}
	// 			return nil
	// 		}
	//
	// 		connState := cc.GetState()
	// 		go func(cs connectivity.State) {
	// 			for cs != connectivity.Ready && connState != connectivity.Shutdown {
	// 				handlerCh <- fmt.Errorf("connection lost, will reconnect")
	// 			}
	// 		}(connState)
	// 		<-handlerCh
	// 		cc.Close()
	// 	}
	// }()

	builder := ctrl.NewControllerManagedBy(mgr).For(&clusterv1.BYOC{})
	watchList := []client.Object{
		&storagev1.StorageClass{},
		&networkingv1.IngressClass{},
		&corev1.Node{},
		&appsv1.Deployment{},
	}

	for i := range watchList {
		builder.Watches(
			&source.Kind{Type: watchList[i]},
			handler.EnqueueRequestsFromMapFunc(func(client.Object) []reconcile.Request {
				var byocList clusterv1.BYOCList
				if err := r.List(context.TODO(), &byocList); err != nil {
					return nil
				}

				reqs := make([]reconcile.Request, len(byocList.Items))
				for i := range byocList.Items {
					reqs[i] = reconcile.Request{
						NamespacedName: client.ObjectKeyFromObject(&byocList.Items[i]),
					}
				}
				return reqs
			}),
		)
	}

	builder.WithOptions(controller.Options{MaxConcurrentReconciles: r.Env.MaxConcurrentReconciles})
	builder.WithEventFilter(rApi.ReconcileFilter())
	return builder.Complete(r)
}
