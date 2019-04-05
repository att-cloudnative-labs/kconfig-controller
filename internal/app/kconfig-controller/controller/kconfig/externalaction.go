package kconfig

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalAction Object representing an addition to an external resource
// (ConfigMap or Secret). This object is intended to be used with the
// ExternalActionCache to store these updates so changes to the same resource
// can be updated all at once to avoid conflicts.
type ExternalAction struct {
	Key   string
	Value string
}

// ExternalActionCache cache of external actions
type ExternalActionCache map[string][]ExternalAction

// NewExternalActionCache creates new cache
func NewExternalActionCache() ExternalActionCache {
	return make(map[string][]ExternalAction)
}

// Put add action to cache
func (e ExternalActionCache) Put(key string, action ExternalAction) {
	if e[key] == nil {
		e[key] = make([]ExternalAction, 0)
	}
	e[key] = append(e[key], action)
}

// Get return action for key
func (e ExternalActionCache) Get(key string) []ExternalAction {
	return e[key]
}

// ExternalActionKeyFunc returns the cache key for a resource type (ConfigMap, Secret), namespace, name
func ExternalActionKeyFunc(resType, name string) string {
	return fmt.Sprintf("%s/%s", resType, name)
}

// SplitExternalActionKey returns type/namespace/name
func SplitExternalActionKey(key string) (string, string, error) {
	slice := strings.Split(key, "/")
	if len(slice) != 2 {
		return "", "", fmt.Errorf("invalid key")
	}
	return slice[0], slice[1], nil
}

// ExecuteExternalActions executes all actions
func (c *Controller) ExecuteExternalActions(namespace string, cache ExternalActionCache) error {
	for k := range cache {
		if resType, name, err := SplitExternalActionKey(k); err == nil {
			switch strings.ToLower(resType) {
			case "configmap":
				c.updateConfigMap(cache.Get(k), namespace, name)

			case "secret":
				c.updateSecret(cache.Get(k), namespace, name)
			}
		} else {
			return err
		}
	}
	return nil
}

func (c *Controller) updateConfigMap(actions []ExternalAction, namespace, name string) error {
	configMap, err := c.configmaplister.ConfigMaps(namespace).Get(name)
	create := false
	if err != nil {
		if errors.IsNotFound(err) {
			create = true
			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
				Data: make(map[string]string),
			}
		} else {
			return err
		}
	}
	for _, action := range actions {
		configMap.Data[action.Key] = action.Value
	}
	if create {
		_, err = c.stdclient.CoreV1().ConfigMaps(namespace).Create(configMap)
	} else {
		_, err = c.stdclient.CoreV1().ConfigMaps(namespace).Update(configMap)
	}
	return err
}

func (c *Controller) updateSecret(actions []ExternalAction, namespace, name string) error {
	secret, err := c.secretlister.Secrets(namespace).Get(name)
	create := false
	if err != nil {
		if errors.IsNotFound(err) {
			create = true
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
				Data: make(map[string][]byte),
			}
		} else {
			return err
		}
	}
	for _, action := range actions {
		secret.Data[action.Key] = []byte(action.Value)
	}
	if create {
		_, err = c.stdclient.CoreV1().Secrets(namespace).Create(secret)
	} else {
		_, err = c.stdclient.CoreV1().Secrets(namespace).Update(secret)
	}
	return err
}
