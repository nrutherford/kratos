package text

func NewInfoNodeLabelVerifyOTP() *Message {
	return &Message{
		ID:   InfoNodeLabelVerifyOTP,
		Text: "Verify code",
		Type: Info,
	}
}

func NewInfoNodeInputPassword() *Message {
	return &Message{
		ID:   InfoNodeLabelInputPassword,
		Text: "Password",
		Type: Info,
	}
}

func NewInfoNodeLabelGenerated(title string) *Message {
	return &Message{
		ID:   InfoNodeLabelGenerated,
		Text: title,
		Type: Info,
	}
}

func NewInfoNodeLabelSave() *Message {
	return &Message{
		ID:   InfoNodeLabelSave,
		Text: "Save",
		Type: Info,
	}
}

func NewInfoNodeLabelSubmit() *Message {
	return &Message{
		ID:   InfoNodeLabelSubmit,
		Text: "Submit",
		Type: Info,
	}
}

func NewInfoNodeLabelReturn() *Message {
	return &Message{
		ID:   InfoNodeLabelReturn,
		Text: "Return",
		Type: Info,
	}
}

func NewInfoNodeLabelID() *Message {
	return &Message{
		ID:   InfoNodeLabelID,
		Text: "ID",
		Type: Info,
	}
}

func NewInfoNodeInputEmail() *Message {
	return &Message{
		ID:   InfoNodeLabelEmail,
		Text: "Email",
		Type: Info,
	}
}

func NewInfoNodeResendOTP() *Message {
	return &Message{
		ID:   InfoNodeLabelEmail,
		Text: "Resend code",
		Type: Info,
	}
}
