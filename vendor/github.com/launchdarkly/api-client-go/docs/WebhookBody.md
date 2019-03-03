# WebhookBody

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Url** | **string** | The URL of the remote webhook. | [default to null]
**Secret** | **string** | If sign is true, and the secret attribute is omitted, LaunchDarkly will automatically generate a secret for you. | [optional] [default to null]
**Sign** | **bool** | If sign is false, the webhook will not include a signature header, and the secret can be omitted. | [default to null]
**On** | **bool** | Whether this webhook is enabled or not. | [default to null]
**Name** | **string** | The name of the webhook. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


