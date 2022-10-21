package email

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/ory/kratos/courier/template"
)

type (
	VerificationCodeValid struct {
		deps  template.Dependencies
		model *VerificationCodeValidModel
	}
	VerificationCodeValidModel struct {
		To               string
		VerificationCode string
		Identity         map[string]interface{}
	}
)

func NewVerificationCodeValid(d template.Dependencies, m *VerificationCodeValidModel) *VerificationCodeValid {
	return &VerificationCodeValid{deps: d, model: m}
}

func (t *VerificationCodeValid) EmailRecipient() (string, error) {
	return t.model.To, nil
}

func (t *VerificationCodeValid) EmailSubject(ctx context.Context) (string, error) {
	subject, err := template.LoadText(ctx, t.deps, os.DirFS(t.deps.CourierConfig().CourierTemplatesRoot(ctx)), "verification_code/valid/email.subject.gotmpl", "verification_code/valid/email.subject*", t.model, t.deps.CourierConfig().CourierTemplatesVerificationCodeValid(ctx).Subject)

	return strings.TrimSpace(subject), err
}

func (t *VerificationCodeValid) EmailBody(ctx context.Context) (string, error) {
	return template.LoadHTML(ctx, t.deps, os.DirFS(t.deps.CourierConfig().CourierTemplatesRoot(ctx)), "verification_code/valid/email.body.gotmpl", "verification_code/valid/email.body*", t.model, t.deps.CourierConfig().CourierTemplatesVerificationCodeValid(ctx).Body.HTML)
}

func (t *VerificationCodeValid) EmailBodyPlaintext(ctx context.Context) (string, error) {
	return template.LoadText(ctx, t.deps, os.DirFS(t.deps.CourierConfig().CourierTemplatesRoot(ctx)), "verification_code/valid/email.body.plaintext.gotmpl", "verification_code/valid/email.body.plaintext*", t.model, t.deps.CourierConfig().CourierTemplatesVerificationCodeValid(ctx).Body.PlainText)
}

func (t *VerificationCodeValid) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.model)
}
