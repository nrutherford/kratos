package code

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/schema"
	"github.com/ory/kratos/selfservice/flow"
	"github.com/ory/kratos/selfservice/flow/verification"
	"github.com/ory/kratos/text"
	"github.com/ory/kratos/ui/container"
	"github.com/ory/kratos/ui/node"
	"github.com/ory/kratos/x"
	"github.com/ory/x/decoderx"
	"github.com/ory/x/sqlxx"
	"github.com/ory/x/urlx"
)

func (s *Strategy) VerificationStrategyID() string {
	return verification.StrategyVerificationCodeName
}

func (s *Strategy) RegisterPublicVerificationRoutes(public *x.RouterPublic) {
}

func (s *Strategy) RegisterAdminVerificationRoutes(admin *x.RouterAdmin) {
}

func (s *Strategy) PopulateVerificationMethod(r *http.Request, f *verification.Flow) error {
	f.UI.SetCSRF(s.deps.GenerateCSRFToken(r))
	f.UI.GetNodes().Upsert(
		// v0.5: form.Field{Name: "email", Type: "email", Required: true}
		node.NewInputField("email", nil, node.CodeGroup, node.InputAttributeTypeEmail, node.WithRequiredInputAttribute).WithMetaLabel(text.NewInfoNodeInputEmail()),
	)
	f.UI.GetNodes().Append(node.NewInputField("method", s.VerificationStrategyID(), node.LinkGroup, node.InputAttributeTypeSubmit).WithMetaLabel(text.NewInfoNodeLabelSubmit()))
	return nil
}

type verificationSubmitPayload struct {
	Method    string `json:"method" form:"method"`
	Code      string `json:"code" form:"code"`
	CSRFToken string `json:"csrf_token" form:"csrf_token"`
	Flow      string `json:"flow" form:"flow"`
	Email     string `json:"email" form:"email"`
}

func (s *Strategy) decodeVerification(r *http.Request) (*verificationSubmitPayload, error) {
	var body verificationSubmitPayload

	compiler, err := decoderx.HTTPRawJSONSchemaCompiler(verificationMethodSchema)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := s.dx.Decode(r, &body, compiler,
		decoderx.HTTPDecoderUseQueryAndBody(),
		decoderx.HTTPKeepRequestBody(true),
		decoderx.HTTPDecoderAllowedMethods("POST", "GET"),
		decoderx.HTTPDecoderSetValidatePayloads(true),
		decoderx.HTTPDecoderJSONFollowsFormFormat(),
	); err != nil {
		return nil, errors.WithStack(err)
	}

	return &body, nil
}

// handleVerificationError is a convenience function for handling all types of errors that may occur (e.g. validation error).
func (s *Strategy) handleVerificationError(w http.ResponseWriter, r *http.Request, f *verification.Flow, body *verificationSubmitPayload, err error) error {
	if f != nil {
		f.UI.SetCSRF(s.deps.GenerateCSRFToken(r))
		f.UI.GetNodes().Upsert(
			// v0.5: form.Field{Name: "email", Type: "email", Required: true, Value: body.Body.Email}
			node.NewInputField("email", body.Email, node.LinkGroup, node.InputAttributeTypeEmail, node.WithRequiredInputAttribute).WithMetaLabel(text.NewInfoNodeInputEmail()),
		)
	}

	return err
}

// swagger:model submitSelfServiceVerificationFlowWithLinkMethodBody
// nolint:deadcode,unused
type submitSelfServiceVerificationFlowWithLinkMethodBody struct {
	// Email to Verify
	//
	// Needs to be set when initiating the flow. If the email is a registered
	// verification email, a verification link will be sent. If the email is not known,
	// a email with details on what happened will be sent instead.
	//
	// format: email
	// required: true
	Email string `json:"email"`

	// Sending the anti-csrf token is only required for browser login flows.
	CSRFToken string `form:"csrf_token" json:"csrf_token"`

	// Method is the recovery method
	//
	// enum:
	// - link
	// - code
	// required: true
	Method string `json:"method"`
}

