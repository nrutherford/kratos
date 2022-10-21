package email

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/ory/kratos/courier/template"
)

type (
	VerificationCodeInvalid struct {
		deps  template.Dependencies
		model *VerificationCodeInvalidModel
	}
	VerificationCodeInvalidModel struct {
		To string
	}
)

func NewVerificationCodeInvalid(d template.Dependencies, m *VerificationCodeInvalidModel) *VerificationCodeInvalid {
	return &VerificationCodeInvalid{deps: d, model: m}
}

func (t *VerificationCodeInvalid) EmailRecipient() (string, error) {
	return t.model.To, nil
}

func (t *VerificationCodeInvalid) EmailSubject(ctx context.Context) (string, error) {
	filesystem := os.DirFS(t.deps.CourierConfig().CourierTemplatesRoot(ctx))
	remoteURL := t.deps.CourierConfig().CourierTemplatesVerificationCodeInvalid(ctx).Subject

	subject, err := template.LoadText(ctx, t.deps, filesystem, "verification_code/invalid/email.subject.gotmpl", "verification_code/invalid/email.subject*", t.model, remoteURL)

	return strings.TrimSpace(subject), err
}

func (t *VerificationCodeInvalid) EmailBody(ctx context.Context) (string, error) {
	return template.LoadHTML(ctx, t.deps, os.DirFS(t.deps.CourierConfig().CourierTemplatesRoot(ctx)), "verification_code/invalid/email.body.gotmpl", "verification_code/invalid/email.body*", t.model, t.deps.CourierConfig().CourierTemplatesVerificationCodeInvalid(ctx).Body.HTML)
}

func (t *VerificationCodeInvalid) EmailBodyPlaintext(ctx context.Context) (string, error) {
	return template.LoadText(ctx, t.deps, os.DirFS(t.deps.CourierConfig().CourierTemplatesRoot(ctx)), "verification_code/invalid/email.body.plaintext.gotmpl", "verification_code/invalid/email.body.plaintext*", t.model, t.deps.CourierConfig().CourierTemplatesVerificationCodeInvalid(ctx).Body.PlainText)
}

func (t *VerificationCodeInvalid) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.model)
}
