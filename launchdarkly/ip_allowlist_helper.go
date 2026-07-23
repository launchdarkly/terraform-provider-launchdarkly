package launchdarkly

import (
	"sync"

	ldapi "github.com/launchdarkly/api-client-go/v23"
)

// LD's IP allowlist API persists the whole allowlist as a single account
// document with optimistic concurrency. Two truly-simultaneous writes can
// race that version. ipAllowlistWriteMu serialises POST/PATCH/PUT/DELETE
// in-process so two test goroutines (e.g. parallel TestCases inside one
// test binary) never race the version. GETs skip the mutex.
//
// Note: the API returns 409 `optimistic_locking_error` for BOTH genuine
// version races AND attempts to insert a duplicate IP. The two are
// indistinguishable from the error body, so we don't auto-retry — a
// retry on a duplicate-IP 409 is a no-op that wastes time and can mask
// orphans from prior failed tests. Cleanup hooks in test PreChecks
// handle the orphan scenario.
var ipAllowlistWriteMu sync.Mutex

// ipAllowlistBetaClient returns a beta-configured client for the IP
// allowlist endpoints. The generated IPAllowlistBetaApi request builders
// do not expose a per-request `.LDAPIVersion("beta")` setter, so we force
// the `LD-API-Version: beta` header at the transport layer (see
// betaVersionRoundTripper for why a default header is insufficient).
func ipAllowlistBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	forceBetaAPIVersion(beta.ld)
	forceBetaAPIVersion(beta.ld404Retry)
	return beta, nil
}

func getIpAllowlist(client *Client) (*ldapi.IpAllowlistResponse, error) {
	beta, err := ipAllowlistBetaClient(client)
	if err != nil {
		return nil, err
	}

	var result *ldapi.IpAllowlistResponse
	err = beta.withConcurrency(beta.ctx, func() error {
		result, _, err = beta.ld.IPAllowlistBetaApi.GetIpAllowlist(beta.ctx).Execute()
		return err
	})
	if err != nil {
		return nil, handleLdapiErr(err)
	}
	return result, nil
}

func createIpAllowlistEntry(client *Client, ipAddress string, description *string) (*ldapi.IpAllowlistEntryResponse, error) {
	beta, err := ipAllowlistBetaClient(client)
	if err != nil {
		return nil, err
	}

	reqBody := ldapi.CreateIpAllowlistEntryRequest{
		IpAddress:   ipAddress,
		Description: description,
	}

	ipAllowlistWriteMu.Lock()
	defer ipAllowlistWriteMu.Unlock()

	var result *ldapi.IpAllowlistEntryResponse
	err = beta.withConcurrency(beta.ctx, func() error {
		result, _, err = beta.ld.IPAllowlistBetaApi.CreateIpAllowlistEntry(beta.ctx).
			CreateIpAllowlistEntryRequest(reqBody).
			Execute()
		return err
	})
	if err != nil {
		return nil, handleLdapiErr(err)
	}
	return result, nil
}

func patchIpAllowlistEntry(client *Client, id string, description string) (*ldapi.IpAllowlistEntryResponse, error) {
	beta, err := ipAllowlistBetaClient(client)
	if err != nil {
		return nil, err
	}

	reqBody := ldapi.PatchIpAllowlistEntryRequest{
		Description: description,
	}

	ipAllowlistWriteMu.Lock()
	defer ipAllowlistWriteMu.Unlock()

	var result *ldapi.IpAllowlistEntryResponse
	err = beta.withConcurrency(beta.ctx, func() error {
		result, _, err = beta.ld.IPAllowlistBetaApi.PatchIpAllowlistEntry(beta.ctx, id).
			PatchIpAllowlistEntryRequest(reqBody).
			Execute()
		return err
	})
	if err != nil {
		return nil, handleLdapiErr(err)
	}
	return result, nil
}

func deleteIpAllowlistEntry(client *Client, id string) error {
	beta, err := ipAllowlistBetaClient(client)
	if err != nil {
		return err
	}

	ipAllowlistWriteMu.Lock()
	defer ipAllowlistWriteMu.Unlock()

	err = beta.withConcurrency(beta.ctx, func() error {
		_, err = beta.ld.IPAllowlistBetaApi.DeleteIpAllowlistEntry(beta.ctx, id).Execute()
		return err
	})
	if err != nil {
		return handleLdapiErr(err)
	}
	return nil
}

func patchIpAllowlistConfig(client *Client, sessionEnabled, scopedEnabled *bool) (*ldapi.IpAllowlistResponse, error) {
	beta, err := ipAllowlistBetaClient(client)
	if err != nil {
		return nil, err
	}

	reqBody := ldapi.PatchIpAllowlistConfigRequest{
		SessionAllowlistEnabled:  sessionEnabled,
		ApiTokenAllowlistEnabled: scopedEnabled,
	}

	ipAllowlistWriteMu.Lock()
	defer ipAllowlistWriteMu.Unlock()

	var result *ldapi.IpAllowlistResponse
	err = beta.withConcurrency(beta.ctx, func() error {
		result, _, err = beta.ld.IPAllowlistBetaApi.PatchIpAllowlistConfig(beta.ctx).
			PatchIpAllowlistConfigRequest(reqBody).
			Execute()
		return err
	})
	if err != nil {
		return nil, handleLdapiErr(err)
	}
	return result, nil
}

func findIpAllowlistEntryByID(entries []ldapi.IpAllowlistEntryResponse, id string) *ldapi.IpAllowlistEntryResponse {
	for _, entry := range entries {
		if entry.Id == id {
			return &entry
		}
	}
	return nil
}
