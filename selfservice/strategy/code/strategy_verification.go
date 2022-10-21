package code

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/ory/x/decoderx"
	"github.com/ory/x/urlx"

	"github.com/ory/kratos/selfservice/flow"
	"github.com/ory/kratos/selfservice/flow/verification"
	"github.com/ory/kratos/text"
	"github.com/ory/kratos/ui/node"
	"github.com/ory/kratos/x"
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
		node.NewInputField("email", nil, node.CodeGroup, node.InputAttributeTypeEmail, node.WithRequiredInputAttribute).
			WithMetaLabel(text.NewInfoNodeInputEmail()),
	)
	f.UI.
		GetNodes().
		Append(node.NewInputField("method", s.VerificationStrategyID(), node.CodeGroup, node.InputAttributeTypeSubmit).
			WithMetaLabel(text.NewInfoNodeLabelSubmit()))

	return nil
}

// swagger:model submitSelfServiceVerificationFlowWithCodeMethodBody
// nolint:deadcode,unused
type submitSelfServiceVerificationFlowWithCodeMethodBody struct {
	// Email to Verify
	//
	// Needs to be set when initiating the flow. If the email is a registered
	// verification email, a verification code will be sent. If the email is not known,
	// an email with details on what happened will be sent instead.
	//
	// format: email
	// required: true
	Email string `json:"email" form:"email"`

	// Code from verification email
	//
	// Sent to the user once a verification has been initiated and is used to prove
	// that the user is in possession of the email
	//
	// required: false
	Code string `json:"code" form:"code"`

	// Sending the anti-csrf token is only required for browser login flows.
	CSRFToken string `form:"csrf_token" json:"csrf_token"`

	// Method supports `link` and `code` only right now.
	//
	// enum:
	// - link
	// - code
	// required: true
	Method string `json:"method"`
}

func (s *Strategy) Verify(w http.ResponseWriter, r *http.Request, f *verification.Flow) (err error) {
	if !s.isVerificationCodeFlow(f) {
		return errors.WithStack(flow.ErrStrategyNotResponsible)
	}

	body, err := s.decodeVerification(r)
	if err != nil {
		return s.handleVerificationError(w, r, nil, body, err)
	}
	ctx := r.Context()

	// If a CSRF violation occurs the flow is most likely FUBAR, as the user either lost the CSRF token, or an attack occurred.
	// In this case, we just issue a new flow and "abandon" the old flow.
	if err := flow.EnsureCSRF(s.deps, r, f.Type, s.deps.Config().DisableAPIFlowEnforcement(ctx), s.deps.GenerateCSRFToken, body.CSRFToken); err != nil {
		return s.retryVerificationFlowWithError(w, r, flow.TypeBrowser, err)
	}

	return nil
}

func (s *Strategy) isVerificationCodeFlow(f *verification.Flow) bool {
	value, err := f.Active.Value()
	if err != nil {
		return false
	}
	return value == s.VerificationNodeGroup().String()
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
		decoderx.HTTPDecoderAllowedMethods("POST"),
		decoderx.HTTPDecoderSetValidatePayloads(true),
		decoderx.HTTPDecoderJSONFollowsFormFormat(),
	); err != nil {
		return nil, errors.WithStack(err)
	}

	return &body, nil
}

func (s *Strategy) handleVerificationError(w http.ResponseWriter, r *http.Request, flow *verification.Flow, body *verificationSubmitPayload, err error) error {
	if flow != nil {
		email := ""
		if body != nil {
			email = body.Email
		}

		flow.UI.SetCSRF(s.deps.GenerateCSRFToken(r))
		flow.UI.GetNodes().Upsert(
			node.NewInputField("email", email, node.CodeGroup, node.InputAttributeTypeEmail, node.WithRequiredInputAttribute).
				WithMetaLabel(text.NewInfoNodeInputEmail()),
		)
	}

	return err
}

func (s *Strategy) retryVerificationFlowWithMessage(w http.ResponseWriter, r *http.Request, ft flow.Type, message *text.Message) error {
	s.deps.Logger().
		WithRequest(r).
		WithField("message", message).
		Debug("A verification flow is being retried because a validation error occurred.")

	ctx := r.Context()
	config := s.deps.Config()

	f, err := verification.NewFlow(config, config.SelfServiceFlowVerificationRequestLifespan(ctx), s.deps.CSRFHandler().RegenerateToken(w, r), r, verification.Strategies{s}, ft)
	if err != nil {
		return err
	}

	f.UI.Messages.Add(message)
	if err := s.deps.VerificationFlowPersister().CreateVerificationFlow(ctx, f); err != nil {
		return err
	}

	if x.IsJSONRequest(r) {
		http.Redirect(w, r, urlx.CopyWithQuery(urlx.AppendPaths(config.SelfPublicURL(ctx),
			verification.RouteGetFlow), url.Values{"id": {f.ID.String()}}).String(), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, f.AppendTo(config.SelfServiceFlowVerificationUI(ctx)).String(), http.StatusSeeOther)
	}

	return errors.WithStack(flow.ErrCompletedByStrategy)
}

func (s *Strategy) retryVerificationFlowWithError(w http.ResponseWriter, r *http.Request, ft flow.Type, recErr error) error {
	s.deps.Logger().
		WithRequest(r).
		WithError(recErr).
		Debug("A verification flow is being retried because a validation error occurred.")

	ctx := r.Context()
	config := s.deps.Config()

	if expired := new(flow.ExpiredError); errors.As(recErr, &expired) {
		return s.retryVerificationFlowWithMessage(w, r, ft, text.NewErrorValidationVerificationFlowExpired(expired.Ago))
	}

	f, err := verification.NewFlow(config, config.SelfServiceFlowVerificationRequestLifespan(ctx), s.deps.CSRFHandler().RegenerateToken(w, r), r, verification.Strategies{s}, ft)
	if err != nil {
		return err
	}
	if err := f.UI.ParseError(node.CodeGroup, recErr); err != nil {
		return err
	}
	if err := s.deps.VerificationFlowPersister().CreateVerificationFlow(ctx, f); err != nil {
		return err
	}

	if x.IsJSONRequest(r) {
		http.Redirect(w, r, urlx.CopyWithQuery(urlx.AppendPaths(config.SelfPublicURL(ctx),
			verification.RouteGetFlow), url.Values{"id": {f.ID.String()}}).String(), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, f.AppendTo(config.SelfServiceFlowVerificationUI(ctx)).String(), http.StatusSeeOther)
	}

	return errors.WithStack(flow.ErrCompletedByStrategy)
}