func (s *Strategy) Verify(w http.ResponseWriter, r *http.Request, f *verification.Flow) (err error) {
	body, err := s.decodeVerification(r)
	if err != nil {
		return s.handleVerificationError(w, r, nil, body, err)
	}

	if len(body.Code) > 0 {
		if err := flow.MethodEnabledAndAllowed(r.Context(), s.VerificationStrategyID(), s.VerificationStrategyID(), s.deps); err != nil {
			return s.handleVerificationError(w, r, nil, body, err)
		}

		if r.Method == http.MethodGet {
			return s.handleLinkClick(w, r, f, body.Code)
		}

		return s.verificationUseCode(w, r, body, f)
	}

	if err := flow.MethodEnabledAndAllowed(r.Context(), s.VerificationStrategyID(), body.Method, s.deps); err != nil {
		return s.handleVerificationError(w, r, f, body, err)
	}

	if err := f.Valid(); err != nil {
		return s.handleVerificationError(w, r, f, body, err)
	}

	switch f.State {
	case verification.StateChooseMethod:
		fallthrough
	case verification.StateEmailSent:
		// Do nothing (continue with execution after this switch statement)
		return s.verificationHandleFormSubmission(w, r, f)
	case verification.StatePassedChallenge:
		return s.retryVerificationFlowWithMessage(w, r, f.Type, text.NewErrorValidationVerificationRetrySuccess())
	default:
		return s.retryVerificationFlowWithMessage(w, r, f.Type, text.NewErrorValidationVerificationStateFailure())
	}
}

func (s *Strategy) createVerificationCodeForm(action string, code *string, email *string) *container.Container {
	// re-initialize the UI with a "clean" new state
	c := &container.Container{
		Method: "POST",
		Action: action,
	}

	c.Nodes.Append(
		node.
			NewInputField("code", code, node.CodeGroup, node.InputAttributeTypeNumber, node.WithRequiredInputAttribute).
			WithMetaLabel(text.NewInfoNodeLabelVerifyOTP()),
	)
	c.Nodes.Append(node.NewInputField("method", s.VerificationNodeGroup(), node.CodeGroup, node.InputAttributeTypeHidden))

	c.
		Nodes.
		Append(node.NewInputField("method", s.VerificationStrategyID(), node.CodeGroup, node.InputAttributeTypeSubmit).
			WithMetaLabel(text.NewInfoNodeLabelSubmit()))

	if email != nil {
		c.Nodes.Append(node.NewInputField("email", email, node.CodeGroup, node.InputAttributeTypeSubmit).
			WithMetaLabel(text.NewInfoNodeResendOTP()),
		)
	}

	return c
}

func (s *Strategy) handleLinkClick(w http.ResponseWriter, r *http.Request, f *verification.Flow, code string) error {
	f.UI = s.createVerificationCodeForm(flow.AppendFlowTo(urlx.AppendPaths(s.deps.Config().SelfPublicURL(r.Context()), verification.RouteSubmitFlow), f.ID).String(), &code, nil)

	// In the verification flow, we can't enforce CSRF if the flow is opened from an email
	csrfToken := s.deps.CSRFHandler().RegenerateToken(w, r)
	f.UI.SetCSRF(csrfToken)
	f.CSRFToken = csrfToken

	if err := s.deps.VerificationFlowPersister().UpdateVerificationFlow(r.Context(), f); err != nil {
		return err
	}

	// we always redirect to the browser UI here to allow API flows to complete aswell
	// In the future, we might want to redirect to a custom URI scheme here, to allow to open e.g. an app on the device of
	// the user to handle the flow directly. For now, the browser is good enough, as no session is issued after the
	// verification is done.
	http.Redirect(w, r, f.AppendTo(s.deps.Config().SelfServiceFlowVerificationUI(r.Context())).String(), http.StatusSeeOther)

	return errors.WithStack(flow.ErrCompletedByStrategy)
}

