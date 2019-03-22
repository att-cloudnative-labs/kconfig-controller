package util

const (
	// DefaultNamespace Default Namespace
	DefaultNamespace = "testnamespace"
	// DefaultName default name
	DefaultName = "testname"
	// DefaultLevel Default level
	DefaultLevel = 1
	// DefaultSelectorKey Default selector key
	DefaultSelectorKey = "app"
	// DefaultSelectorValue Default selector value
	DefaultSelectorValue = "testapp"
	// DefaultKey Default Key
	DefaultKey = "TEST_KEY"
	// DefaultReferenceKey Default reference key, must match conversion of defaultKey
	DefaultReferenceKey = "testkey"
	// DefaultFieldPath Default fieldPath
	DefaultFieldPath = "status.hostIP"
	// DefaultResourceFieldRefResource Default ResourceFieldRef resource
	DefaultResourceFieldRefResource = "requests.cpu"
	// DefaultValue Default Value
	DefaultValue = "testvalue"
	// DefaultEncodedValue Value base64 encoded
	DefaultEncodedValue = "dGVzdHZhbHVl"
	// DefaultNewValue Additional Default Value
	DefaultNewValue = "testnewvalue"
	// DefaultEncodedNewValue Additional value base64 encoded
	DefaultEncodedNewValue = "dGVzdG5ld3ZhbHVl"
	// ValueType Value Type
	ValueType = "Value"
	// ConfigMapType ConfigMap Type
	ConfigMapType = "ConfigMap"
	// SecretType Secret Type
	SecretType = "Secret"
	// FieldRefType FieldRef Type
	FieldRefType = "FieldRef"
	// ResourceFieldRefType ResourceFieldRef Type
	ResourceFieldRefType = "ResourceFieldRef"
	// DefaultConfigMapName Default ConfigMap Name
	DefaultConfigMapName = "testconfigmapname"
	// DefaultSecretName Default Secret Name
	DefaultSecretName = "testsecretname"
	// DefaultKconfigEnvsKey Default KconfigEnvs Key
	DefaultKconfigEnvsKey = DefaultNamespace + "/" + DefaultName
	// TestNamespace test namespace
	TestNamespace = "testnamespace"
	// TestName test name
	TestName = "testname"
	// TestKey1 test key 1
	TestKey1 = "TESTKEY1"
	// TestKey2 test key 2
	TestKey2 = "TESTKEY2"
	// TestSecretKey1 test secret key 1
	TestSecretKey1 = "TESTSECRETKEY1"
	// TestValue1 test value 1
	TestValue1 = "testvalue1"
	// TestValue2 test value 1
	TestValue2 = "testvalue2"
	// TestSecretValue1 test secret value 1
	TestSecretValue1 = "testsecretvalue1"
	// TestSecretKeySelectorKey1 test SecretKeySelector key
	TestSecretKeySelectorKey1 = "testsecretkeyselector1"
	// TestAppKey test app key
	TestAppKey = "app"
	// TestAppName test app name
	TestAppName = "testapp"
	// TestSecretName1 test secret name 1
	TestSecretName1 = "testsecretname"
	// AddAction add action
	AddAction = "Add"
	// UpdateAction update action
	UpdateAction = "Update"
	// DeleteAction delete action
	DeleteAction = "Delete"
)
