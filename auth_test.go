package s3gof3r

import (
	"os"
	"strings"
	"testing"
)

var testKeys = Keys{
	AccessKey: "AKIDEXAMPLE",
	SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
}

var testTokenKeys = Keys{
	AccessKey:     "AKIDEXAMPLE",
	SecretKey:     "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
	SecurityToken: "testtoken",
}

type authT struct {
	env []string
}

// save current environment for restoration
func (s *authT) saveEnv() {
	s.env = os.Environ()
	os.Clearenv()
}

// restore environment after each test
func (s *authT) restoreEnv() {
	os.Clearenv()
	for _, kv := range s.env {
		l := strings.SplitN(kv, "=", 2)
		os.Setenv(l[0], l[1])
	}
}

func TestEnvKeys(t *testing.T) {
	s := authT{}
	s.saveEnv()
	os.Setenv("AWS_ACCESS_KEY_ID", testKeys.AccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", testKeys.SecretKey)
	keys, err := EnvKeys()
	if err != nil {
		t.Error(err)
	}
	if keys != testKeys {
		t.Errorf("Keys do not match. Expected: %v. Actual: %v", testKeys, keys)
	}
	s.restoreEnv()
}

func TestEnvKeysNotSet(t *testing.T) {
	s := authT{}
	s.saveEnv()
	_, err := EnvKeys()
	expErr := "keys not set in environment: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY"
	if err.Error() != expErr {
		t.Errorf("Expected error: %v. Actual: %v", expErr, err)
	}
	s.restoreEnv()
}
