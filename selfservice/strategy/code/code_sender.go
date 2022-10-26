package code

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/ory/kratos/courier/template/sms"
	"github.com/pkg/errors"

	"github.com/ory/herodot"
	"github.com/ory/kratos/courier/template/email"

	"github.com/ory/x/errorsx"
	"github.com/ory/x/httpx"
	"github.com/ory/x/sqlcon"
	"github.com/ory/x/stringsx"
	"github.com/ory/x/urlx"

	"github.com/ory/kratos/courier"
	"github.com/ory/kratos/driver/config"
	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/selfservice/flow/recovery"
	"github.com/ory/kratos/selfservice/flow/verification"
	"github.com/ory/kratos/x"
)

type (
	senderDependencies interface {
		courier.Provider
		courier.ConfigProvider

		identity.PoolProvider
		identity.ManagementProvider
		identity.PrivilegedPoolProvider
		x.LoggingProvider
		config.Provider

		RecoveryCodePersistenceProvider
		VerificationCodePersistenceProvider

		HTTPClient(ctx context.Context, opts ...httpx.ResilientOptions) *retryablehttp.Client
	}
	CodeSenderProvider interface {
		CodeSender() *CodeSender
	}

	CodeSender struct {
		deps senderDependencies
	}
)

var ErrUnknownAddress = herodot.ErrNotFound.WithReason("recovery requested for unknown address")

func NewSender(deps senderDependencies) *CodeSender {
	return &CodeSender{deps: deps}
}

// SendRecoveryCode sends a recovery code to the specified address.
// If the address does not exist in the store, an email is still being sent to prevent account
// enumeration attacks. In that case, this function returns the ErrUnknownAddress error.
func (s *CodeSender) SendRecoveryCode(ctx context.Context, r *http.Request, f *recovery.Flow, via identity.VerifiableAddressType, to string) error {
	s.deps.Logger().
		WithField("via", via).
		WithSensitiveField("address", to).
		Debug("Preparing recovery code.")

	address, err := s.deps.IdentityPool().FindRecoveryAddressByValue(ctx, identity.RecoveryAddressTypeEmail, to)
	if err != nil {
		if err := s.send(ctx, string(via), email.NewRecoveryCodeInvalid(s.deps, &email.RecoveryCodeInvalidModel{To: to})); err != nil {
			return err
		}
		return ErrUnknownAddress
	}

	// Get the identity associated with the recovery address
	i, err := s.deps.IdentityPool().GetIdentity(ctx, address.IdentityID)
	if err != nil {
		return err
	}

	rawCode := GenerateRecoveryCode()

	var code *RecoveryCode
	if code, err = s.deps.
		RecoveryCodePersister().
		CreateRecoveryCode(ctx, &CreateRecoveryCodeParams{
			RawCode:         rawCode,
			CodeType:        RecoveryCodeTypeSelfService,
			ExpiresIn:       s.deps.Config().SelfServiceCodeMethodLifespan(r.Context()),
			RecoveryAddress: address,
			FlowID:          f.ID,
			IdentityID:      i.ID,
		}); err != nil {
		return err
	}

	return s.SendRecoveryCodeTo(ctx, i, rawCode, code)
}

func (s *CodeSender) SendRecoveryCodeTo(ctx context.Context, i *identity.Identity, codeString string, code *RecoveryCode) error {
	s.deps.Audit().
		WithField("via", code.RecoveryAddress.Via).
		WithField("identity_id", code.RecoveryAddress.IdentityID).
		WithField("recovery_code_id", code.ID).
		WithSensitiveField("email_address", code.RecoveryAddress.Value).
		WithSensitiveField("recovery_code", codeString).
		Info("Sending out recovery email with recovery code.")

	model, err := x.StructToMap(i)
	if err != nil {
		return err
	}

	emailModel := email.RecoveryCodeValidModel{
		To:           code.RecoveryAddress.Value,
		RecoveryCode: codeString,
		Identity:     model,
	}

	return s.send(ctx, string(code.RecoveryAddress.Via), email.NewRecoveryCodeValid(s.deps, &emailModel))
}

