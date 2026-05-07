package types

import "testing"

func TestAlertService_GetType_attributes(t *testing.T) {
	a := &AlertService{Settings: map[string]interface{}{"type": "Mail"}}
	if got := a.GetType(); got != "Mail" {
		t.Errorf("attrs path: got %q, want Mail", got)
	}
}

func TestAlertService_GetType_topLevelFallback(t *testing.T) {
	a := &AlertService{Type: "Slack"}
	if got := a.GetType(); got != "Slack" {
		t.Errorf("top-level fallback: got %q, want Slack", got)
	}
}

func TestAlertService_GetType_attrsEmpty_fallsBack(t *testing.T) {
	a := &AlertService{
		Settings: map[string]interface{}{"type": ""},
		Type:     "Telegram",
	}
	if got := a.GetType(); got != "Telegram" {
		t.Errorf("empty attrs type: got %q, want Telegram", got)
	}
}

func TestAlertService_GetType_attrsNonString_fallsBack(t *testing.T) {
	a := &AlertService{
		Settings: map[string]interface{}{"type": 42},
		Type:     "PagerDuty",
	}
	if got := a.GetType(); got != "PagerDuty" {
		t.Errorf("non-string attrs type: got %q, want PagerDuty", got)
	}
}

func TestAlertService_GetType_empty(t *testing.T) {
	a := &AlertService{}
	if got := a.GetType(); got != "" {
		t.Errorf("empty: got %q, want empty", got)
	}
}
