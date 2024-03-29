package v1

import (
	"context"
	"fmt"
	"path/filepath"

	"sigs.k8s.io/controller-runtime/pkg/client"

	fn "github.com/kloudlite/operator/pkg/functions"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Bool bool

func (b Bool) Status() metav1.ConditionStatus {
	if b {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

type Condition struct {
	Type               string
	Status             string // "True", "False", "Unknown"
	ObservedGeneration int64
	Reason             string
	Message            string
}

type Operations struct {
	Apply  string `json:"create"`
	Delete string `json:"delete"`
}

type Output struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// +kubebuilder:validation:Enum=config;secret;pvc
type ConfigOrSecret string

const (
	SecretType ConfigOrSecret = "secret"
	ConfigType ConfigOrSecret = "config"
	PVCType    ConfigOrSecret = "pvc"
)

func ParseVolumes(containers []AppContainer) (volumes []corev1.Volume, volumeMounts [][]corev1.VolumeMount) {
	m := map[string][]ContainerVolume{}

	for _, container := range containers {
		mounts := make([]corev1.VolumeMount, 0, len(container.Volumes))

		for _, volume := range container.Volumes {
			volName := fn.Md5([]byte(volume.MountPath))
			if len(volName) > 50 {
				volName = volName[:50]
			}
			if m[volName] == nil {
				m[volName] = []ContainerVolume{}
			}
			m[volName] = append(m[volName], volume)

			if len(volume.Items) > 0 {
				for _, item := range volume.Items {
					if item.FileName == "" {
						item.FileName = item.Key
					}

					mount := corev1.VolumeMount{
						Name:      volName,
						MountPath: filepath.Join(volume.MountPath, item.FileName),
						SubPath:   item.FileName,
					}
					mounts = append(mounts, mount)
				}
			} else {
				mount := corev1.VolumeMount{
					Name:      volName,
					MountPath: volume.MountPath,
				}

				mounts = append(mounts, mount)
			}
		}

		volumeMounts = append(volumeMounts, mounts)
	}

	for k, cVolumes := range m {
		volume := corev1.Volume{Name: k}

		// volume.VolumeSource.Projected = nil
		for _, volm := range cVolumes {
			// projection := corev1.VolumeProjection{}
			var kp []corev1.KeyToPath
			if len(volm.Items) > 0 {
				for _, item := range volm.Items {
					if item.FileName == "" {
						item.FileName = item.Key
					}
					kp = append(
						kp, corev1.KeyToPath{
							Key:  item.Key,
							Path: item.FileName,
							Mode: nil,
						},
					)
				}
			}
			switch volm.Type {
			case SecretType:
				{
					if volume.Projected == nil {
						volume.Projected = &corev1.ProjectedVolumeSource{}
					}
					volume.Projected.Sources = append(volume.Projected.Sources, corev1.VolumeProjection{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: volm.RefName,
							},
							Items: kp,
						},
					})
				}
			case ConfigType:
				{
					if volume.Projected == nil {
						volume.Projected = &corev1.ProjectedVolumeSource{}
					}
					volume.Projected.Sources = append(volume.Projected.Sources, corev1.VolumeProjection{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: volm.RefName,
							},
							Items: kp,
						},
					})
				}
			case PVCType:
				{
					volume.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: volm.RefName,
						ReadOnly:  false,
					}
				}
			default:
				{
					fmt.Println("invalid type, not config, secret")
				}
			}
		}
		volumes = append(volumes, volume)
	}

	return volumes, volumeMounts
}

func IsBlueprintNamespace(ctx context.Context, k8sClient client.Client, ns string) bool {
	var prj Project
	err := k8sClient.Get(ctx, fn.NN("", ns), &prj)
	return err != nil
}
