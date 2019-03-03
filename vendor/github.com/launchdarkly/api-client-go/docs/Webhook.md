# Webhook

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Links** | [***Links**](Links.md) |  | [optional] [default to null]
**Id** | **string** | The unique resource id. | [optional] [default to null]
**Url** | **string** | The URL of the remote webhook. | [optional] [default to null]
**Secret** | **string** | If defined, the webhooks post request will include a X-LD-Signature header whose value will contain an HMAC SHA256 hex digest of the webhook payload, using the secret as the key. | [optional] [default to null]
**On** | **bool** | Whether this webhook is enabled or not. | [optional] [default to null]
**Name** | **string** | The name of the webhook. | [optional] [default to null]
**Tags** | **[]string** | Tags assigned to this webhook. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


