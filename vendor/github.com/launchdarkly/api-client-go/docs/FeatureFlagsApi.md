# \FeatureFlagsApi

All URIs are relative to *https://app.launchdarkly.com/api/v2*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CopyFeatureFlag**](FeatureFlagsApi.md#CopyFeatureFlag) | **Post** /flags/{projectKey}/{environmentKey}/{featureFlagKey}/copy | Copies the feature flag configuration from one environment to the same feature flag in another environment.
[**DeleteFeatureFlag**](FeatureFlagsApi.md#DeleteFeatureFlag) | **Delete** /flags/{projectKey}/{featureFlagKey} | Delete a feature flag in all environments. Be careful-- only delete feature flags that are no longer being used by your application.
[**GetFeatureFlag**](FeatureFlagsApi.md#GetFeatureFlag) | **Get** /flags/{projectKey}/{featureFlagKey} | Get a single feature flag by key.
[**GetFeatureFlagStatus**](FeatureFlagsApi.md#GetFeatureFlagStatus) | **Get** /flag-statuses/{projectKey}/{environmentKey}/{featureFlagKey} | Get the status for a particular feature flag.
[**GetFeatureFlagStatuses**](FeatureFlagsApi.md#GetFeatureFlagStatuses) | **Get** /flag-statuses/{projectKey}/{environmentKey} | Get a list of statuses for all feature flags. The status includes the last time the feature flag was requested, as well as the state of the flag.
[**GetFeatureFlags**](FeatureFlagsApi.md#GetFeatureFlags) | **Get** /flags/{projectKey} | Get a list of all features in the given project.
[**PatchFeatureFlag**](FeatureFlagsApi.md#PatchFeatureFlag) | **Patch** /flags/{projectKey}/{featureFlagKey} | Perform a partial update to a feature.
[**PostFeatureFlag**](FeatureFlagsApi.md#PostFeatureFlag) | **Post** /flags/{projectKey} | Creates a new feature flag.


# **CopyFeatureFlag**
> FeatureFlag CopyFeatureFlag(ctx, projectKey, environmentKey, featureFlagKey)
Copies the feature flag configuration from one environment to the same feature flag in another environment.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **environmentKey** | **string**| The environment key, used to tie together flag configuration and users under one environment so they can be managed together. | 
  **featureFlagKey** | **string**| The feature flag&#39;s key. The key identifies the flag in your code. | 

### Return type

[**FeatureFlag**](FeatureFlag.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteFeatureFlag**
> DeleteFeatureFlag(ctx, projectKey, featureFlagKey)
Delete a feature flag in all environments. Be careful-- only delete feature flags that are no longer being used by your application.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **featureFlagKey** | **string**| The feature flag&#39;s key. The key identifies the flag in your code. | 

### Return type

 (empty response body)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetFeatureFlag**
> FeatureFlag GetFeatureFlag(ctx, projectKey, featureFlagKey, optional)
Get a single feature flag by key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **featureFlagKey** | **string**| The feature flag&#39;s key. The key identifies the flag in your code. | 
 **optional** | ***GetFeatureFlagOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetFeatureFlagOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **env** | **optional.String**| By default, each feature will include configurations for each environment. You can filter environments with the env query parameter. For example, setting env&#x3D;production will restrict the returned configurations to just your production environment. | 

### Return type

[**FeatureFlag**](FeatureFlag.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetFeatureFlagStatus**
> FeatureFlagStatus GetFeatureFlagStatus(ctx, projectKey, environmentKey, featureFlagKey)
Get the status for a particular feature flag.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **environmentKey** | **string**| The environment key, used to tie together flag configuration and users under one environment so they can be managed together. | 
  **featureFlagKey** | **string**| The feature flag&#39;s key. The key identifies the flag in your code. | 

### Return type

[**FeatureFlagStatus**](FeatureFlagStatus.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetFeatureFlagStatuses**
> FeatureFlagStatuses GetFeatureFlagStatuses(ctx, projectKey, environmentKey)
Get a list of statuses for all feature flags. The status includes the last time the feature flag was requested, as well as the state of the flag.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **environmentKey** | **string**| The environment key, used to tie together flag configuration and users under one environment so they can be managed together. | 

### Return type

[**FeatureFlagStatuses**](FeatureFlagStatuses.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetFeatureFlags**
> FeatureFlags GetFeatureFlags(ctx, projectKey, optional)
Get a list of all features in the given project.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
 **optional** | ***GetFeatureFlagsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetFeatureFlagsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **env** | **optional.String**| By default, each feature will include configurations for each environment. You can filter environments with the env query parameter. For example, setting env&#x3D;production will restrict the returned configurations to just your production environment. | 
 **tag** | **optional.String**| Filter by tag. A tag can be used to group flags across projects. | 

### Return type

[**FeatureFlags**](FeatureFlags.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PatchFeatureFlag**
> FeatureFlag PatchFeatureFlag(ctx, projectKey, featureFlagKey, patchComment)
Perform a partial update to a feature.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **featureFlagKey** | **string**| The feature flag&#39;s key. The key identifies the flag in your code. | 
  **patchComment** | [**PatchComment**](PatchComment.md)| Requires a JSON Patch representation of the desired changes to the project, and an optional comment. &#39;http://jsonpatch.com/&#39; Feature flag patches also support JSON Merge Patch format. &#39;https://tools.ietf.org/html/rfc7386&#39; The addition of comments is also supported. | 

### Return type

[**FeatureFlag**](FeatureFlag.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PostFeatureFlag**
> FeatureFlag PostFeatureFlag(ctx, projectKey, featureFlagBody, optional)
Creates a new feature flag.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **featureFlagBody** | [**FeatureFlagBody**](FeatureFlagBody.md)| Create a new feature flag. | 
 **optional** | ***PostFeatureFlagOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PostFeatureFlagOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **clone** | **optional.String**| The key of the feature flag to be cloned. The key identifies the flag in your code.  For example, setting clone&#x3D;flagKey will copy the full targeting configuration for all environments (including on/off state) from the original flag to the new flag. | 

### Return type

[**FeatureFlag**](FeatureFlag.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

