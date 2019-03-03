# UserFlagSetting

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Links** | [***Links**](Links.md) |  | [optional] [default to null]
**Value** | **bool** | The most important attribute in the response. The _value is the current setting for the user. For a boolean feature toggle, this will be true, false, or null if there is no defined fallthrough value. | [optional] [default to null]
**Setting** | **bool** | The setting attribute indicates whether you&#39;ve explicitly targeted this user to receive a particular variation. For example, if you have explicitly turned off a feature toggle for a user, setting will be false. A setting of null means that you haven&#39;t assigned that user to a specific variation. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