// SendVerificationCode sends a verification link to the specified address. If the address does not exist in the store, an email is
// still being sent to prevent account enumeration attacks. In that case, this function returns the ErrUnknownAddress
// error.
func (s *CodeSender) SendVerificationCode(ctx context.Context, f *verification.Flow, via identity.VerifiableAddressType, to string) error {
	s.deps.Logger().
		WithField("via", via).
		WithSensitiveField("address", to).
		Debug("Preparing verification code.")

	address, err := s.deps.IdentityPool().FindVerifiableAddressByValue(ctx, via, to)
	if err != nil {
		if errorsx.Cause(err) == sqlcon.ErrNoRows {
			if via == identity.VerifiableAddressTypeEmail {
				s.deps.Audit().
					WithField("via", via).
					WithSensitiveField("email_address", address).
					Info("Sending out invalid verification email because address is unknown.")
				if err := s.send(ctx, string(via), email.NewVerificationInvalid(s.deps, &email.VerificationInvalidModel{To: to})); err != nil {
					return err
				}
			}
			return errors.Cause(ErrUnknownAddress)
		}
		return err
	}

	rawCode := GenerateRecoveryCode()
	var code *VerificationCode
	if code, err = s.deps.VerificationCodePersister().CreateVerificationCode(ctx, &CreateVerificationCodeParams{
		RawCode:           rawCode,
		ExpiresIn:         s.deps.Config().SelfServiceCodeMethodLifespan(ctx),
		VerifiableAddress: address,
		FlowID:            f.ID,
	}); err != nil {
		return err
	}

	// Get the identity associated with the recovery address
	i, err := s.deps.IdentityPool().GetIdentity(ctx, address.IdentityID)
	if err != nil {
		return err
	}

	if err := s.SendVerificationCodeTo(ctx, f, i, rawCode, code); err != nil {
		return err
	}
	return nil
}

func (s *CodeSender) constructVerificationLink(ctx context.Context, fID uuid.UUID, codeStr string) string {
	return urlx.CopyWithQuery(
		urlx.AppendPaths(s.deps.Config().SelfServiceLinkMethodBaseURL(ctx), verification.RouteSubmitFlow),
		url.Values{
			"flow": {fID.String()},
			"code": {codeStr},
		}).String()
}

func (s *CodeSender) SendVerificationCodeTo(ctx context.Context, f *verification.Flow, i *identity.Identity, codeString string, code *VerificationCode) error {
	switch code.VerifiableAddress.Via {
	case identity.VerifiableAddressTypeEmail:
		return s.sendVerificationCodeEmailTo(ctx, f, i, codeString, code)

	case identity.VerifiableAddressTypePhone:
		return s.sendVerificationCodeSMSTo(ctx, i, codeString, code)

	default:
		return ErrUnknownAddress
	}
}

func (s *CodeSender) sendVerificationCodeEmailTo(ctx context.Context, f *verification.Flow, i *identity.Identity, codeString string, code *VerificationCode) error {
	s.deps.Audit().
		WithField("via", code.VerifiableAddress.Via).
		WithField("identity_id", i.ID).
		WithField("verification_code_id", code.ID).
		WithSensitiveField("address", code.VerifiableAddress.Value).
		WithSensitiveField("verification_code", codeString).
		Info("Sending out verification email with verification code.")

	model, err := x.StructToMap(i)
	if err != nil {
		return err
	}

	if err := s.send(ctx, string(code.VerifiableAddress.Via), email.NewVerificationCodeValid(s.deps,
		&email.VerificationCodeValidModel{
			To:               code.VerifiableAddress.Value,
			VerificationURL:  s.constructVerificationLink(ctx, f.ID, codeString),
			Identity:         model,
			VerificationCode: codeString,
		})); err != nil {
		return err
	}
	code.VerifiableAddress.Status = identity.VerifiableAddressStatusSent
	return s.deps.PrivilegedIdentityPool().UpdateVerifiableAddress(ctx, code.VerifiableAddress)
}

func (s *CodeSender) sendVerificationCodeSMSTo(ctx context.Context, i *identity.Identity, codeString string, code *VerificationCode) error {
	s.deps.Audit().
		WithField("via", code.VerifiableAddress.Via).
		WithField("identity_id", i.ID).
		WithField("verification_code_id", code.ID).
		WithSensitiveField("address", code.VerifiableAddress.Value).
		WithSensitiveField("verification_code", codeString).
		Info("Sending out verification SMS with verification code.")

	model, err := x.StructToMap(i)
	if err != nil {
		return err
	}

	if err := s.send(ctx, string(code.VerifiableAddress.Via), sms.NewOTPMessage(s.deps,
		&sms.OTPMessageModel{
			To:       code.VerifiableAddress.Value,
			Identity: model,
			Code:     codeString,
		})); err != nil {
		return err
	}
	code.VerifiableAddress.Status = identity.VerifiableAddressStatusSent
	if err := s.deps.PrivilegedIdentityPool().UpdateVerifiableAddress(ctx, code.VerifiableAddress); err != nil {
		return err
	}
	return nil
}

func (s *CodeSender) send(ctx context.Context, via string, t any) error {
	switch f := stringsx.SwitchExact(via); {
	case f.AddCase(identity.AddressTypeEmail):
		_, err := s.deps.Courier(ctx).QueueEmail(ctx, t.(courier.EmailTemplate))
		return err
	case f.AddCase(identity.AddressTypePhone):
		_, err := s.deps.Courier(ctx).QueueSMS(ctx, t.(courier.SMSTemplate))
		return err
	default:
		return f.ToUnknownCaseErr()
	}
}
