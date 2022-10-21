# SubmitSelfServiceVerificationFlowWithCodeMethodBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Code** | Pointer to **string** | Code from verification email  Sent to the user once a verification has been initiated and is used to prove that the user is in possession of the email | [optional] 
**CsrfToken** | Pointer to **string** | Sending the anti-csrf token is only required for browser login flows. | [optional] 
**Email** | **string** | Email to Verify  Needs to be set when initiating the flow. If the email is a registered verification email, a verification code will be sent. If the email is not known, an email with details on what happened will be sent instead.  format: email | 
**Method** | **string** | Method supports &#x60;link&#x60; and &#x60;code&#x60; only right now. | 

## Methods

### NewSubmitSelfServiceVerificationFlowWithCodeMethodBody

`func NewSubmitSelfServiceVerificationFlowWithCodeMethodBody(email string, method string, ) *SubmitSelfServiceVerificationFlowWithCodeMethodBody`

NewSubmitSelfServiceVerificationFlowWithCodeMethodBody instantiates a new SubmitSelfServiceVerificationFlowWithCodeMethodBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSubmitSelfServiceVerificationFlowWithCodeMethodBodyWithDefaults

`func NewSubmitSelfServiceVerificationFlowWithCodeMethodBodyWithDefaults() *SubmitSelfServiceVerificationFlowWithCodeMethodBody`

NewSubmitSelfServiceVerificationFlowWithCodeMethodBodyWithDefaults instantiates a new SubmitSelfServiceVerificationFlowWithCodeMethodBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCode

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) GetCode() string`

GetCode returns the Code field if non-nil, zero value otherwise.

### GetCodeOk

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) GetCodeOk() (*string, bool)`

GetCodeOk returns a tuple with the Code field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCode

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) SetCode(v string)`

SetCode sets Code field to given value.

### HasCode

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) HasCode() bool`

HasCode returns a boolean if a field has been set.

### GetCsrfToken

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) GetCsrfToken() string`

GetCsrfToken returns the CsrfToken field if non-nil, zero value otherwise.

### GetCsrfTokenOk

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) GetCsrfTokenOk() (*string, bool)`

GetCsrfTokenOk returns a tuple with the CsrfToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCsrfToken

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) SetCsrfToken(v string)`

SetCsrfToken sets CsrfToken field to given value.

### HasCsrfToken

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) HasCsrfToken() bool`

HasCsrfToken returns a boolean if a field has been set.

### GetEmail

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) SetEmail(v string)`

SetEmail sets Email field to given value.


### GetMethod

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) GetMethod() string`

GetMethod returns the Method field if non-nil, zero value otherwise.

### GetMethodOk

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) GetMethodOk() (*string, bool)`

GetMethodOk returns a tuple with the Method field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMethod

`func (o *SubmitSelfServiceVerificationFlowWithCodeMethodBody) SetMethod(v string)`

SetMethod sets Method field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