func (s *Strategy) verificationHandleFormSubmission(w http.ResponseWriter, r *http.Request, f *verification.Flow) error {
	body, err := s.decodeVerification(r)
	if err != nil {
		return s.handleVerificationError(w, r, f, body, err)
	}

	if len(body.Email) == 0 {
		return s.handleVerificationError(w, r, f, body, schema.NewRequiredError("#/email", "email"))
	}

	if err := flow.EnsureCSRF(s.deps, r, f.Type, s.deps.Config().DisableAPIFlowEnforcement(r.Context()), s.deps.GenerateCSRFToken, body.CSRFToken); err != nil {
		return s.handleVerificationError(w, r, f, body, err)
	}

	if err := s.deps.CodeSender().SendVerificationCode(r.Context(), f, identity.VerifiableAddressTypeEmail, body.Email); err != nil {
		if !errors.Is(err, ErrUnknownAddress) {
			return s.handleVerificationError(w, r, f, body, err)
		}
		// Continue execution
	}
	f.State = verification.StateEmailSent

	f.UI = s.createVerificationCodeForm(flow.AppendFlowTo(urlx.AppendPaths(s.deps.Config().SelfPublicURL(r.Context()), verification.RouteSubmitFlow), f.ID).String(), nil, &body.Email)
	f.UI.Messages.Set(text.NewVerificationEmailSent())
	f.UI.SetCSRF(s.deps.GenerateCSRFToken(r))

	if err := s.deps.VerificationFlowPersister().UpdateVerificationFlow(r.Context(), f); err != nil {
		return s.handleVerificationError(w, r, f, body, err)
	}

	return nil
}

// nolint:deadcode,unused
// swagger:parameters selfServiceBrowserVerify
type selfServiceBrowserVerifyParameters struct {
	// required: true
	// in: query
	Code string `json:"code"`
}

func (s *Strategy) verificationUseCode(w http.ResponseWriter, r *http.Request, body *verificationSubmitPayload, f *verification.Flow) error {
	code, err := s.deps.VerificationCodePersister().UseVerificationCode(r.Context(), f.ID, body.Code)
	if errors.Is(err, ErrCodeNotFound) {
		f.UI.Messages.Clear()
		f.UI.Messages.Add(text.NewErrorValidationRecoveryCodeInvalidOrAlreadyUsed())
		if err := s.deps.VerificationFlowPersister().UpdateVerificationFlow(r.Context(), f); err != nil {
			return s.retryVerificationFlowWithError(w, r, f.Type, err)
		}

		// No error
		return nil
	} else if err != nil {
		return s.retryVerificationFlowWithError(w, r, f.Type, err)
	}

	if err := code.Valid(); err != nil {
		return s.retryVerificationFlowWithError(w, r, flow.TypeBrowser, err)
	}

	i, err := s.deps.IdentityPool().GetIdentity(r.Context(), code.VerifiableAddress.IdentityID)
	if err != nil {
		return s.retryVerificationFlowWithError(w, r, flow.TypeBrowser, err)
	}

	if err := s.deps.VerificationExecutor().PostVerificationHook(w, r, f, i); err != nil {
		return s.retryVerificationFlowWithError(w, r, flow.TypeBrowser, err)
	}

	address := code.VerifiableAddress
	address.Verified = true
	verifiedAt := sqlxx.NullTime(time.Now().UTC())
	address.VerifiedAt = &verifiedAt
	address.Status = identity.VerifiableAddressStatusCompleted
	if err := s.deps.PrivilegedIdentityPool().UpdateVerifiableAddress(r.Context(), address); err != nil {
		return s.retryVerificationFlowWithError(w, r, flow.TypeBrowser, err)
	}

	returnTo := s.getRedirectURL(r.Context(), f)

	f.UI = &container.Container{
		Method: "GET",
		Action: returnTo.String(),
	}

	f.State = verification.StatePassedChallenge
	// See https://github.com/ory/kratos/issues/1547
	f.SetCSRFToken(flow.GetCSRFToken(s.deps, w, r, f.Type))
	f.UI.Messages.Set(text.NewInfoSelfServiceVerificationSuccessful())
	f.UI.
		Nodes.
		Append(node.NewAnchorField("go-back", returnTo.String(), node.CodeGroup, text.NewInfoNodeLabelReturn()))

	if err := s.deps.VerificationFlowPersister().UpdateVerificationFlow(r.Context(), f); err != nil {
		return s.retryVerificationFlowWithError(w, r, flow.TypeBrowser, err)
	}

	return nil
}

