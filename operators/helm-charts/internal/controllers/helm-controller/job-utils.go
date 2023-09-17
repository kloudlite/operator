package helm_controller

import (
	"context"
	"encoding/json"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiLabels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetPodsForJob(ctx context.Context, cli client.Client, jobNamespace string, jobName string) (*corev1.PodList, error) {
	var podsList corev1.PodList

	if err := cli.List(ctx, &podsList, &client.ListOptions{
		Namespace: jobNamespace,
		LabelSelector: apiLabels.SelectorFromValidatedSet(map[string]string{
			"job-name": jobName,
		}),
	}); err != nil {
		return nil, err
	}

	return &podsList, nil
}

func GetLatestPodForJob(ctx context.Context, cli client.Client, jobNamespace string, jobName string) (*corev1.Pod, error) {
	pl, err := GetPodsForJob(ctx, cli, jobNamespace, jobName)
	if err != nil {
		return nil, err
	}
	if len(pl.Items) == 0 {
		return nil, fmt.Errorf("no pods found")
	}

	var latestPod corev1.Pod
	for i := range pl.Items {
		if pl.Items[i].CreationTimestamp.Unix()-latestPod.CreationTimestamp.Unix() > 0 {
			latestPod = pl.Items[i]
		}
	}

	return &latestPod, nil
}

func GetTerminationLogFromPod(ctx context.Context, pod *corev1.Pod) string {
	m := map[string]string{}

	for i := range pod.Status.ContainerStatuses {
		cs := pod.Status.ContainerStatuses[i]
		if cs.State.Terminated != nil {
			m[cs.Name] = cs.State.Terminated.Message
		}
	}

	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

type JobStatus string

const (
	JobStatusUnknown JobStatus = "job-status-unknown"
	JobStatusRunning JobStatus = "job-status-running"
	JobStatusFailed  JobStatus = "job-status-failed"
	JobStatusSuccess JobStatus = "job-status-success"
)

func checkIfJobHasFinished(ctx context.Context, job *batchv1.Job) JobStatus {
	for _, v := range job.Status.Conditions {
		if v.Type == batchv1.JobComplete && v.Status == "True" {
			return JobStatusSuccess
		}

		if v.Type == batchv1.JobFailed && v.Status == "True" {
			return JobStatusFailed
		}

		if v.Type == batchv1.JobSuspended && v.Status == "True" {
			return JobStatusFailed
		}
	}

	return JobStatusUnknown
}
