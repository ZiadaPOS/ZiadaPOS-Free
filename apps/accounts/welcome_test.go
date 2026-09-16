package accounts

import (
	"strings"
	"testing"
)

func TestBuildWelcomeEmailBody(t *testing.T) {
	link := "https://app.ziadapos.com/api/v1/auth/confirm-email/?token=abc123"
	body := BuildWelcomeEmailBody("Amina Juma", "Duka la Amina", link)

	for _, want := range []string{
		"Amina Juma",
		" for Duka la Amina",
		"7-day free trial",
		link,
		"Confirm my email",
		"app.ziadapos.com",
		"no-reply@ziadapos.com",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("welcome body missing %q", want)
		}
	}
}

func TestBuildWelcomeEmailBodyNoShop(t *testing.T) {
	body := BuildWelcomeEmailBody("Amina Juma", "", "https://example.com/x")
	if strings.Contains(body, "Your account for ") {
		t.Errorf("expected no shop suffix in account line")
	}
	if !strings.Contains(body, "Your account has been created") {
		t.Errorf("expected generic account line")
	}
}

func TestShopSuffix(t *testing.T) {
	if got := shopSuffix("Duka"); got != " for Duka" {
		t.Errorf("got %q", got)
	}
	if got := shopSuffix("  "); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