func (s *Strategy) getRedirectURL(ctx context.Context, f *verification.Flow) *url.URL {
	defaultRedirectURL := s.deps.Config().SelfServiceFlowVerificationReturnTo(ctx, f.AppendTo(s.deps.Config().SelfServiceFlowVerificationUI(ctx)))

	verificationRequestURL, err := urlx.Parse(f.GetRequestURL())
	if err != nil {
		s.deps.Logger().Debugf("error parsing verification requestURL, using default redirect url %s: %s\n", defaultRedirectURL.String(), err)
		return defaultRedirectURL
	}

	verificationRequest := http.Request{URL: verificationRequestURL}

	returnTo, err := x.SecureRedirectTo(&verificationRequest, defaultRedirectURL,
		x.SecureRedirectAllowSelfServiceURLs(s.deps.Config().SelfPublicURL(ctx)),
		x.SecureRedirectAllowURLs(s.deps.Config().SelfServiceBrowserAllowedReturnToDomains(ctx)),
	)
	if err != nil {
		s.deps.Logger().Debugf("error parsing redirectTo from verification, using default redirect url %s: %s\n", defaultRedirectURL.String(), err)
		return defaultRedirectURL
	}
	return returnTo
}

func (s *Strategy) retryVerificationFlowWithMessage(w http.ResponseWriter, r *http.Request, ft flow.Type, message *text.Message) error {
	s.deps.
		Logger().
		WithRequest(r).
		WithField("message", message).
		Debug("A verification flow is being retried because a validation error occurred.")

	f, err := verification.NewFlow(s.deps.Config(),
		s.deps.Config().SelfServiceFlowVerificationRequestLifespan(r.Context()), s.deps.CSRFHandler().RegenerateToken(w, r), r, s, ft)
	if err != nil {
		return s.handleVerificationError(w, r, f, nil, err)
	}

	f.UI.Messages.Add(message)

	if err := s.deps.VerificationFlowPersister().CreateVerificationFlow(r.Context(), f); err != nil {
		return s.handleVerificationError(w, r, f, nil, err)
	}

	if x.IsJSONRequest(r) {
		// Use the expired error here, in the future we should probably have a `SelfServiceFlowGoneError` or similar`
		expired := new(flow.ExpiredError)
		s.deps.Writer().WriteError(w, r, expired.WithFlow(f))
	} else {
		http.Redirect(w, r, f.AppendTo(s.deps.Config().SelfServiceFlowVerificationUI(r.Context())).String(), http.StatusSeeOther)
	}

	return errors.WithStack(flow.ErrCompletedByStrategy)
}

func (s *Strategy) retryVerificationFlowWithError(w http.ResponseWriter, r *http.Request, ft flow.Type, verErr error) error {
	s.deps.
		Logger().
		WithRequest(r).
		WithError(verErr).
		Debug("A verification flow is being retried because an error occurred.")

	f, err := verification.NewFlow(s.deps.Config(),
		s.deps.Config().SelfServiceFlowVerificationRequestLifespan(r.Context()), s.deps.CSRFHandler().RegenerateToken(w, r), r, s, ft)
	if err != nil {
		return s.handleVerificationError(w, r, f, nil, err)
	}

	var toReturn error

	if expired := new(flow.ExpiredError); errors.As(verErr, &expired) {
		f.UI.Messages.Add(text.NewErrorValidationVerificationFlowExpired(expired.Ago))
		toReturn = expired.WithFlow(f)
	} else if err := f.UI.ParseError(node.LinkGroup, verErr); err != nil {
		return err
	}

	if err := s.deps.VerificationFlowPersister().CreateVerificationFlow(r.Context(), f); err != nil {
		return s.handleVerificationError(w, r, f, nil, err)
	}

	if x.IsJSONRequest(r) {
		s.deps.Writer().WriteError(w, r, toReturn)
	} else {
		http.Redirect(w, r, f.AppendTo(s.deps.Config().SelfServiceFlowVerificationUI(r.Context())).String(), http.StatusSeeOther)
	}

	return errors.WithStack(flow.ErrCompletedByStrategy)
}

func (s *Strategy) SendVerificationEmail(ctx context.Context, f *verification.Flow, i *identity.Identity, a *identity.VerifiableAddress) (err error) {

	rawCode := GenerateRecoveryCode()

	code, err := s.deps.VerificationCodePersister().CreateVerificationCode(ctx, &CreateVerificationCodeParams{
		RawCode:           rawCode,
		ExpiresIn:         s.deps.Config().SelfServiceCodeMethodLifespan(ctx),
		VerifiableAddress: a,
		FlowID:            f.ID,
	})

	if err != nil {
		return err
	}

	return s.deps.CodeSender().SendVerificationCodeTo(ctx, f, i, rawCode, code)
}
