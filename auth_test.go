package s3gof3r

import (
	"os"
	"strings"
	"testing"
	. "launchpad.net/gocheck"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var testKeys = Keys{
	AccessKey: "AKIDEXAMPLE",
	SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
}

var testTokenKeys = Keys{
	AccessKey:     "AKIDEXAMPLE",
	SecretKey:     "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
	SecurityToken: "testtoken",
}

type AuthT struct {
	env []string
}

var _ = Suite(&AuthT{})

// save current environment for restoration
func (s *AuthT) SetUpSuite(c *C) {
	s.env = os.Environ()
}

func (s *AuthT) SetUpTest(c *C) {
	os.Clearenv()
}

// restore environment after each test
func (s *AuthT) TearDownTest(c *C) {
	os.Clearenv()
	for _, kv := range s.env {
		l := strings.SplitN(kv, "=", 2)
		os.Setenv(l[0], l[1])
	}
}

func (s *AuthT) TestEnvKeys(c *C) {
	os.Setenv("AWS_ACCESS_KEY_ID", testKeys.AccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", testKeys.SecretKey)
	keys, err := EnvKeys()
	c.Assert(err, IsNil)
	c.Assert(keys, Equals, testKeys)
}

func (s *AuthT) TestEnvKeysNotSet(c *C) {
	_, err := EnvKeys()
	c.Assert(err, ErrorMatches, "Keys not set in environment: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY")
}
