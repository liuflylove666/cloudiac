package apps

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	cloudwebhooksrv "cloudiac/portal/services/cloudwebhook"
)

func TestCloudItsmCallbackSignatureMatches(t *testing.T) {
	cfg := models.CloudItsmConfig{
		Metadata: models.ResAttrs{
			"callbackSecret": "s3cr3t",
		},
	}
	body := []byte(`{"externalKey":"ITSM-1","status":"resolved"}`)
	secret := cloudItsmCallbackSecret(cfg)
	signature := cloudwebhooksrv.Signature(secret, body)

	if secret != "s3cr3t" {
		t.Fatalf("unexpected callback secret: %s", secret)
	}
	if !cloudItsmCallbackSignatureMatches(secret, body, signature) {
		t.Fatalf("expected sha256 signature to match")
	}
	if !cloudItsmCallbackSignatureMatches(secret, body, strings.TrimPrefix(signature, "sha256=")) {
		t.Fatalf("expected bare hex signature to match")
	}
	if cloudItsmCallbackSignatureMatches(secret, body, "sha256=bad") {
		t.Fatalf("expected invalid signature to be rejected")
	}
}

func TestCloudItsmGitOpsGateCallbackSecret(t *testing.T) {
	cfg := models.CloudItsmConfig{
		Metadata: models.ResAttrs{
			"callbackSecret":             "itsm-secret",
			"gitOpsGateCallbackSecret":   "gitops-secret",
			"statusCallbackSecret":       "status-secret",
			"unusedGitOpsCallbackSecret": "ignored",
		},
	}
	if got := cloudItsmGitOpsGateCallbackSecret(cfg); got != "gitops-secret" {
		t.Fatalf("expected dedicated gitops callback secret, got %s", got)
	}
	cfg.Metadata = models.ResAttrs{"callbackSecret": "fallback-secret"}
	if got := cloudItsmGitOpsGateCallbackSecret(cfg); got != "fallback-secret" {
		t.Fatalf("expected fallback callback secret, got %s", got)
	}
}

func TestCloudItsmCallbackUpdateStatusForm(t *testing.T) {
	callback := &forms.CloudItsmTicketStatusCallbackForm{
		ConnectorId:      models.Id("citc-1"),
		Status:           models.CloudItsmTicketStatusResolved,
		ExternalKey:      "ITSM-1",
		Comment:          "外部 ITSM 已解决",
		SignatureVersion: "1",
		Payload: models.ResAttrs{
			"resolution": "fixed",
		},
	}
	callback.Bind(url.Values{
		"connectorId": {"citc-1"},
		"status":      {models.CloudItsmTicketStatusResolved},
		"externalKey": {"ITSM-1"},
		"comment":     {"外部 ITSM 已解决"},
		"payload":     {"true"},
	})

	form := cloudItsmCallbackUpdateStatusForm(models.Id("cit-1"), callback)
	if form.Id != "cit-1" || form.Status != models.CloudItsmTicketStatusResolved {
		t.Fatalf("unexpected status form identity: %#v", form)
	}
	if !form.HasKey("externalKey") || form.HasKey("externalId") {
		t.Fatalf("unexpected external key flags")
	}
	if form.ExternalKey != "ITSM-1" || form.Comment != "外部 ITSM 已解决" {
		t.Fatalf("unexpected status form fields: %#v", form)
	}
	payload := modelResAttrs(form.Payload)
	if payload["resolution"] != "fixed" {
		t.Fatalf("expected callback payload to be preserved, got %#v", payload)
	}
	callbackInfo := modelResAttrs(payload["callback"])
	if callbackInfo["connectorId"] != "citc-1" || callbackInfo["signatureVerified"] != true {
		t.Fatalf("unexpected callback metadata: %#v", callbackInfo)
	}
}

