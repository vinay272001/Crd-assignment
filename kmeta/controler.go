package kmeta

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FilterController(r OwnerRefable) func(obj interface{}) bool {
	return FilterControllerGK(r.GetGroupVersionKind().GroupKind())
}

// FilterControllerGK makes it simple to create FilterFunc's for use with
// cache.FilteringResourceEventHandler that filter based on the
// schema.GroupKind of the controlling resources.
func FilterControllerGK(gk schema.GroupKind) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}

		owner := metav1.GetControllerOf(object)
		if owner == nil {
			return false
		}

		ownerGV, err := schema.ParseGroupVersion(owner.APIVersion)
		return err == nil &&
			ownerGV.Group == gk.Group &&
			owner.Kind == gk.Kind
	}
}