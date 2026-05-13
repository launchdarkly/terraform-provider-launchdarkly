package launchdarkly

import (
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// getCurrentCSA reads the current default_client_side_availability from the API
// so it can be passed through unchanged on PUT requests. This avoids conflicting
// with the launchdarkly_project resource which owns CSA settings.
func getCurrentCSA(client *Client, projectKey string) (*ldapi.DefaultClientSideAvailability, error) {
	var flagDefaults *ldapi.FlagDefaultsRep
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		flagDefaults, _, err = client.ld.ProjectsApi.GetFlagDefaultsByProject(client.ctx, projectKey).Execute()
		return err
	})
	if err != nil {
		return nil, err
	}

	// Convert from the GET response type (pointer fields) to the PUT request type (value fields).
	// Fallback defaults match LaunchDarkly's project defaults: using_environment_id=false, using_mobile_key=true.
	csa := ldapi.NewDefaultClientSideAvailability(false, true)
	if flagDefaults.DefaultClientSideAvailability != nil {
		if flagDefaults.DefaultClientSideAvailability.UsingMobileKey != nil {
			csa.UsingMobileKey = *flagDefaults.DefaultClientSideAvailability.UsingMobileKey
		}
		if flagDefaults.DefaultClientSideAvailability.UsingEnvironmentId != nil {
			csa.UsingEnvironmentId = *flagDefaults.DefaultClientSideAvailability.UsingEnvironmentId
		}
	}

	return csa, nil
}
