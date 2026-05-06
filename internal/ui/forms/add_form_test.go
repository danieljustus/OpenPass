package forms

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAddEntryFormInit(t *testing.T) {
	f := NewAddEntryForm(false)

	if f.focused != fieldUsername {
		t.Errorf("expected focus on username, got %d", f.focused)
	}

	if f.username.Focused() != true {
		t.Error("expected username field to be focused")
	}
}

func TestAddEntryFormTabNavigation(t *testing.T) {
	f := NewAddEntryForm(false)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyTab})
	frm := m.(*AddEntryForm)

	if frm.focused != fieldPassword {
		t.Errorf("expected focus on password after tab, got %d", frm.focused)
	}
}

func TestAddEntryFormShiftTabNavigation(t *testing.T) {
	f := NewAddEntryForm(false)
	f.focusField(fieldPassword)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	frm := m.(*AddEntryForm)

	if frm.focused != fieldUsername {
		t.Errorf("expected focus on username after shift+tab, got %d", frm.focused)
	}
}

func TestAddEntryFormEnterAdvances(t *testing.T) {
	f := NewAddEntryForm(false)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	frm := m.(*AddEntryForm)

	if frm.focused != fieldPassword {
		t.Errorf("expected focus on password after enter, got %d", frm.focused)
	}
}

func TestAddEntryFormEnterSubmitsOnLastField(t *testing.T) {
	f := NewAddEntryForm(false)
	f.focusField(fieldTOTPAccount)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	frm := m.(*AddEntryForm)

	if !frm.submitted {
		t.Error("expected form to be submitted on enter from last field")
	}
}

func TestAddEntryFormCtrlCCancels(t *testing.T) {
	f := NewAddEntryForm(false)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	frm := m.(*AddEntryForm)

	if !frm.cancelled {
		t.Error("expected form to be cancelled on ctrl+c")
	}
}

func TestAddEntryFormBackspaceOnEmptyField(t *testing.T) {
	f := NewAddEntryForm(false)
	f.focusField(fieldPassword)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	frm := m.(*AddEntryForm)

	if frm.focused != fieldUsername {
		t.Errorf("expected focus on username after backspace on empty password, got %d", frm.focused)
	}
}

func TestAddEntryFormPasswordValidation(t *testing.T) {
	f := NewAddEntryForm(false)
	f.focusField(fieldPassword)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("weak")})
	frm := m.(*AddEntryForm)

	if frm.passwordErr == nil {
		t.Error("expected password validation error for weak password")
	}

	f2 := NewAddEntryForm(false)
	f2.focusField(fieldPassword)
	m2, _ := f2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("StrongP@ssw0rd123")})
	frm2 := m2.(*AddEntryForm)

	if frm2.passwordErr != nil {
		t.Errorf("expected no password error for strong password, got %v", frm2.passwordErr)
	}
}

func TestAddEntryFormForceSkipsValidation(t *testing.T) {
	f := NewAddEntryForm(true)
	f.focusField(fieldPassword)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("weak")})
	frm := m.(*AddEntryForm)

	if frm.passwordErr != nil {
		t.Errorf("expected no validation error with force=true, got %v", frm.passwordErr)
	}
}

func TestAddEntryFormDataCollection(t *testing.T) {
	f := NewAddEntryForm(false)
	f.username.SetValue("alice")
	f.password.SetValue("StrongP@ssw0rd123")
	f.url.SetValue("https://example.com")
	f.notes.SetValue("some notes")
	f.totpSecret.SetValue("GEZDGNBVGY3TQOJQ")
	f.totpIssuer.SetValue("Example")
	f.totpAccount.SetValue("alice@example.com")

	data := f.Data()

	if data["username"] != "alice" {
		t.Errorf("expected username alice, got %v", data["username"])
	}
	if data["password"] != "StrongP@ssw0rd123" {
		t.Errorf("expected password StrongP@ssw0rd123, got %v", data["password"])
	}
	if data["url"] != "https://example.com" {
		t.Errorf("expected url https://example.com, got %v", data["url"])
	}
	if data["notes"] != "some notes" {
		t.Errorf("expected notes, got %v", data["notes"])
	}

	totp, ok := data["totp"].(map[string]any)
	if !ok {
		t.Fatal("expected totp data")
	}
	if totp["secret"] != "GEZDGNBVGY3TQOJQ" {
		t.Errorf("expected totp secret, got %v", totp["secret"])
	}
	if totp["issuer"] != "Example" {
		t.Errorf("expected totp issuer, got %v", totp["issuer"])
	}
	if totp["account_name"] != "alice@example.com" {
		t.Errorf("expected totp account_name, got %v", totp["account_name"])
	}
}

func TestAddEntryFormDataSkipsEmptyFields(t *testing.T) {
	f := NewAddEntryForm(false)
	f.password.SetValue("StrongP@ssw0rd123")

	data := f.Data()

	if _, ok := data["username"]; ok {
		t.Error("expected no username in data when empty")
	}
	if _, ok := data["url"]; ok {
		t.Error("expected no url in data when empty")
	}
	if _, ok := data["notes"]; ok {
		t.Error("expected no notes in data when empty")
	}
	if _, ok := data["totp"]; ok {
		t.Error("expected no totp in data when empty")
	}
}

func TestAddEntryFormSetDefaults(t *testing.T) {
	f := NewAddEntryForm(false)
	defaults := map[string]any{
		"username": "bob",
		"password": "AnotherStrong1!",
		"url":      "https://test.com",
		"notes":    "test notes",
		"totp": map[string]any{
			"secret":       "SECRET123",
			"issuer":       "Test",
			"account_name": "bob@test",
		},
	}

	f.SetDefaults(defaults)

	if f.username.Value() != "bob" {
		t.Errorf("expected username bob, got %s", f.username.Value())
	}
	if f.password.Value() != "AnotherStrong1!" {
		t.Errorf("expected password, got %s", f.password.Value())
	}
	if f.url.Value() != "https://test.com" {
		t.Errorf("expected url, got %s", f.url.Value())
	}
	if f.notes.Value() != "test notes" {
		t.Errorf("expected notes, got %s", f.notes.Value())
	}
	if f.totpSecret.Value() != "SECRET123" {
		t.Errorf("expected totp secret, got %s", f.totpSecret.Value())
	}
	if f.totpIssuer.Value() != "Test" {
		t.Errorf("expected totp issuer, got %s", f.totpIssuer.Value())
	}
	if f.totpAccount.Value() != "bob@test" {
		t.Errorf("expected totp account, got %s", f.totpAccount.Value())
	}
}

func TestAddEntryFormNotesEnterAddsNewline(t *testing.T) {
	f := NewAddEntryForm(false)
	f.focusField(fieldNotes)

	m, _ := f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	frm := m.(*AddEntryForm)

	if frm.focused != fieldNotes {
		t.Errorf("expected focus to stay on notes after enter, got %d", frm.focused)
	}

	if !frm.submitted {
		if frm.notes.Value() == "" {
			t.Error("expected notes to contain newline after enter")
		}
	}
}

func TestAddEntryFormWindowSize(t *testing.T) {
	f := NewAddEntryForm(false)

	m, _ := f.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	frm := m.(*AddEntryForm)

	if frm.width != 120 {
		t.Errorf("expected width 120, got %d", frm.width)
	}
}
