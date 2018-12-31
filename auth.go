package s3gof3r

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// Keys for an Amazon Web Services account.
// Used for signing http requests.
type Keys struct {
	AccessKey     string
	SecretKey     string
	SecurityToken string
}

type mdCreds struct {
	Code            string
	LastUpdated     string
	Type            string
	AccessKeyID     string `xml:"AccessKeyId"`
	SecretAccessKey string
	Token           string
	Expiration      string
}

// Helper function for getting data
// from instance metadata server via string of role path
func getMetaDataKeys(rolePath string) (keys Keys, err error) {
	var creds mdCreds

	// request the role name for the instance
	// assumes there is only one
	resp, err := ClientWithTimeout(2 * time.Second).Get(rolePath)
	if err != nil {
		return
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 200 {
		err = newRespError(resp)
		return
	}
	role, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return
	}

	// request the credential metadata for the role
	resp, err = http.Get(rolePath + string(role))
	if err != nil {
		return
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 200 {
		err = newRespError(resp)
		return
	}
	metadata, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if err = json.Unmarshal([]byte(metadata), &creds); err != nil {
		return
	}
	keys = Keys{
		AccessKey:     creds.AccessKeyID,
		SecretKey:     creds.SecretAccessKey,
		SecurityToken: creds.Token,
	}

	return
}

// EcsKeys pulls the credentials inside a running
// ECS container. The URL requires AWS_CONTAINER_CREDENTIALS_RELATIVE_URI
// environment variable.
// See https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html
func ECSKeys() (keys Keys, err error) {
	roleRelativeUri := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")

	return getMetaDataKeys(fmt.Sprint("http://169.254.170.2/", roleRelativeUri))
}

// InstanceKeys Requests the AWS keys from the instance-based metadata on EC2
// Assumes only one IAM role.
func InstanceKeys() (keys Keys, err error) {
	return getMetaDataKeys("http://169.254.169.254/latest/meta-data/iam/security-credentials/")
}

// EnvKeys Reads the AWS keys from the environment
func EnvKeys() (keys Keys, err error) {
	keys = Keys{
		AccessKey:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey:     os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SecurityToken: os.Getenv("AWS_SECURITY_TOKEN"),
	}
	if keys.AccessKey == "" || keys.SecretKey == "" {
		err = fmt.Errorf("keys not set in environment: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY")
	}
	return
}
