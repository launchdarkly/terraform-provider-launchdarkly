# FeatureFlagStatus

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Links** | [***Links**](Links.md) |  | [optional] [default to null]
**Name** | **string** | | Name     | Description | | --------:| ----------- | | new      | the feature flag was created within the last 7 days, and has not been requested yet | | active   | the feature flag was requested by your servers or clients within the last 7 days | | inactive | the feature flag was created more than 7 days ago, and hasn&#39;t been requested by your servers or clients within the past 7 days | | launched | one variation of the feature flag has been rolled out to all your users for at least 7 days |  | [optional] [default to null]
**LastRequested** | **string** |  | [optional] [default to null]
**Default_** | [***interface{}**](interface{}.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


