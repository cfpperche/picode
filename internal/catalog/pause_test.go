package catalog

import "testing"

func TestPauseAccountPromotesAnotherLiveRow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := PutAPIKey("xai", "sk-two"); err != nil {
		t.Fatal(err)
	}
	var active, other string
	for _, a := range AccountsFor("xai") {
		if a.Active {
			active = a.ID
		} else {
			other = a.ID
		}
	}
	if active == "" || other == "" {
		t.Fatalf("want two rows, got %+v", AccountsFor("xai"))
	}
	if err := PauseAccount("xai", active, true); err != nil {
		t.Fatalf("pause: %v", err)
	}
	accs := AccountsFor("xai")
	for _, a := range accs {
		if a.ID == active {
			if !a.Paused {
				t.Fatal("paused row is not marked paused")
			}
			if a.Active {
				t.Fatal("a paused row must not stay active")
			}
		}
		if a.ID == other && !a.Active {
			t.Fatal("the live row should have been promoted")
		}
	}
	// auth.json must now hold the promoted credential, not the paused one.
	if key := mustKey(t, home, "xai"); key != "sk-one" {
		t.Fatalf("auth.json holds %q, want the promoted row", key)
	}
	if err := ActivateAccount("xai", active); err == nil {
		t.Fatal("activating a paused account must be refused")
	}
	if err := PauseAccount("xai", active, false); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if err := ActivateAccount("xai", active); err != nil {
		t.Fatalf("activate after resume: %v", err)
	}
}

func TestPauseLastLiveAccountRefused(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := PutAPIKey("groq", "sk-only"); err != nil {
		t.Fatal(err)
	}
	accs := AccountsFor("groq")
	if len(accs) != 1 {
		t.Fatalf("want one row, got %d", len(accs))
	}
	if err := PauseAccount("groq", accs[0].ID, true); err == nil {
		t.Fatal("pausing the only live account must be refused — that is Sign out")
	}
}

func TestSetAccountIdentityKeepsWhatItAlreadyKnows(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := PutAPIKey("kimi-coding", "sk-a"); err != nil {
		t.Fatal(err)
	}
	id := AccountsFor("kimi-coding")[0].ID
	if err := SetAccountIdentity("kimi-coding", id, "who@example.com", "Pro"); err != nil {
		t.Fatal(err)
	}
	// A later adapter that only knows the plan must not erase the email.
	if err := SetAccountIdentity("kimi-coding", id, "", "Max"); err != nil {
		t.Fatal(err)
	}
	a := AccountsFor("kimi-coding")[0]
	if a.Email != "who@example.com" {
		t.Fatalf("email %q was lost", a.Email)
	}
	if a.Plan != "Max" {
		t.Fatalf("plan %q was not updated", a.Plan)
	}
}

func TestEnvKeyMakesAProviderSignedIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GROQ_API_KEY", "sk-from-the-environment")
	name, val, ok := EnvKeyName("groq")
	if !ok || name != "GROQ_API_KEY" || val != "sk-from-the-environment" {
		t.Fatalf("EnvKeyName = %q %v", name, ok)
	}
	if got := ActiveAuthType("groq"); got != LoginAPIKey {
		t.Fatalf("ActiveAuthType = %q, want api_key: pi reads the variable", got)
	}
	if k, ok := ActiveAPIKey("groq"); !ok || k != "sk-from-the-environment" {
		t.Fatalf("ActiveAPIKey = %q %v", k, ok)
	}
	if got := ActiveLabel("groq"); got != "GROQ_API_KEY" {
		t.Fatalf("ActiveLabel = %q, want the variable name", got)
	}
	// auth.json still wins when it has an entry.
	if err := PutAPIKey("groq", "sk-from-auth-json"); err != nil {
		t.Fatal(err)
	}
	if k, _ := ActiveAPIKey("groq"); k != "sk-from-auth-json" {
		t.Fatalf("ActiveAPIKey = %q, want the stored credential", k)
	}
}
