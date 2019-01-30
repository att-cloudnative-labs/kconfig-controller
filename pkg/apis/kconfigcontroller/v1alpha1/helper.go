package v1alpha1

import (
	"reflect"
)

// KconfigEqual returns if kconfig is equal to another
func (k Kconfig) KconfigEqual(other Kconfig) bool {
	return reflect.DeepEqual(k.Spec, other.Spec)
}
