# FeatureFlagBody

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | A human-friendly name for the feature flag. Remember to note if this flag is intended to be temporary or permanent. | [default to null]
**Key** | **string** | A unique key that will be used to reference the flag in your code. | [default to null]
**Description** | **string** | A description of the feature flag. | [optional] [default to null]
**Variations** | [**[]Variation**](Variation.md) | An array of possible variations for the flag. | [default to null]
**Temporary** | **bool** | Whether or not the flag is a temporary flag. | [optional] [default to null]
**Tags** | **[]string** | Tags for the feature flag. | [optional] [default to null]
**IncludeInSnippet** | **bool** | Whether or not this flag should be made available to the client-side JavaScript SDK. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


