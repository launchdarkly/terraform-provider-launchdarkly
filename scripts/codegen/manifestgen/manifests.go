package manifestgen

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type ManifestsResponse struct {
	Items []Manifest
}

type Manifest struct {
	Key           string
	RequiresOauth bool
	Capabilities  map[string]interface{}
	FormVariables []FormVariable
}

type FormVariable struct {
	AllowedValues []string
	DefaultValue  interface{}
	Description   string
	IsOptional    bool
	IsSecret      bool
	Key           string
	Type          string
}

func FetchManifests(appHost, accessToken string) ([]Manifest, error) {
	manifestURL := url.URL{
		Scheme: "https",
		Host:   appHost,
		Path:   "api/v2/integration-manifests",
	}
	req, err := http.NewRequest(http.MethodGet, manifestURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", accessToken)

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.New("Get a nil response when fetching manifests")
	}
	if res.Body == nil {
		return nil, errors.New("Got a nil response body when fetching manifests")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var responseBody ManifestsResponse
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Items, nil
}
