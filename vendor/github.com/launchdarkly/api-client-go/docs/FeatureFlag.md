# FeatureFlag

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | **string** |  | [optional] [default to null]
**Name** | **string** | Name of the feature flag. | [optional] [default to null]
**Description** | **string** | Description of the feature flag. | [optional] [default to null]
**Kind** | **string** | Whether the feature flag is a boolean flag or multivariate. | [optional] [default to null]
**CreationDate** | **float32** | A unix epoch time in milliseconds specifying the creation time of this flag. | [optional] [default to null]
**IncludeInSnippet** | **bool** |  | [optional] [default to null]
**Temporary** | **bool** | Whether or not this flag is temporary. | [optional] [default to null]
**MaintainerId** | **string** | The ID of the member that should maintain this flag. | [optional] [default to null]
**Tags** | **[]string** | An array of tags for this feature flag. | [optional] [default to null]
**Variations** | [**[]Variation**](Variation.md) | The variations for this feature flag. | [optional] [default to null]
**GoalIds** | **[]string** | An array goals from all environments associated with this feature flag | [optional] [default to null]
**Version** | **int32** |  | [optional] [default to null]
**CustomProperties** | [**map[string]CustomProperty**](CustomProperty.md) | A mapping of keys to CustomProperty entries. | [optional] [default to null]
**Links** | [***Links**](Links.md) |  | [optional] [default to null]
**Maintainer** | [***Member**](Member.md) |  | [optional] [default to null]
**Environments** | [**map[string]FeatureFlagConfig**](FeatureFlagConfig.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


