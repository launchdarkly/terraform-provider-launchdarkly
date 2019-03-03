# \AuditLogApi

All URIs are relative to *https://app.launchdarkly.com/api/v2*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetAuditLogEntries**](AuditLogApi.md#GetAuditLogEntries) | **Get** /auditlog | Get a list of all audit log entries. The query parameters allow you to restrict the returned results by date ranges, resource specifiers, or a full-text search query.
[**GetAuditLogEntry**](AuditLogApi.md#GetAuditLogEntry) | **Get** /auditlog/{resourceId} | Use this endpoint to fetch a single audit log entry by its resouce ID.


# **GetAuditLogEntries**
> AuditLogEntries GetAuditLogEntries(ctx, optional)
Get a list of all audit log entries. The query parameters allow you to restrict the returned results by date ranges, resource specifiers, or a full-text search query.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***GetAuditLogEntriesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetAuditLogEntriesOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **before** | **optional.Float32**| A timestamp filter, expressed as a Unix epoch time in milliseconds. All entries returned will have before this timestamp. | 
 **after** | **optional.Float32**| A timestamp filter, expressed as a Unix epoch time in milliseconds. All entries returned will have occured after this timestamp. | 
 **q** | **optional.String**| Text to search for. You can search for the full or partial name of the resource involved or fullpartial email address of the member who made the change. | 
 **limit** | **optional.Float32**| A limit on the number of audit log entries to be returned, between 1 and 20. | 
 **spec** | **optional.String**| A resource specifier, allowing you to filter audit log listings by resource. | 

### Return type

[**AuditLogEntries**](AuditLogEntries.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetAuditLogEntry**
> AuditLogEntry GetAuditLogEntry(ctx, resourceId)
Use this endpoint to fetch a single audit log entry by its resouce ID.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **resourceId** | **string**| The resource ID. | 

### Return type

[**AuditLogEntry**](AuditLogEntry.md)

### Authorization

[Token](../README.md#Token)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

