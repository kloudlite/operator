package operator

import (
	"encoding/json"
	"github.com/kloudlite/operator/apis/common-types"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	jsonPatch "github.com/kloudlite/operator/pkg/json-patch"
)

type res struct {
	Enabled   bool `json:"enabled,omitempty"`
	Overrides struct {
		Patches []jsonPatch.PatchOperation `json:"patches,omitempty"`
	} `json:"overrides,omitempty"`
	Status struct {
		Checks  map[string]common_types.Check `json:"checks,omitempty"`
		IsReady *bool                         `json:"isReady"`
	} `json:"status"`
}

func getRes(obj runtime.Object) res {
	b, err := json.Marshal(obj)
	if err != nil {
		return res{}
	}
	var xRes res
	if err := json.Unmarshal(b, &xRes); err != nil {
		return res{}
	}

	return xRes
}

func ReconcileFilter() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(ev event.UpdateEvent) bool {
			oldObj := ev.ObjectOld
			newObj := ev.ObjectNew

			if newObj.GetGeneration() > oldObj.GetGeneration() {
				return true
			}

			if newObj.GetDeletionTimestamp() != oldObj.GetDeletionTimestamp() {
				return true
			}

			if len(oldObj.GetLabels()) != len(newObj.GetLabels()) || !reflect.DeepEqual(oldObj.GetLabels(), newObj.GetLabels()) {
				return true
			}

			if len(oldObj.GetAnnotations()) != len(newObj.GetAnnotations()) || !reflect.DeepEqual(oldObj.GetAnnotations(), newObj.GetAnnotations()) {
				return true
			}

			if len(oldObj.GetFinalizers()) != len(newObj.GetFinalizers()) || !reflect.DeepEqual(oldObj.GetFinalizers(), newObj.GetFinalizers()) {
				return true
			}

			if len(oldObj.GetOwnerReferences()) != len(newObj.GetOwnerReferences()) ||
				!reflect.DeepEqual(oldObj.GetOwnerReferences(), newObj.GetOwnerReferences()) {
				return true
			}

			oldRes, newRes := getRes(ev.ObjectOld), getRes(ev.ObjectNew)
			if oldRes.Enabled != newRes.Enabled {
				return true
			}

			if !reflect.DeepEqual(oldRes.Overrides, newRes.Overrides) {
				return true
			}

			if oldRes.Status.IsReady == nil || newRes.Status.IsReady == nil {
				// this is not our object, it is some other k8s resource, just defaulting it to be always watched
				return true
			}

			if *oldRes.Status.IsReady != *newRes.Status.IsReady {
				return true
			}

			if len(oldRes.Status.Checks) != len(newRes.Status.Checks) {
				return true
			}

			for k, v := range oldRes.Status.Checks {
				if newRes.Status.Checks[k] != v {
					return true
				}
			}
			return false
		},
	}
}