func TestCloudItsmGitOpsGateCallbackParams(t *testing.T) {
	now := time.Date(2026, 6, 22, 15, 0, 0, 0, time.UTC)
	params := models.ResAttrs{
		"gitOpsPullRequestUrl": "https://gitlab.example.com/platform/iac/-/merge_requests/7",
		"gitOpsPipelineUrl":    "https://gitlab.example.com/platform/iac/-/pipelines/99",
		"gitOpsReviewStatus":   "pending",
		"gitOpsPipelineStatus": "running",
		"gitOpsCallbackCount":  1,
	}
	form := &forms.CloudItsmGitOpsGateCallbackForm{
		ConnectorId:    models.Id("citc-1"),
		OperationId:    models.Id("cop-1"),
		ReviewStatus:   "approved",
		PipelineStatus: "success",
		CommitSha:      "abc123",
		ExternalRunId:  "pipeline-99",
		Comment:        "MR approved and pipeline passed",
		Payload: models.ResAttrs{
			"source": "gitlab",
		},
	}

	next, gate, callback, err := cloudItsmGitOpsGateCallbackParams(params, form, now)
	if err != nil {
		t.Fatalf("unexpected callback params error: %v", err)
	}
	if gate["status"] != cloudItsmGitOpsGateStatusPassed || gate["reviewStatus"] != "approved" || gate["pipelineStatus"] != "passed" {
		t.Fatalf("unexpected gate: %#v", gate)
	}
	if next["gateStatus"] != cloudItsmGitOpsGateStatusPassed || attrInt(next, "gitOpsCallbackCount") != 2 {
		t.Fatalf("unexpected next params: %#v", next)
	}
	if params["gitOpsPipelineStatus"] != "running" {
		t.Fatalf("expected original params to remain unchanged, got %#v", params)
	}
	if callback["commitSha"] != "abc123" || callback["externalRunId"] != "pipeline-99" || callback["receivedAt"] != now.Format(time.RFC3339) {
		t.Fatalf("unexpected callback snapshot: %#v", callback)
	}
	payload := modelResAttrs(callback["payload"])
	if payload["source"] != "gitlab" {
		t.Fatalf("expected raw callback payload, got %#v", payload)
	}
}

func TestCloudItsmGitOpsGateCallbackSeedsFromExistingGate(t *testing.T) {
	now := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	params := models.ResAttrs{
		"gitOpsGate": models.ResAttrs{
			"pullRequestUrl": "https://github.com/acme/iac/pull/12",
			"pipelineUrl":    "https://github.com/acme/iac/actions/runs/12",
			"reviewStatus":   "pending",
			"pipelineStatus": "running",
		},
	}
	form := &forms.CloudItsmGitOpsGateCallbackForm{
		ConnectorId:    models.Id("citc-1"),
		ReviewStatus:   "approved",
		PipelineStatus: "passed",
	}

	next, gate, _, err := cloudItsmGitOpsGateCallbackParams(params, form, now)
	if err != nil {
		t.Fatalf("unexpected callback params error: %v", err)
	}
	if next["gitOpsPullRequestUrl"] != "https://github.com/acme/iac/pull/12" || gate["status"] != cloudItsmGitOpsGateStatusPassed {
		t.Fatalf("expected gate fields seeded from snapshot, next=%#v gate=%#v", next, gate)
	}
}

func TestCloudItsmTicketStatusEndpoint(t *testing.T) {
	cfg := models.CloudItsmConfig{
		BaseUrl: "https://itsm.example.com/api",
		Metadata: models.ResAttrs{
			"statusFetchPath": "/tickets/{externalKey}",
		},
	}
	ticket := models.CloudItsmTicket{
		ExternalKey: "ITSM-1",
	}
	ticket.Id = models.Id("cit-1")
	if got := cloudItsmTicketStatusEndpoint(cfg, ticket); got != "https://itsm.example.com/api/tickets/ITSM-1" {
		t.Fatalf("unexpected endpoint: %s", got)
	}

	cfg = models.CloudItsmConfig{
		Provider: models.CloudItsmProviderJira,
		BaseUrl:  "https://jira.example.com",
	}
	if got := cloudItsmTicketStatusEndpoint(cfg, ticket); got != "https://jira.example.com/rest/api/2/issue/ITSM-1" {
		t.Fatalf("unexpected jira endpoint: %s", got)
	}
}

func TestCloudItsmExternalStatusMapping(t *testing.T) {
	cfg := models.CloudItsmConfig{
		Metadata: models.ResAttrs{
			"statusMap": models.ResAttrs{
				"待发布": models.CloudItsmTicketStatusInProgress,
			},
		},
	}
	cases := []struct {
		raw  string
		want string
	}{
		{raw: "待发布", want: models.CloudItsmTicketStatusInProgress},
		{raw: "In Progress", want: models.CloudItsmTicketStatusInProgress},
		{raw: "Done", want: models.CloudItsmTicketStatusResolved},
		{raw: "7", want: models.CloudItsmTicketStatusClosed},
		{raw: "cancelled", want: models.CloudItsmTicketStatusCanceled},
	}
	for _, tc := range cases {
		if got := cloudItsmNormalizeExternalStatus(cfg, tc.raw); got != tc.want {
			t.Fatalf("status %s expected %s, got %s", tc.raw, tc.want, got)
		}
	}
}

func TestCloudItsmExternalStatusString(t *testing.T) {
	payload := models.ResAttrs{
		"fields": models.ResAttrs{
			"status": models.ResAttrs{
				"name": "Done",
			},
		},
	}
	if got := cloudItsmExternalStatusString(payload); got != "Done" {
		t.Fatalf("expected nested status, got %s", got)
	}
}
