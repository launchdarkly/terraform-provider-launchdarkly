# UserSegment

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | **string** | Unique identifier for the user segment. | [default to null]
**Name** | **string** | Name of the user segment. | [default to null]
**Description** | **string** | Description of the user segment. | [optional] [default to null]
**Tags** | **[]string** | An array of tags for this user segment. | [optional] [default to null]
**CreationDate** | **float32** | A unix epoch time in milliseconds specifying the creation time of this flag. | [default to null]
**Included** | **[]string** | An array of user keys that are included in this segment. | [optional] [default to null]
**Excluded** | **[]string** | An array of user keys that should not be included in this segment, unless they are also listed in \&quot;included\&quot;. | [optional] [default to null]
**Rules** | [**[]UserSegmentRule**](UserSegmentRule.md) | An array of rules that can cause a user to be included in this segment. | [optional] [default to null]
**Version** | **int32** |  | [optional] [default to null]
**Links** | [***Links**](Links.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


