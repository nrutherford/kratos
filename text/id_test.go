package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIDs(t *testing.T) {
	assert.Equal(t, 1010000, int(InfoSelfServiceLoginRoot))

	assert.Equal(t, 1020000, int(InfoSelfServiceLogout))

	assert.Equal(t, 1030000, int(InfoSelfServiceMFA))

	assert.Equal(t, 1040000, int(InfoSelfServiceRegistrationRoot))
	assert.Equal(t, 1040001, int(InfoSelfServiceRegistration))

	assert.Equal(t, 1050000, int(InfoSelfServiceSettings))
	assert.Equal(t, 1050001, int(InfoSelfServiceSettingsUpdateSuccess))

	assert.Equal(t, 1060000, int(InfoSelfServiceRecovery))
	assert.Equal(t, 1060001, int(InfoSelfServiceRecoverySuccessful))
	assert.Equal(t, 1060002, int(InfoSelfServiceRecoveryEmailSent))
	assert.Equal(t, 1060003, int(InfoSelfServiceRecoveryEmailWithCodeSent))

	assert.Equal(t, 1070000, int(InfoNodeLabel))

	assert.Equal(t, 1080000, int(InfoSelfServiceVerification))
	assert.Equal(t, 1080001, int(InfoSelfServiceVerificationEmailSent))
	assert.Equal(t, 1080002, int(InfoSelfServiceVerificationSuccessful))
	assert.Equal(t, 1080003, int(InfoSelfServiceVerificationEmailWithCodeSent))

	assert.Equal(t, 4000000, int(ErrorValidation))
	assert.Equal(t, 4000001, int(ErrorValidationGeneric))
	assert.Equal(t, 4000002, int(ErrorValidationRequired))

	assert.Equal(t, 4010000, int(ErrorValidationLogin))
	assert.Equal(t, 4010001, int(ErrorValidationLoginFlowExpired))

	assert.Equal(t, 4040000, int(ErrorValidationRegistration))
	assert.Equal(t, 4040001, int(ErrorValidationRegistrationFlowExpired))

	assert.Equal(t, 4050000, int(ErrorValidationSettings))
	assert.Equal(t, 4050001, int(ErrorValidationSettingsFlowExpired))

	assert.Equal(t, 4060000, int(ErrorValidationRecovery))
	assert.Equal(t, 4060001, int(ErrorValidationRecoveryRetrySuccess))
	assert.Equal(t, 4060002, int(ErrorValidationRecoveryStateFailure))
	assert.Equal(t, 4060004, int(ErrorValidationRecoveryTokenInvalidOrAlreadyUsed))
	assert.Equal(t, 4060005, int(ErrorValidationRecoveryFlowExpired))
	assert.Equal(t, 4060006, int(ErrorValidationRecoveryCodeInvalidOrAlreadyUsed))

	assert.Equal(t, 4070000, int(ErrorValidationVerification))
	assert.Equal(t, 4070001, int(ErrorValidationVerificationTokenInvalidOrAlreadyUsed))
	assert.Equal(t, 4070002, int(ErrorValidationVerificationRetrySuccess))
	assert.Equal(t, 4070003, int(ErrorValidationVerificationStateFailure))
	assert.Equal(t, 4070005, int(ErrorValidationVerificationFlowExpired))
	assert.Equal(t, 4070006, int(ErrorValidationVerificationCodeInvalidOrAlreadyUsed))

	assert.Equal(t, 5000000, int(ErrorSystem))
}
