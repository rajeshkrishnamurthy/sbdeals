package suppliers

// Supplier captures the supplier fields required by Sprint 1 Feature 01.
type Supplier struct {
	ID       int
	Name     string
	WhatsApp string
	Location string
	Notes    string
}

// Input is the editable supplier payload for create/update forms.
type Input struct {
	Name     string
	WhatsApp string
	Location string
	Notes    string
}
