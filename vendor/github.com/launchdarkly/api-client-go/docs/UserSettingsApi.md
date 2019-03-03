# \UserSettingsApi

All URIs are relative to *https://app.launchdarkly.com/api/v2*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetUserFlagSetting**](UserSettingsApi.md#GetUserFlagSetting) | **Get** /users/{projectKey}/{environmentKey}/{userKey}/flags/{featureFlagKey} | Fetch a single flag setting for a user by key.
[**GetUserFlagSettings**](UserSettingsApi.md#GetUserFlagSettings) | **Get** /users/{projectKey}/{environmentKey}/{userKey}/flags | Fetch a single flag setting for a user by key.
[**PutFlagSetting**](UserSettingsApi.md#PutFlagSetting) | **Put** /users/{projectKey}/{environmentKey}/{userKey}/flags/{featureFlagKey} | Specifically enable or disable a feature flag for a user based on their key.


# **GetUserFlagSetting**
> UserFlagSetting GetUserFlagSetting(ctx, projectKey, environmentKey, userKey, featureFlagKey)
Fetch a single flag setting for a user by key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **environmentKey** | **string**| The environment key, used to tie together flag configuration and users under one environment so they can be managed together. | 
  **userKey** | **string**| The user&#39;s key. | 
  **featureFlagKey** | **string**| The feature flag&#39;s key. The key identifies the flag in your code. | 

### Return type

[**UserFlagSetting**](UserFlagSetting.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetUserFlagSettings**
> UserFlagSettings GetUserFlagSettings(ctx, projectKey, environmentKey, userKey)
Fetch a single flag setting for a user by key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **environmentKey** | **string**| The environment key, used to tie together flag configuration and users under one environment so they can be managed together. | 
  **userKey** | **string**| The user&#39;s key. | 

### Return type

[**UserFlagSettings**](UserFlagSettings.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PutFlagSetting**
> PutFlagSetting(ctx, projectKey, environmentKey, userKey, featureFlagKey, userSettingsBody)
Specifically enable or disable a feature flag for a user based on their key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **projectKey** | **string**| The project key, used to tie the flags together under one project so they can be managed together. | 
  **environmentKey** | **string**| The environment key, used to tie together flag configuration and users under one environment so they can be managed together. | 
  **userKey** | **string**| The user&#39;s key. | 
  **featureFlagKey** | **string**| The feature flag&#39;s key. The key identifies the flag in your code. | 
  **userSettingsBody** | [**UserSettingsBody**](UserSettingsBody.md)|  | 

### Return type

 (empty response body)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

