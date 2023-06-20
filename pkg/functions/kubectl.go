package functions

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"strings"

	"github.com/gobuffalo/flect"
)

// func KubectlApplyExec(ctx context.Context, stdin ...[]byte) (err error) {
// 	c := exec.Command("kubectl", "apply", "-f", "-")
// 	outStream, errStream := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
// 	inputYAML := bytes.Join(stdin, []byte("\n---\n"))
// 	c.Stdin = bytes.NewBuffer(inputYAML)
// 	c.Stdout = outStream
// 	c.Stderr = errStream
//
// 	logger, hasLogger := ctx.Value("logger").(logging.Logger)
// 	if hasLogger {
// 		logger = logger.WithName("kubectl")
// 	}
//
// 	if err := c.Run(); err != nil {
// 		if hasLogger {
// 			logger.Debugf("input YAML: \n#---START---\n%s\n#---END---", inputYAML)
// 			logger.Errorf(err, errStream.String())
// 		}
// 		return errors.NewEf(err, errStream.String())
// 	}
// 	if hasLogger {
// 		logger.Infof(outStream.String())
// 	}
// 	return nil
// }

func toUnstructured(obj client.Object) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	t := &unstructured.Unstructured{Object: m}
	return t, nil
}

func AsOwner(r client.Object, controller ...bool) metav1.OwnerReference {
	ctrler := false
	if len(controller) > 0 {
		ctrler = controller[0]
	}
	return metav1.OwnerReference{
		APIVersion: r.GetObjectKind().GroupVersionKind().GroupVersion().String(),
		Kind:       r.GetObjectKind().GroupVersionKind().Kind,
		Name:       r.GetName(),
		UID:        r.GetUID(),
		Controller: &ctrler,
		//BlockOwnerDeletion: New(false),
		BlockOwnerDeletion: &ctrler,
	}
}

func IsOwner(obj client.Object, ownerRef metav1.OwnerReference) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Name == ownerRef.Name &&
			ref.UID == ownerRef.UID &&
			ref.Kind == ownerRef.Kind && ref.
			APIVersion == ownerRef.APIVersion {
			return true
		}
	}
	return false
}

func GVK(obj client.Object) metav1.GroupVersionKind {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return metav1.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}
}

// RegularPlural is used to pluralize group of k8s CRDs from kind
// It is copied from https://github.com/kubernetes-sigs/kubebuilder/blob/afce6a0e8c2a6d5682be07bbe502e728dd619714/pkg/model/resource/utils.go#L71
func RegularPlural(singular string) string {
	return flect.Pluralize(strings.ToLower(singular))
}
