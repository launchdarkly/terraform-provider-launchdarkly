# \WebhooksApi

All URIs are relative to *https://app.launchdarkly.com/api/v2*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeleteWebhook**](WebhooksApi.md#DeleteWebhook) | **Delete** /webhooks/{resourceId} | Delete a webhook by ID.
[**GetWebhook**](WebhooksApi.md#GetWebhook) | **Get** /webhooks/{resourceId} | Get a webhook by ID.
[**GetWebhooks**](WebhooksApi.md#GetWebhooks) | **Get** /webhooks | Fetch a list of all webhooks.
[**PatchWebhook**](WebhooksApi.md#PatchWebhook) | **Patch** /webhooks/{resourceId} | Modify a webhook by ID.
[**PostWebhook**](WebhooksApi.md#PostWebhook) | **Post** /webhooks | Create a webhook.


# **DeleteWebhook**
> DeleteWebhook(ctx, resourceId)
Delete a webhook by ID.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **resourceId** | **string**| The resource ID. | 

### Return type

 (empty response body)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetWebhook**
> Webhook GetWebhook(ctx, resourceId)
Get a webhook by ID.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **resourceId** | **string**| The resource ID. | 

### Return type

[**Webhook**](Webhook.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetWebhooks**
> Webhooks GetWebhooks(ctx, )
Fetch a list of all webhooks.

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**Webhooks**](Webhooks.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PatchWebhook**
> Webhook PatchWebhook(ctx, resourceId, patchDelta)
Modify a webhook by ID.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **resourceId** | **string**| The resource ID. | 
  **patchDelta** | [**[]PatchOperation**](PatchOperation.md)| Requires a JSON Patch representation of the desired changes to the project. &#39;http://jsonpatch.com/&#39; | 

### Return type

[**Webhook**](Webhook.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PostWebhook**
> Webhook PostWebhook(ctx, webhookBody)
Create a webhook.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **webhookBody** | [**WebhookBody**](WebhookBody.md)| New webhook. | 

### Return type

[**Webhook**](Webhook.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

