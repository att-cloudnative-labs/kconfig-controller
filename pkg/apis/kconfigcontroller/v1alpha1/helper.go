package v1alpha1

import (
	"reflect"
)

// func (s *KconfigSecretValue) String() string {
// 	if s == nil {
// 		return "nil"
// 	}
// 	return "{SecretKeyRef:" + s.SecretKeyRef.String() + ", SecretName:" + s.SecretName + ",Action:" + s.Action + "}"
// }

// KconfigEqual returns if kconfig is equal to another
func (k Kconfig) KconfigEqual(other Kconfig) bool {
	return reflect.DeepEqual(k.Spec, other.Spec)
}
