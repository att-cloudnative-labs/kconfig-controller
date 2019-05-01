package util

import (
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"

	kconfigv1alpha1 "github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
)

// ByLevel sort function for KconfigEnvs
type ByLevel []kconfigv1alpha1.KconfigEnvs

func (c ByLevel) Len() int           { return len(c) }
func (c ByLevel) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c ByLevel) Less(i, j int) bool { return c[i].Level < c[j].Level }

// GetEnvKey returns string to use as the key in the map that holds envVars for each Kconfig
func GetEnvKey(namespace, name string) string {
	return namespace + "/" + name
}

// GetSecretRefKey returns a key to use for secretKeyRef based on scrubbed envkey and 5 char hash
func GetSecretRefKey(envkey, value string) (string, error) {
	// remove non alphanumeric characters from key
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "", err
	}
	scrubbedKey := strings.ToLower(reg.ReplaceAllString(envkey, ""))
	h := sha1.New()
	h.Write([]byte(envkey + value))
	bs := fmt.Sprintf("%x", h.Sum(nil))
	return fmt.Sprintf("%s-%.5s", scrubbedKey, bs), nil
}

// GetNewKeyReference returns key
func GetNewKeyReference(key string) (string, error) {
	// TODO: Do we still want keys to be hashed?
	// dateString := time.Now().Format("20060102150405")
	// remove non alphanumeric characters from key
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "", err
	}
	scrubbedKey := strings.ToLower(reg.ReplaceAllString(key, ""))
	// return fmt.Sprintf("%s-%s", scrubbedKey, dateString), nil
	return fmt.Sprintf("%s", scrubbedKey), nil
}
