package buildkite

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestParseWebHook(t *testing.T) {
	tests := []struct {
		payload      interface{}
		messageType  string
		errorMessage string
	}{
		{
			payload:     &JobScheduledEvent{},
			messageType: "job.scheduled",
		},
		{
			payload:     &PingEvent{},
			messageType: "ping",
		},
		{
			payload:      &PingEvent{},
			messageType:  "invalid",
			errorMessage: "unknown X-Buildkite-Event in message: invalid",
		},
	}

	for _, test := range tests {
		p, err := json.Marshal(test.payload)
		if err != nil {
			t.Fatalf("Marshal(%#v): %v", test.payload, err)
		}

		got, err := ParseWebHook(test.messageType, p)
		if err != nil {
			if test.errorMessage != "" {
				if err.Error() != test.errorMessage {
					t.Errorf("ParseWebHook(%#v, %#v) expected error, got %#v", test.messageType, test.payload, err.Error())
				}
				continue
			}
			t.Fatalf("ParseWebHook: %v", err)
		}

		if want := test.payload; !reflect.DeepEqual(got, want) {
			t.Errorf("ParseWebHook(%#v, %#v) = %#v, want %#v", test.messageType, p, got, want)
		}
	}
}

func TestValidatePayload(t *testing.T) {
	const defaultBody = `{"event":"ping","service":{"id":"c9f8372d-c0cd-43dc-9274-768a875cf6ca","provider":"webhook","settings":{"url":"https://server.com/webhooks"}},"organization":{"id":"49801950-1df0-474f-bb56-ad6a930c5cb9","graphql_id":"T3JnYW5pemF0aW9uLS0tZTBmMzk3MgsTksGkxOWYtZTZjNzczZTJiYjEy","url":"https://api.buildkite.com/v2/organizations/acme-inc","web_url":"https://buildkite.com/acme-inc","name":"ACME Inc","slug":"acme-inc","agents_url":"https://api.buildkite.com/v2/organizations/acme-inc/agents","emojis_url":"https://api.buildkite.com/v2/organizations/acme-inc/emojis","created_at":"2021-02-03T20:34:10.486Z","pipelines_url":"https://api.buildkite.com/v2/organizations/acme-inc/pipelines"},"sender":{"id":"c9f8372d-c0cd-43dc-9269-bcbb7f308e3f","name":"ACME Man"}}`
	const defaultSignature = "timestamp=1642080837,signature=582d496ac2d869dd97a3101c4cda346288c49a742592daf582ec64c86449f79c"
	const payloadSignatureError = "payload signature check failed"
	secretKey := []byte("29b1ff5779c76bd48ba6705eb99ff970")

	tests := []struct {
		signature   string
		event       string
		wantEvent   string
		wantPayload string
	}{
		// The following tests generate expected errors:
		{},                     // Missing signature
		{signature: "invalid"}, // Invalid signature format
		{signature: "timestamp=1642080837,signature=yo"}, // Signature not hex string
		// The following tests expect err=nil:
		{
			signature:   defaultSignature,
			event:       "ping",
			wantEvent:   "ping",
			wantPayload: defaultBody,
		},
	}

	for _, test := range tests {
		buf := bytes.NewBufferString(defaultBody)
		req, err := http.NewRequest("POST", "http://localhost/webhook", buf)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}

		if test.signature != "" {
			req.Header.Set(signatureHeader, test.signature)
		}

		req.Header.Set("Content-Type", "application/json")

		got, err := ValidatePayload(req, secretKey)
		if err != nil {
			if test.wantPayload != "" {
				t.Errorf("ValidatePayload(%#v): err = %v, want nil", test, err)
			}

			continue
		}

		if string(got) != test.wantPayload {
			t.Errorf("ValidatePayload = %q, want %q", got, test.wantPayload)
		}
	}
}

func TestWebHookType(t *testing.T) {
	eventType := "ping"

	req, err := http.NewRequest("POST", "http://localhost", nil)
	if err != nil {
		t.Fatalf("Error building requet: %v", err)
	}

	req.Header.Set(eventTypeHeader, eventType)

	got := WebHookType(req)
	if got != eventType {
		t.Errorf("WebHookType(%#v) = %q, want %q", req, got, eventType)
	}
}
