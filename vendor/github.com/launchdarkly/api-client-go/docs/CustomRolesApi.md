# \CustomRolesApi

All URIs are relative to *https://app.launchdarkly.com/api/v2*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeleteCustomRole**](CustomRolesApi.md#DeleteCustomRole) | **Delete** /roles/{customRoleKey} | Delete a custom role by key.
[**GetCustomRole**](CustomRolesApi.md#GetCustomRole) | **Get** /roles/{customRoleKey} | Get one custom role by key.
[**GetCustomRoles**](CustomRolesApi.md#GetCustomRoles) | **Get** /roles | Return a complete list of custom roles.
[**PatchCustomRole**](CustomRolesApi.md#PatchCustomRole) | **Patch** /roles/{customRoleKey} | Modify a custom role by key.
[**PostCustomRole**](CustomRolesApi.md#PostCustomRole) | **Post** /roles | Create a new custom role.


# **DeleteCustomRole**
> DeleteCustomRole(ctx, customRoleKey)
Delete a custom role by key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **customRoleKey** | **string**| The custom role key. | 

### Return type

 (empty response body)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetCustomRole**
> CustomRole GetCustomRole(ctx, customRoleKey)
Get one custom role by key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **customRoleKey** | **string**| The custom role key. | 

### Return type

[**CustomRole**](CustomRole.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetCustomRoles**
> CustomRoles GetCustomRoles(ctx, )
Return a complete list of custom roles.

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**CustomRoles**](CustomRoles.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PatchCustomRole**
> CustomRole PatchCustomRole(ctx, customRoleKey, patchDelta)
Modify a custom role by key.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **customRoleKey** | **string**| The custom role key. | 
  **patchDelta** | [**[]PatchOperation**](PatchOperation.md)| Requires a JSON Patch representation of the desired changes to the project. &#39;http://jsonpatch.com/&#39; | 

### Return type

[**CustomRole**](CustomRole.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PostCustomRole**
> PostCustomRole(ctx, customRoleBody)
Create a new custom role.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **customRoleBody** | [**CustomRoleBody**](CustomRoleBody.md)| New role or roles to create. | 

### Return type

 (empty response body)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

