package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	ldapi "github.com/launchdarkly/api-client-go/v23"
)

// sdkKeyID builds the composite Terraform ID for an SDK key.
// Format: <projectKey>/<environmentKey>/<sdkKeyKey>.
func sdkKeyID(projectKey, environmentKey, sdkKeyKey string) string {
	return fmt.Sprintf("%s/%s/%s", projectKey, environmentKey, sdkKeyKey)
}

// sdkKeyIDToKeys splits a composite SDK key ID into its three parts.
func sdkKeyIDToKeys(id string) (projectKey, environmentKey, sdkKeyKey string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("found unexpected SDK key ID format: %q. expected format: 'project_key/environment_key/key'", id)
	}
	return parts[0], parts[1], parts[2], nil
}

// getSdkKey fetches a single SDK key by its identifying key. The SDK Keys API
// is a beta endpoint, so callers must pass a beta-configured client and the
// request is issued with the beta LD-API-Version.
func getSdkKey(client *Client, projectKey, environmentKey, sdkKeyKey string) (*ldapi.SdkKey, *http.Response, error) {
	var (
		sdkKey *ldapi.SdkKey
		res    *http.Response
		err    error
	)
	err = client.withConcurrency(client.ctx, func() error {
		sdkKey, res, err = client.ld.SDKKeysBetaApi.GetSdkKeyByKey(client.ctx, projectKey, environmentKey, sdkKeyKey).
			LDAPIVersion("beta").
			Execute()
		return err
	})
	if err != nil {
		return nil, res, err
	}
	return sdkKey, res, nil
}

// createSdkKey creates a new SDK key in the given environment.
func createSdkKey(client *Client, projectKey, environmentKey string, post ldapi.SdkKeyPost) (*ldapi.SdkKey, error) {
	var (
		sdkKey *ldapi.SdkKey
		err    error
	)
	err = client.withConcurrency(client.ctx, func() error {
		sdkKey, _, err = client.ld.SDKKeysBetaApi.PostSdkKey(client.ctx, projectKey, environmentKey).
			LDAPIVersion("beta").
			SdkKeyPost(post).
			Execute()
		return err
	})
	if err != nil {
		return nil, err
	}
	return sdkKey, nil
}

// patchSdkKey updates the mutable fields (name, description, expiry) of an
// existing SDK key.
func patchSdkKey(client *Client, projectKey, environmentKey, sdkKeyKey string, patch ldapi.SdkKeyPatch) (*ldapi.SdkKey, error) {
	var (
		sdkKey *ldapi.SdkKey
		err    error
	)
	err = client.withConcurrency(client.ctx, func() error {
		sdkKey, _, err = client.ld.SDKKeysBetaApi.PatchSdkKeyByKey(client.ctx, projectKey, environmentKey, sdkKeyKey).
			LDAPIVersion("beta").
			SdkKeyPatch(patch).
			Execute()
		return err
	})
	if err != nil {
		return nil, err
	}
	return sdkKey, nil
}

// deleteSdkKey deletes an SDK key by its identifying key.
func deleteSdkKey(client *Client, projectKey, environmentKey, sdkKeyKey string) (*http.Response, error) {
	var (
		res *http.Response
		err error
	)
	err = client.withConcurrency(client.ctx, func() error {
		res, err = client.ld.SDKKeysBetaApi.DeleteSdkKeyByKey(client.ctx, projectKey, environmentKey, sdkKeyKey).
			LDAPIVersion("beta").
			Execute()
		return err
	})
	return res, err
}
