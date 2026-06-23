package apps

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
)

func TestCloudEventItsmDispatchSpecFromPayload(t *testing.T) {
	event := models.CloudEvent{
		Source:    models.CloudEventSourceSync,
		EventType: "cloud.sync.policy.failed",
		Level:     models.CloudEventLevelError,
		Title:     "采集失败",
		Message:   "permission denied",
		Payload: models.ResAttrs{
			"itsmAutoTicket":   true,
			"itsmConnectorIds": []string{"citc-1", "citc-2"},
			"itsmPriority":     "critical",
		},
	}
	event.Id = models.Id("cev-1")

	spec := cloudEventItsmDispatchSpecFromPayload(event)
	if !spec.Enabled || spec.Reason != "auto_ticket" {
		t.Fatalf("expected auto ITSM dispatch, got %#v", spec)
	}
	if spec.Priority != "critical" {
		t.Fatalf("expected configured priority, got %#v", spec.Priority)
	}
	if len(spec.ConnectorIds) != 2 || spec.ConnectorIds[0] != "citc-1" || spec.ConnectorIds[1] != "citc-2" {
		t.Fatalf("unexpected connector ids: %#v", spec.ConnectorIds)
	}

	event.Payload = models.ResAttrs{
		"itsmAutoTicket":  false,
		"itsmConnectorId": "citc-1",
	}
	spec = cloudEventItsmDispatchSpecFromPayload(event)
	if spec.Enabled || spec.Reason != "disabled" {
		t.Fatalf("expected explicit disable to win, got %#v", spec)
	}

	event.Payload = models.ResAttrs{"itsmConnectorId": "citc-1"}
	spec = cloudEventItsmDispatchSpecFromPayload(event)
	if !spec.Enabled || spec.Reason != "connector_route" {
		t.Fatalf("expected connector route dispatch, got %#v", spec)
	}
}

func TestCloudItsmTicketRequestPayloadForEvent(t *testing.T) {
	c := &ctx.ServiceContext{
		UserId:   models.Id("u-1"),
		Username: "alice",
		Email:    "alice@example.com",
	}
	event := models.CloudEvent{
		OrgId:          models.Id("org-1"),
		ProjectId:      models.Id("p-1"),
		EnvId:          models.Id("env-1"),
		ActorId:        models.Id("u-2"),
		Source:         models.CloudEventSourceSync,
		EventType:      "cloud.sync.policy.failed",
		Level:          models.CloudEventLevelError,
		Status:         "failed",
		Provider:       "oci",
		AccountId:      "acct-1",
		CloudAccountId: models.Id("ca-1"),
		ResourceType:   "cloud_sync_policy",
		ResourceId:     "csp-1",
		ResourceName:   "生产采集策略",
		Title:          "云采集失败",
		Message:        "permission denied",
		Payload: models.ResAttrs{
			"failureCategory": "permission",
			"cloudService":    "iaas",
			"itsmAutoTicket":  true,
		},
	}
	event.Id = models.Id("cev-1")
	cfg := models.CloudItsmConfig{
		OrgId:      event.OrgId,
		Name:       "本地工单",
		Provider:   models.CloudItsmProviderGeneric,
		ProjectKey: "CLOUD",
		TicketType: "incident",
	}
	cfg.Id = models.Id("citc-1")
	spec := cloudEventItsmDispatchSpecFromPayload(event)

	payload := cloudItsmTicketRequestPayloadForEvent(c, event, cfg, spec)
	if payload["title"] != "云采集失败" || payload["priority"] != "high" {
		t.Fatalf("unexpected top-level payload: %#v", payload)
	}
	eventPayload := modelResAttrs(payload["event"])
	if eventPayload["id"] != "cev-1" || eventPayload["eventType"] != "cloud.sync.policy.failed" {
		t.Fatalf("unexpected event payload: %#v", eventPayload)
	}
	dispatchPayload := modelResAttrs(payload["dispatch"])
	if dispatchPayload["reason"] != "auto_ticket" {
		t.Fatalf("unexpected dispatch payload: %#v", dispatchPayload)
	}
	robotPayload := modelResAttrs(payload["robot"])
	if !attrBool(robotPayload, "processed") || robotPayload["processor"] != cloudItsmRobotProcessorCloudiac {
		t.Fatalf("expected robot processing payload, got %#v", robotPayload)
	}
	if robotPayload["automationMode"] != cloudItsmRobotModeEventTicket || robotPayload["ticketChannel"] != cloudItsmRobotChannelLocal {
		t.Fatalf("unexpected robot mode/channel: %#v", robotPayload)
	}
	if !cloudItsmTestStringSliceContains(cloudItsmRobotProcessingTags(robotPayload["tags"]), cloudItsmRobotTagEventAutoTicket) {
		t.Fatalf("expected event auto ticket tag, got %#v", robotPayload["tags"])
	}
	requester := modelResAttrs(payload["requester"])
	if requester["userId"] != "u-2" {
		t.Fatalf("expected source event actor as requester, got %#v", requester)
	}
}

func TestCloudItsmExternalPayloadForJira(t *testing.T) {
	c := &ctx.ServiceContext{
		UserId:   models.Id("u-1"),
		Username: "alice",
		Email:    "alice@example.com",
	}
	cfg := models.CloudItsmConfig{
		Name:       "Jira",
		Provider:   models.CloudItsmProviderJira,
		ProjectKey: "CLOUD",
		TicketType: "Change",
		Metadata: models.ResAttrs{
			"labels": []interface{}{"iac"},
			"priorityMapping": models.ResAttrs{
				"high": "High",
			},
		},
	}
	operation := models.CloudOperation{
		Name:          "GitOps/IaC 变更申请",
		OperationType: models.CloudOperationTypeSelfService,
		Action:        models.CloudOperationActionGitOpsIacChange,
		Status:        models.CloudOperationStatusRunning,
		RiskLevel:     models.CloudOperationRiskHigh,
		Params: models.ResAttrs{
			"catalogKey": models.CloudOperationActionGitOpsIacChange,
		},
	}
	operation.Id = models.Id("co-1")

	payload := cloudItsmTicketRequestPayload(c, operation, cfg, "GitOps/IaC 变更申请", "PR review passed", "high")
	externalPayload := modelResAttrs(payload["externalPayload"])
	fields := modelResAttrs(externalPayload["fields"])
	project := modelResAttrs(fields["project"])
	issueType := modelResAttrs(fields["issuetype"])
	priority := modelResAttrs(fields["priority"])

	if payload["externalPayloadProvider"] != models.CloudItsmProviderJira || payload["externalPayloadMode"] != "provider_custom_field_mapping" {
		t.Fatalf("expected jira external payload metadata, got %#v", payload)
	}
	if project["key"] != "CLOUD" || fields["summary"] != "GitOps/IaC 变更申请" || issueType["name"] != "Change" {
		t.Fatalf("unexpected jira fields: %#v", fields)
	}
	if priority["name"] != "High" {
		t.Fatalf("unexpected jira priority: %#v", priority)
	}
	labels := cloudSyncPolicyAttrStringSlice(fields["labels"])
	for _, label := range []string{"iac", "cloudiac", "cloudiac_high"} {
		if !cloudItsmTestStringSliceContains(labels, label) {
			t.Fatalf("expected jira label %s in %#v", label, labels)
		}
	}
}

func TestCloudItsmExternalPayloadCustomFieldMapping(t *testing.T) {
	c := &ctx.ServiceContext{
		UserId:   models.Id("u-1"),
		Username: "alice",
		Email:    "alice@example.com",
	}
	cfg := models.CloudItsmConfig{
		Name:     "Generic ITSM",
		Provider: models.CloudItsmProviderGeneric,
		Metadata: models.ResAttrs{
			"fieldDefaults": models.ResAttrs{
				"details.source": "cloudiac",
			},
			"fieldMappings": models.ResAttrs{
				"short_description":    "$.title",
				"details.operation_id": "$.operation.id",
				"details.catalog_key":  "$.operation.params.catalogKey",
				"details.literal":      "literal-value",
			},
		},
	}
	operation := models.CloudOperation{
		Name:          "权限申请",
		OperationType: models.CloudOperationTypeSelfService,
		Action:        models.CloudOperationActionPermissionRequest,
		Status:        models.CloudOperationStatusRunning,
		RiskLevel:     models.CloudOperationRiskMedium,
		Params: models.ResAttrs{
			"catalogKey": models.CloudOperationActionPermissionRequest,
		},
	}
	operation.Id = models.Id("co-2")

	payload := cloudItsmTicketRequestPayload(c, operation, cfg, "权限申请", "申请生产只读权限", "medium")
	externalPayload := modelResAttrs(payload["externalPayload"])
	details := modelResAttrs(externalPayload["details"])

	if payload["externalPayloadMode"] != "custom_field_mapping" {
		t.Fatalf("expected custom field mapping mode, got %#v", payload["externalPayloadMode"])
	}
	if externalPayload["short_description"] != "权限申请" {
		t.Fatalf("expected mapped title, got %#v", externalPayload)
	}
	if details["operation_id"] != "co-2" || details["catalog_key"] != models.CloudOperationActionPermissionRequest ||
		details["source"] != "cloudiac" || details["literal"] != "literal-value" {
		t.Fatalf("unexpected mapped details: %#v", details)
	}
}

func TestCloudItsmSubmitTicketUsesServiceNowPayload(t *testing.T) {
	var received models.ResAttrs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/now/table/incident" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"sys_id":"sys-1","number":"INC001","self":"https://servicenow.example.com/nav_to.do?uri=incident.do?sys_id=sys-1"}}`))
	}))
	defer server.Close()

	c := &ctx.ServiceContext{
		UserId:   models.Id("u-1"),
		Username: "alice",
		Email:    "alice@example.com",
	}
	cfg := models.CloudItsmConfig{
		Name:       "ServiceNow",
		Provider:   models.CloudItsmProviderServiceNow,
		BaseUrl:    server.URL,
		TicketType: "incident",
		Metadata: models.ResAttrs{
			"createTicketPath": "/api/now/table/incident",
		},
	}
	operation := models.CloudOperation{
		Name:          "扩容申请",
		OperationType: models.CloudOperationTypeSelfService,
		Action:        models.CloudOperationActionResizeInstance,
		Status:        models.CloudOperationStatusRunning,
		RiskLevel:     models.CloudOperationRiskHigh,
		ResourceType:  "aws.ec2.instance",
		ResourceId:    "i-1",
	}
	operation.Id = models.Id("co-3")
	operation.ProjectId = models.Id("p-1")

	payload := cloudItsmTicketRequestPayload(c, operation, cfg, "扩容申请", "EC2 扩容到 4C8G", "high")
	submit := submitCloudItsmTicket(cfg, payload)

	if submit.Status != models.CloudItsmTicketStatusSubmitted || submit.ExternalId != "sys-1" || submit.ExternalKey != "INC001" {
		t.Fatalf("unexpected submit result: %#v", submit)
	}
	if received["short_description"] != "扩容申请" || received["u_cloudiac_operation_id"] != "co-3" ||
		received["u_cloudiac_resource_type"] != "aws.ec2.instance" || received["urgency"] != "1" {
		t.Fatalf("unexpected ServiceNow request body: %#v", received)
	}
	if _, ok := received["operation"]; ok {
		t.Fatalf("expected mapped external payload, got canonical body: %#v", received)
	}
	if submit.ResponsePayload["requestPayloadMode"] != "provider_default_field_mapping" ||
		submit.ResponsePayload["requestPayloadProvider"] != models.CloudItsmProviderServiceNow {
		t.Fatalf("unexpected submit response metadata: %#v", submit.ResponsePayload)
	}
}

func TestCloudItsmSubmitRetryMetadata(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cfg := models.CloudItsmConfig{
		BaseUrl: "https://itsm.example.com",
		Metadata: models.ResAttrs{
			"submitRetryMaxAttempts":       4,
			"submitRetryBackoffSeconds":    60,
			"submitRetryMaxBackoffSeconds": 600,
		},
	}
	result := cloudItsmSubmitResult{
		Status:       models.CloudItsmTicketStatusFailed,
		ErrorMessage: "ITSM 工单接口返回 HTTP 500",
		ResponsePayload: models.ResAttrs{
			"statusCode": 500,
		},
	}

	result = cloudItsmAttachSubmitRetryResult(cfg, result, 2, now)
	retry := modelResAttrs(result.ResponsePayload["submitRetry"])
	if retry["enabled"] != true || retry["eligible"] != true || retry["attempt"] != 2 || retry["nextAttempt"] != 3 {
		t.Fatalf("unexpected retry metadata: %#v", retry)
	}
	if retry["backoffSeconds"] != 120 || retry["maxAttempts"] != 4 || retry["lastStatusCode"] != 500 {
		t.Fatalf("unexpected retry backoff/status: %#v", retry)
	}
	if retry["nextRetryAt"] != now.Add(120*time.Second).Format(time.RFC3339) {
		t.Fatalf("unexpected next retry time: %#v", retry)
	}
}

func TestCloudItsmTicketSubmitRetryEligibility(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cfg := models.CloudItsmConfig{
		BaseUrl: "https://itsm.example.com",
		Metadata: models.ResAttrs{
			"createTicketPath":          "/tickets",
			"submitRetryMaxAttempts":    3,
			"submitRetryBackoffSeconds": 60,
		},
	}
	ticket := models.CloudItsmTicket{
		Status: models.CloudItsmTicketStatusFailed,
		RequestPayload: models.ResAttrs{
			"title": "失败提交",
		},
		ResponsePayload: models.ResAttrs{
			"submitRetry": models.ResAttrs{
				"attempt":     1,
				"nextRetryAt": now.Add(-time.Minute).Format(time.RFC3339),
			},
		},
	}
	ticket.Id = models.Id("cit-1")

	eligibility := cloudItsmTicketSubmitRetryEligibility(cfg, ticket, now, false)
	if !eligibility.Eligible || eligibility.NextAttempt != 2 || eligibility.Reason != "due" {
		t.Fatalf("expected due retry eligibility, got %#v", eligibility)
	}

	ticket.ExternalKey = "INC001"
	eligibility = cloudItsmTicketSubmitRetryEligibility(cfg, ticket, now, false)
	if eligibility.Eligible || eligibility.Reason != "external_identity_present" {
		t.Fatalf("expected external identity skip, got %#v", eligibility)
	}
}

func TestCloudItsmSubmitRetryDeadLetterAtMaxAttempts(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cfg := models.CloudItsmConfig{
		BaseUrl: "https://itsm.example.com",
		Metadata: models.ResAttrs{
			"submitRetryMaxAttempts": 2,
		},
	}
	ticket := models.CloudItsmTicket{
		Status: models.CloudItsmTicketStatusFailed,
		RequestPayload: models.ResAttrs{
			"title": "失败提交",
		},
		ResponsePayload: models.ResAttrs{
			"submitRetry": models.ResAttrs{
				"attempt":     2,
				"nextRetryAt": now.Add(-time.Minute).Format(time.RFC3339),
			},
		},
	}

	eligibility := cloudItsmTicketSubmitRetryEligibility(cfg, ticket, now, true)
	if eligibility.Eligible || !eligibility.DeadLetter || eligibility.Reason != "max_attempts_reached" {
		t.Fatalf("expected dead letter at max attempts, got %#v", eligibility)
	}
}

func TestCloudItsmSubmitRetryHttpFailureRecordsNextRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"bad gateway"}`))
	}))
	defer server.Close()

	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cfg := models.CloudItsmConfig{
		BaseUrl: server.URL,
		Metadata: models.ResAttrs{
			"createTicketPath":          "/tickets",
			"submitRetryBackoffSeconds": 60,
		},
	}
	result := submitCloudItsmTicketWithAttempt(cfg, models.ResAttrs{"title": "失败提交"}, 1, now)
	retry := modelResAttrs(result.ResponsePayload["submitRetry"])
	if result.Status != models.CloudItsmTicketStatusFailed || result.ErrorMessage != "ITSM 工单接口返回 HTTP 502" {
		t.Fatalf("unexpected submit failure result: %#v", result)
	}
	if retry["eligible"] != true || retry["attempt"] != 1 || retry["nextAttempt"] != 2 || retry["backoffSeconds"] != 60 {
		t.Fatalf("unexpected retry metadata: %#v", retry)
	}
	if retry["nextRetryAt"] != now.Add(time.Minute).Format(time.RFC3339) {
		t.Fatalf("unexpected next retry time: %#v", retry)
	}
}

func TestCloudItsmSubmitRetryQueueStatus(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cfg := &models.CloudItsmConfig{
		Status:  models.CloudItsmConfigStatusEnabled,
		BaseUrl: "https://itsm.example.com",
		Metadata: models.ResAttrs{
			"createTicketPath":       "/tickets",
			"submitRetryMaxAttempts": 3,
		},
	}
	ticket := models.CloudItsmTicket{
		Status: models.CloudItsmTicketStatusFailed,
		RequestPayload: models.ResAttrs{
			"title": "失败提交",
		},
		ResponsePayload: models.ResAttrs{
			"submitRetry": models.ResAttrs{
				"attempt":     1,
				"nextAttempt": 2,
				"nextRetryAt": now.Add(-time.Minute).Format(time.RFC3339),
			},
		},
	}
	status, eligibility := cloudItsmSubmitRetryQueueStatus(cfg, ticket, now)
	if status != "due" || !eligibility.Eligible || eligibility.Reason != "due" {
		t.Fatalf("expected due queue item, got status=%s eligibility=%#v", status, eligibility)
	}

	ticket.ResponsePayload["submitRetry"] = models.ResAttrs{
		"attempt":     1,
		"nextAttempt": 2,
		"nextRetryAt": now.Add(time.Hour).Format(time.RFC3339),
	}
	status, eligibility = cloudItsmSubmitRetryQueueStatus(cfg, ticket, now)
	if status != "future" || eligibility.Reason != "not_due" {
		t.Fatalf("expected future queue item, got status=%s eligibility=%#v", status, eligibility)
	}

	ticket.ResponsePayload["submitRetry"] = models.ResAttrs{
		"attempt":     3,
		"nextAttempt": 4,
		"deadLetter":  true,
	}
	status, eligibility = cloudItsmSubmitRetryQueueStatus(cfg, ticket, now)
	if status != "dead_letter" || !eligibility.DeadLetter {
		t.Fatalf("expected dead letter queue item, got status=%s eligibility=%#v", status, eligibility)
	}

	status, eligibility = cloudItsmSubmitRetryQueueStatus(nil, ticket, now)
	if status != "dead_letter" || eligibility.Reason != "connector_missing" || !eligibility.DeadLetter {
		t.Fatalf("expected missing connector dead letter item, got status=%s eligibility=%#v", status, eligibility)
	}

	ticket.ResponsePayload["submitRetry"] = models.ResAttrs{
		"attempt":     1,
		"nextAttempt": 2,
		"nextRetryAt": now.Add(-time.Minute).Format(time.RFC3339),
	}
	status, eligibility = cloudItsmSubmitRetryQueueStatus(nil, ticket, now)
	if status != "skipped" || eligibility.Reason != "connector_missing" {
		t.Fatalf("expected connector missing skipped item, got status=%s eligibility=%#v", status, eligibility)
	}
}

func TestCloudItsmSubmitRetryQueueReportBreakdown(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cfg := &models.CloudItsmConfig{
		Name:     "Jira生产",
		Provider: "jira",
		Status:   models.CloudItsmConfigStatusEnabled,
		BaseUrl:  "https://itsm.example.com",
		Metadata: models.ResAttrs{
			"createTicketPath":       "/tickets",
			"submitRetryMaxAttempts": 2,
		},
	}
	cfg.Id = models.Id("citc-1")
	tickets := []models.CloudItsmTicket{
		{
			ConnectorId: cfg.Id,
			ProjectId:   models.Id("p-platform"),
			Status:      models.CloudItsmTicketStatusFailed,
			Title:       "到期可重试",
			RequestPayload: models.ResAttrs{
				"title":       "到期可重试",
				"requestType": models.CloudOperationActionGitOpsIacChange,
				"params": models.ResAttrs{
					"team": models.ResAttrs{
						"id":   "team-platform",
						"name": "平台团队",
					},
				},
			},
			ResponsePayload: models.ResAttrs{
				"statusCode": 503,
				"json": models.ResAttrs{
					"error": models.ResAttrs{
						"code": "ITSM_TIMEOUT",
					},
				},
				"submitRetry": models.ResAttrs{
					"attempt":     1,
					"nextAttempt": 2,
					"nextRetryAt": now.Add(-time.Minute).Format(time.RFC3339),
				},
			},
		},
		{
			ConnectorId: cfg.Id,
			ProjectId:   models.Id("p-platform"),
			Status:      models.CloudItsmTicketStatusFailed,
			Title:       "等待退避",
			RequestPayload: models.ResAttrs{
				"title":       "等待退避",
				"requestType": models.CloudOperationActionPermissionRequest,
				"params": models.ResAttrs{
					"teamId":   "team-platform",
					"teamName": "平台团队",
				},
			},
			ResponsePayload: models.ResAttrs{
				"statusCode": 429,
				"errorCode":  "RATE_LIMIT",
				"submitRetry": models.ResAttrs{
					"attempt":     1,
					"nextAttempt": 2,
					"nextRetryAt": now.Add(time.Hour).Format(time.RFC3339),
				},
			},
		},
		{
			ConnectorId: cfg.Id,
			ProjectId:   models.Id("p-security"),
			Status:      models.CloudItsmTicketStatusFailed,
			Title:       "达到最大尝试次数",
			RequestPayload: models.ResAttrs{
				"title":       "达到最大尝试次数",
				"requestType": models.CloudOperationActionGitOpsIacChange,
				"params": models.ResAttrs{
					"team": "安全团队",
				},
			},
			ResponsePayload: models.ResAttrs{
				"statusCode": 500,
				"json": models.ResAttrs{
					"code": "MAX_ATTEMPTS",
				},
				"submitRetry": models.ResAttrs{
					"attempt":     2,
					"nextAttempt": 3,
					"deadLetter":  true,
				},
			},
		},
		{
			ConnectorId: models.Id("citc-missing"),
			Status:      models.CloudItsmTicketStatusFailed,
			Title:       "连接器缺失",
			RequestPayload: models.ResAttrs{
				"title": "连接器缺失",
			},
			ResponsePayload: models.ResAttrs{
				"submitRetry": models.ResAttrs{
					"attempt":     1,
					"nextAttempt": 2,
					"nextRetryAt": now.Add(-time.Minute).Format(time.RFC3339),
				},
			},
		},
	}
	tickets[0].UpdatedAt = models.Time(now.Add(-30 * time.Minute))
	tickets[1].UpdatedAt = models.Time(now.Add(-2 * time.Hour))
	tickets[2].UpdatedAt = models.Time(now.Add(-30 * time.Hour))
	tickets[3].UpdatedAt = models.Time(now.Add(-8 * time.Hour))

	report := cloudItsmSubmitRetryQueueReport(nil, tickets, map[models.Id]*models.CloudItsmConfig{
		cfg.Id: cfg,
	}, now)
	if len(report.ConnectorBreakdown) != 2 {
		t.Fatalf("expected connector and missing connector breakdowns, got %#v", report.ConnectorBreakdown)
	}
	jira := cloudItsmSubmitRetryQueueFindBreakdown(report.ConnectorBreakdown, cfg.Id.String())
	if jira == nil || jira.DimensionName != "Jira生产" || jira.Provider != "jira" {
		t.Fatalf("expected jira connector breakdown, got %#v", jira)
	}
	if jira.FailedTotal != 3 || jira.Due != 1 || jira.Future != 1 || jira.DeadLetter != 1 || jira.Retryable != 2 {
		t.Fatalf("unexpected jira connector counters: %#v", jira)
	}
	missing := cloudItsmSubmitRetryQueueFindBreakdown(report.ConnectorBreakdown, "citc-missing")
	if missing == nil || missing.Skipped != 1 || missing.FailedTotal != 1 {
		t.Fatalf("expected missing connector skipped breakdown, got %#v", missing)
	}
	if cloudItsmSubmitRetryQueueFindBreakdown(report.ReasonBreakdown, "due") == nil ||
		cloudItsmSubmitRetryQueueFindBreakdown(report.ReasonBreakdown, "not_due") == nil ||
		cloudItsmSubmitRetryQueueFindBreakdown(report.ReasonBreakdown, "max_attempts_reached") == nil ||
		cloudItsmSubmitRetryQueueFindBreakdown(report.ReasonBreakdown, "connector_missing") == nil {
		t.Fatalf("expected retry reason breakdowns, got %#v", report.ReasonBreakdown)
	}
	expectedAges := []string{"lt_1h", "1_6h", "6_24h", "gt_24h"}
	if len(report.AgeBuckets) != len(expectedAges) {
		t.Fatalf("expected all age buckets, got %#v", report.AgeBuckets)
	}
	for idx, expected := range expectedAges {
		if report.AgeBuckets[idx].DimensionId != expected {
			t.Fatalf("unexpected age bucket order: %#v", report.AgeBuckets)
		}
	}
	if len(report.RecentDeadLetters) != 1 || report.RecentDeadLetters[0].QueueStatus != "dead_letter" {
		t.Fatalf("expected one recent dead letter, got %#v", report.RecentDeadLetters)
	}
	platformProject := cloudItsmSubmitRetryQueueFindBreakdown(report.ProjectBreakdown, "p-platform")
	if platformProject == nil || platformProject.FailedTotal != 2 || platformProject.DimensionName != "p-platform" {
		t.Fatalf("expected platform project breakdown, got %#v", platformProject)
	}
	unassignedProject := cloudItsmSubmitRetryQueueFindBreakdown(report.ProjectBreakdown, "unassigned")
	if unassignedProject == nil || unassignedProject.FailedTotal != 1 || unassignedProject.DimensionName != "未关联项目" {
		t.Fatalf("expected unassigned project breakdown, got %#v", unassignedProject)
	}
	platformTeam := cloudItsmSubmitRetryQueueFindBreakdown(report.TeamBreakdown, "team-platform")
	if platformTeam == nil || platformTeam.FailedTotal != 2 || platformTeam.DimensionName != "平台团队" {
		t.Fatalf("expected platform team breakdown, got %#v", platformTeam)
	}
	gitopsRequestType := cloudItsmSubmitRetryQueueFindBreakdown(report.RequestTypeBreakdown, models.CloudOperationActionGitOpsIacChange)
	if gitopsRequestType == nil || gitopsRequestType.FailedTotal != 2 || gitopsRequestType.DimensionName != "GitOps/IaC 变更申请" {
		t.Fatalf("expected gitops request type breakdown, got %#v", gitopsRequestType)
	}
	if cloudItsmSubmitRetryQueueFindBreakdown(report.ErrorCodeBreakdown, "ITSM_TIMEOUT") == nil ||
		cloudItsmSubmitRetryQueueFindBreakdown(report.ErrorCodeBreakdown, "RATE_LIMIT") == nil ||
		cloudItsmSubmitRetryQueueFindBreakdown(report.ErrorCodeBreakdown, "MAX_ATTEMPTS") == nil {
		t.Fatalf("expected error code breakdowns, got %#v", report.ErrorCodeBreakdown)
	}
	http503 := cloudItsmSubmitRetryQueueFindBreakdown(report.ExternalResponseCodeBreakdown, "http_503")
	if http503 == nil || http503.FailedTotal != 1 || http503.DimensionName != "HTTP 503" {
		t.Fatalf("expected HTTP 503 breakdown, got %#v", http503)
	}
	unknownExternal := cloudItsmSubmitRetryQueueFindBreakdown(report.ExternalResponseCodeBreakdown, "unknown")
	if unknownExternal == nil || unknownExternal.FailedTotal != 1 || unknownExternal.DimensionName != "未返回" {
		t.Fatalf("expected unknown external response breakdown, got %#v", unknownExternal)
	}
}

func cloudItsmSubmitRetryQueueFindBreakdown(items []resps.CloudItsmSubmitRetryQueueBreakdownResp, id string) *resps.CloudItsmSubmitRetryQueueBreakdownResp {
	for idx := range items {
		if items[idx].DimensionId == id {
			return &items[idx]
		}
	}
	return nil
}

func TestCloudItsmDeadLetterClosePayload(t *testing.T) {
	now := time.Date(2026, 6, 23, 10, 30, 0, 0, time.UTC)
	ticket := models.CloudItsmTicket{
		ResponsePayload: models.ResAttrs{
			"submitRetry": models.ResAttrs{
				"attempt":     3,
				"nextAttempt": 4,
				"reason":      "max_attempts_reached",
			},
			"providerResponse": "bad gateway",
		},
	}
	payload, reason := cloudItsmDeadLetterClosePayload(ticket, "业务确认无需再提交", models.Id("u-1"), now)
	if reason != "业务确认无需再提交" || payload["providerResponse"] != "bad gateway" {
		t.Fatalf("unexpected close payload base fields: reason=%s payload=%#v", reason, payload)
	}
	retry := modelResAttrs(payload["submitRetry"])
	if retry["attempt"] != 3 || retry["nextAttempt"] != 4 || retry["reason"] != "max_attempts_reached" {
		t.Fatalf("expected retry state to be preserved, got %#v", retry)
	}
	if retry["deadLetter"] != true || retry["deadLetterClosed"] != true ||
		retry["deadLetterClosedBy"] != "u-1" ||
		retry["deadLetterClosedAt"] != now.Format(time.RFC3339) ||
		retry["deadLetterCloseReason"] != "业务确认无需再提交" ||
		retry["deadLetterAction"] != "close" {
		t.Fatalf("unexpected close audit fields: %#v", retry)
	}

	payload, reason = cloudItsmDeadLetterClosePayload(models.CloudItsmTicket{}, "", models.Id("u-2"), now)
	retry = modelResAttrs(payload["submitRetry"])
	if reason != "人工确认关闭死信" || retry["deadLetterCloseReason"] != "人工确认关闭死信" || retry["deadLetterClosedBy"] != "u-2" {
		t.Fatalf("expected default close reason, got reason=%s retry=%#v", reason, retry)
	}
}

func TestCloudItsmDeadLetterActionResultErrors(t *testing.T) {
	result := cloudItsmDeadLetterActionResult("close", 2)
	cloudItsmSubmitRetryAdd(result, "skipped", 1)
	cloudItsmDeadLetterActionAddError(result, models.Id("cit-1"), "not_dead_letter", "当前队列状态为 future")
	if result["action"] != "close" || result["total"] != 2 || cloudItsmSubmitRetryCount(result, "skipped") != 1 {
		t.Fatalf("unexpected result counters: %#v", result)
	}
	errors := cloudItsmStatusSyncErrors(result)
	if len(errors) != 1 || errors[0]["ticketId"] != "cit-1" || errors[0]["reason"] != "not_dead_letter" {
		t.Fatalf("unexpected result errors: %#v", errors)
	}
}

func TestCloudItsmDeadLetterApprovalPayloadHelpers(t *testing.T) {
	ids := cloudItsmDeadLetterApprovalIdsFromPayload([]models.Id{models.Id("cit-1"), models.Id("cit-2")})
	if len(ids) != 2 || ids[0] != "cit-1" || ids[1] != "cit-2" {
		t.Fatalf("expected ids from typed slice, got %#v", ids)
	}
	ids = cloudItsmDeadLetterApprovalIdsFromPayload([]interface{}{"cit-3", "cit-4"})
	if len(ids) != 2 || ids[0] != "cit-3" || ids[1] != "cit-4" {
		t.Fatalf("expected ids from json slice, got %#v", ids)
	}
	projectId, envId := cloudItsmDeadLetterApprovalScope([]models.CloudItsmTicket{
		{ProjectId: models.Id("p-1"), EnvId: models.Id("env-1")},
		{ProjectId: models.Id("p-1"), EnvId: models.Id("env-2")},
	})
	if projectId != "p-1" || envId != "" {
		t.Fatalf("expected common project and mixed env to collapse, got project=%s env=%s", projectId, envId)
	}
	if resourceId := cloudItsmDeadLetterApprovalResourceId([]models.Id{models.Id("cit-1"), models.Id("cit-2"), models.Id("cit-3")}); resourceId != "cit-1 +2" {
		t.Fatalf("unexpected resource id summary: %s", resourceId)
	}
}

func TestCloudItsmDeadLetterApprovalEvidencePayload(t *testing.T) {
	requestedAt := time.Date(2026, 6, 23, 11, 20, 0, 0, time.UTC).Format(time.RFC3339)
	form := &forms.CreateCloudItsmDeadLetterApprovalForm{
		EvidenceUrl: "https://gitlab.example.com/platform/iac/-/merge_requests/42",
		EvidenceItems: []forms.CloudItsmDeadLetterApprovalEvidenceItem{
			{
				Type:  "incident",
				Label: "事故复盘",
				Url:   "https://itsm.example.com/incidents/INC-1",
				Note:  "确认可重放",
			},
			{
				Type: "document",
				Note: "窗口期内执行",
			},
		},
		Evidence: models.ResAttrs{
			"operator": "sre",
		},
	}
	tickets := []models.CloudItsmTicket{{
		OrgId:       models.Id("org-1"),
		ProjectId:   models.Id("p-1"),
		EnvId:       models.Id("env-1"),
		ConnectorId: models.Id("citc-1"),
		ExternalKey: "ITSM-1",
		Title:       "提交失败",
		Status:      models.CloudItsmTicketStatusFailed,
		Provider:    models.CloudItsmProviderGeneric,
		ResponsePayload: models.ResAttrs{
			"statusCode": 502,
			"submitRetry": models.ResAttrs{
				"attempt":     3,
				"nextAttempt": 4,
				"reason":      "max_attempts_reached",
				"deadLetter":  true,
			},
		},
	}}

	evidence, err := cloudItsmDeadLetterApprovalEvidence(form, tickets, models.ResAttrs{"deadLetter": 1}, "replay", "重放", "修复后重放", "u-1", requestedAt)
	if err != nil {
		t.Fatalf("unexpected evidence error: %v", err)
	}
	if evidence["operator"] != "sre" || evidence["url"] != form.EvidenceUrl || evidence["itemCount"] != 3 {
		t.Fatalf("unexpected evidence summary: %#v", evidence)
	}
	items, ok := evidence["items"].([]models.ResAttrs)
	if !ok || len(items) != 3 || items[0]["type"] != "link" || items[1]["type"] != "incident" || items[2]["type"] != "document" {
		t.Fatalf("unexpected evidence items: %#v", evidence["items"])
	}
	snapshot := modelResAttrs(evidence["snapshot"])
	if snapshot["action"] != "replay" || snapshot["ticketCount"] != 1 || snapshot["requestedBy"] != "u-1" {
		t.Fatalf("unexpected snapshot header: %#v", snapshot)
	}
	ticketSnapshots, ok := snapshot["tickets"].([]models.ResAttrs)
	if !ok || len(ticketSnapshots) != 1 {
		t.Fatalf("unexpected ticket snapshots: %#v", snapshot["tickets"])
	}
	ticketSnapshot := ticketSnapshots[0]
	if ticketSnapshot["responseStatusCode"] != 502 || ticketSnapshot["retryReason"] != "max_attempts_reached" {
		t.Fatalf("unexpected ticket retry snapshot: %#v", ticketSnapshot)
	}
	errorCode := modelResAttrs(ticketSnapshot["errorCode"])
	externalResponseCode := modelResAttrs(ticketSnapshot["externalResponseCode"])
	if errorCode["id"] != "http_502" || externalResponseCode["id"] != "http_502" {
		t.Fatalf("unexpected code dimensions: error=%#v external=%#v", errorCode, externalResponseCode)
	}
	strategy := cloudItsmDeadLetterApprovalNotificationStrategy("replay", 1, evidence)
	if strategy["enabled"] != true || strategy["severity"] != models.CloudEventLevelError || strategy["includeEvidenceSnapshot"] != true || strategy["evidenceItemCount"] != 3 {
		t.Fatalf("unexpected notification strategy: %#v", strategy)
	}
	events, ok := strategy["eventTypes"].([]string)
	if !ok || len(events) != 4 || events[0] != "itsm.dead_letter.approval_requested" {
		t.Fatalf("unexpected notification events: %#v", strategy["eventTypes"])
	}
}

func TestCloudItsmDeadLetterApprovalEvidenceRejectsInvalidURL(t *testing.T) {
	_, err := cloudItsmDeadLetterApprovalEvidenceItems("", []forms.CloudItsmDeadLetterApprovalEvidenceItem{{
		Type: "link",
		Url:  "ftp://example.com/evidence",
	}})
	if err == nil {
		t.Fatalf("expected invalid evidence url to be rejected")
	}
	tooMany := make([]forms.CloudItsmDeadLetterApprovalEvidenceItem, 0, cloudItsmDeadLetterApprovalEvidenceItemLimit+1)
	for i := 0; i < cloudItsmDeadLetterApprovalEvidenceItemLimit+1; i++ {
		tooMany = append(tooMany, forms.CloudItsmDeadLetterApprovalEvidenceItem{Note: "evidence"})
	}
	if _, err := cloudItsmDeadLetterApprovalEvidenceItems("", tooMany); err == nil {
		t.Fatalf("expected evidence item limit to be enforced")
	}
}

func TestCloudItsmForcedDeadLetterReplayEligibility(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	cfg := models.CloudItsmConfig{
		BaseUrl: "https://itsm.example.com",
		Metadata: models.ResAttrs{
			"createTicketPath":       "/tickets",
			"submitRetryMaxAttempts": 2,
		},
	}
	ticket := models.CloudItsmTicket{
		Status: models.CloudItsmTicketStatusFailed,
		RequestPayload: models.ResAttrs{
			"title": "失败提交",
		},
		ResponsePayload: models.ResAttrs{
			"submitRetry": models.ResAttrs{
				"attempt":     2,
				"nextAttempt": 3,
				"deadLetter":  true,
			},
		},
	}
	eligibility := cloudItsmTicketSubmitRetryEligibility(cfg, ticket, now, true)
	if eligibility.Eligible || !eligibility.DeadLetter || eligibility.Reason != "max_attempts_reached" {
		t.Fatalf("expected regular retry to keep dead letter closed, got %#v", eligibility)
	}
	if eligibility.DeadLetter && eligibility.Reason == "max_attempts_reached" {
		eligibility.Eligible = true
		eligibility.Reason = "forced_dead_letter_replay"
		eligibility.NextAttempt = eligibility.Attempt + 1
	}
	if !eligibility.Eligible || eligibility.NextAttempt != 3 || eligibility.Reason != "forced_dead_letter_replay" {
		t.Fatalf("expected forced dead letter replay eligibility, got %#v", eligibility)
	}
}

func TestCloudItsmRobotProcessingForSelfService(t *testing.T) {
	c := &ctx.ServiceContext{
		UserId:   models.Id("u-1"),
		Username: "alice",
		Email:    "alice@example.com",
	}
	cfg := models.CloudItsmConfig{
		Name:       "本地工单",
		Provider:   models.CloudItsmProviderGeneric,
		ProjectKey: "CLOUD",
		TicketType: "service_request",
	}
	params := models.ResAttrs{
		"requestType":        models.CloudOperationActionGitOpsIacChange,
		"catalogKey":         models.CloudOperationActionGitOpsIacChange,
		"automationMode":     "itsm",
		"gitOpsRequired":     true,
		"gateStatus":         cloudItsmGitOpsGateStatusPassed,
		"gitOpsRepository":   "gitlab.example.com/platform/iac",
		"gitOpsBranch":       "feature/change",
		"gitOpsTargetBranch": "main",
	}
	item, ok := cloudItsmCatalogItemByKey(nil, models.CloudOperationActionGitOpsIacChange)
	if !ok {
		t.Fatalf("expected gitops catalog item")
	}
	cloudItsmApplyRobotProcessingToParams(params, cloudItsmRobotProcessingForSelfService(item, cfg, params))
	if !attrBool(params, "robotProcessed") {
		t.Fatalf("expected robot processed params, got %#v", params)
	}
	operation := models.CloudOperation{
		Name:          "GitOps/IaC 变更申请",
		OperationType: models.CloudOperationTypeSelfService,
		Action:        models.CloudOperationActionGitOpsIacChange,
		Status:        models.CloudOperationStatusRunning,
		RiskLevel:     models.CloudOperationRiskHigh,
		Params:        params,
	}
	payload := cloudItsmTicketRequestPayload(c, operation, cfg, "GitOps/IaC 变更申请", "PR review passed", "high")
	robotPayload := modelResAttrs(payload["robot"])
	if !attrBool(robotPayload, "processed") || robotPayload["requestType"] != models.CloudOperationActionGitOpsIacChange {
		t.Fatalf("unexpected robot payload: %#v", robotPayload)
	}
	tags := cloudItsmRobotProcessingTags(robotPayload["tags"])
	for _, tag := range []string{cloudItsmRobotTagSelfService, cloudItsmRobotTagGitOpsIac, cloudItsmRobotTagLocalTicket} {
		if !cloudItsmTestStringSliceContains(tags, tag) {
			t.Fatalf("expected robot tag %s in %#v", tag, tags)
		}
	}
	resp := cloudItsmTicketResp(c, models.CloudItsmTicket{
		RequestPayload: payload,
	}, nil)
	if !resp.RobotProcessed || resp.RobotProcessor != cloudItsmRobotProcessorCloudiac || resp.RobotAutomationMode != "itsm" {
		t.Fatalf("unexpected robot response: %#v", resp)
	}
	if !cloudItsmTestStringSliceContains(resp.RobotProcessingTags, cloudItsmRobotTagGitOpsIac) {
		t.Fatalf("expected response robot tags, got %#v", resp.RobotProcessingTags)
	}
}

func TestCloudItsmCatalogPolicyForSelfService(t *testing.T) {
	item, ok := cloudItsmCatalogItemByKey(nil, models.CloudOperationActionGitOpsIacChange)
	if !ok {
		t.Fatalf("expected gitops catalog item")
	}
	if item.PolicyKey != "gitops_iac_change_policy" || item.PolicyName == "" {
		t.Fatalf("expected gitops policy snapshot, got %#v", item)
	}
	if item.SlaMinutes != 1440 || item.SlaDescription == "" {
		t.Fatalf("expected gitops SLA snapshot, got %#v", item)
	}
	if !cloudItsmTestStringSliceContains(item.RequiredRoles, "iac_reviewer") ||
		!cloudItsmTestStringSliceContains(item.AllowedScopes, "env") {
		t.Fatalf("expected gitops role/scope policy, got roles=%#v scopes=%#v", item.RequiredRoles, item.AllowedScopes)
	}

	permission, ok := cloudItsmCatalogItemByKey(nil, models.CloudOperationActionPermissionRequest)
	if !ok || permission.PolicyKey != "self_service_permission_policy" || permission.SlaMinutes != 480 {
		t.Fatalf("expected permission policy/SLA, got %#v", permission)
	}
}

func TestCloudItsmCatalogPolicyOverride(t *testing.T) {
	base, ok := cloudItsmCatalogItemByKey(nil, models.CloudOperationActionPermissionRequest)
	if !ok {
		t.Fatalf("expected permission catalog item")
	}
	item := cloudItsmApplyCatalogPolicyOverride(base, models.ResAttrs{
		"enabled":           false,
		"policyName":        "组织权限申请策略",
		"policyDescription": "组织级权限申请策略覆盖",
		"requiredRoles":     []string{"operator", "operator", "project_owner"},
		"allowedScopes":     []string{"org", "asset", "asset"},
		"slaMinutes":        360,
		"policyVersion":     3,
		"updatedAt":         "2026-06-23T10:00:00+08:00",
		"updatedBy":         "u-1",
		"policyLastDiff": []resps.CloudItsmCatalogPolicyDiffResp{
			{
				Field:  "enabled",
				Label:  "策略状态",
				Before: true,
				After:  false,
			},
		},
	})
	if item.Available || item.Enabled || item.DisabledReason != "策略已停用" {
		t.Fatalf("expected disabled catalog policy override, got %#v", item)
	}
	if !item.PolicyConfigured || item.PolicySource != "org_config" {
		t.Fatalf("expected org policy source, got configured=%v source=%s", item.PolicyConfigured, item.PolicySource)
	}
	if item.PolicyName != "组织权限申请策略" || item.PolicyDescription == "" {
		t.Fatalf("expected overridden policy text, got %#v", item)
	}
	if item.SlaMinutes != 360 || item.SlaDescription != cloudItsmSlaDescription(360) {
		t.Fatalf("expected overridden SLA, got %#v", item)
	}
	if len(item.RequiredRoles) != 2 ||
		item.RequiredRoles[0] != "operator" ||
		item.RequiredRoles[1] != "project_owner" {
		t.Fatalf("expected normalized roles, got %#v", item.RequiredRoles)
	}
	if len(item.AllowedScopes) != 2 ||
		!cloudItsmTestStringSliceContains(item.AllowedScopes, "org") ||
		!cloudItsmTestStringSliceContains(item.AllowedScopes, "asset") {
		t.Fatalf("expected normalized scopes, got %#v", item.AllowedScopes)
	}
	if item.PolicyVersion != 3 || item.PolicyUpdatedBy != "u-1" || item.PolicyUpdatedAt == "" {
		t.Fatalf("expected policy metadata, got %#v", item)
	}
	if len(item.PolicyLastDiff) != 1 || item.PolicyLastDiff[0].Field != "enabled" {
		t.Fatalf("expected policy last diff, got %#v", item.PolicyLastDiff)
	}
}

func TestCloudItsmCatalogPolicyHistoryDiff(t *testing.T) {
	before := models.ResAttrs{
		"enabled":           true,
		"policyName":        "默认策略",
		"policyDescription": "默认说明",
		"requiredRoles":     []string{"project_member"},
		"allowedScopes":     []string{"project", "env"},
		"slaMinutes":        240,
	}
	after := models.ResAttrs{
		"enabled":           false,
		"policyName":        "组织策略",
		"policyDescription": "默认说明",
		"requiredRoles":     []string{"operator", "project_owner"},
		"allowedScopes":     []string{"asset"},
		"slaMinutes":        120,
	}
	entry := cloudItsmCatalogPolicyHistoryEntry("permission_request", "update", 2, "2026-06-23T10:00:00+08:00", "u-1", before, after)
	resp := cloudItsmCatalogPolicyHistoryResp(entry)
	if resp.Version != 2 || resp.Action != "update" || resp.PolicyName != "组织策略" || resp.UpdatedBy != "u-1" {
		t.Fatalf("unexpected history response: %#v", resp)
	}
	expectedFields := []string{"enabled", "policyName", "requiredRoles", "allowedScopes", "slaMinutes"}
	for _, field := range expectedFields {
		found := false
		for _, diff := range resp.Diff {
			if diff.Field == field && diff.Label != "" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected diff field %s in %#v", field, resp.Diff)
		}
	}
	history := cloudItsmAppendCatalogPolicyHistory(nil, "permission_request", entry)
	if next := cloudItsmCatalogPolicyNextVersion(history, "permission_request", nil); next != 3 {
		t.Fatalf("expected next version 3, got %d", next)
	}
}

func TestCloudItsmSelfServicePolicyPayload(t *testing.T) {
	c := &ctx.ServiceContext{
		UserId:   models.Id("u-1"),
		Username: "alice",
		Email:    "alice@example.com",
	}
	cfg := models.CloudItsmConfig{
		Name:       "本地工单",
		Provider:   models.CloudItsmProviderGeneric,
		ProjectKey: "CLOUD",
		TicketType: "service_request",
	}
	item, ok := cloudItsmCatalogItemByKey(nil, models.CloudOperationActionGitOpsIacChange)
	if !ok {
		t.Fatalf("expected gitops catalog item")
	}
	params := models.ResAttrs{
		"requestType": models.CloudOperationActionGitOpsIacChange,
		"catalogKey":  models.CloudOperationActionGitOpsIacChange,
	}
	cloudItsmApplySelfServicePolicyToParams(params, item)
	policy := modelResAttrs(params["selfServicePolicy"])
	if policy["key"] != "gitops_iac_change_policy" {
		t.Fatalf("expected policy snapshot in params, got %#v", policy)
	}
	sla := modelResAttrs(params["sla"])
	if cloudSyncPolicyAttrInt(sla["minutes"]) != 1440 || cloudSyncPolicyAttrInt(sla["target"]) != cloudItsmDefaultSlaTarget {
		t.Fatalf("expected SLA snapshot in params, got %#v", sla)
	}

	operation := models.CloudOperation{
		Name:          "GitOps/IaC 变更申请",
		OperationType: models.CloudOperationTypeSelfService,
		Action:        models.CloudOperationActionGitOpsIacChange,
		Status:        models.CloudOperationStatusRunning,
		RiskLevel:     models.CloudOperationRiskHigh,
		Params:        params,
	}
	payload := cloudItsmTicketRequestPayload(c, operation, cfg, "GitOps/IaC 变更申请", "PR review passed", "high")
	payloadPolicy := modelResAttrs(payload["selfServicePolicy"])
	if payloadPolicy["key"] != "gitops_iac_change_policy" {
		t.Fatalf("expected top-level policy snapshot, got %#v", payloadPolicy)
	}
	payloadSla := modelResAttrs(payload["sla"])
	if cloudSyncPolicyAttrInt(payloadSla["minutes"]) != 1440 {
		t.Fatalf("expected top-level SLA snapshot, got %#v", payloadSla)
	}
	if cloudItsmSlaMinutesFromPayload(payload) != 1440 {
		t.Fatalf("expected SLA minutes recoverable from payload, got %#v", payload)
	}
}

func TestCloudItsmTicketSlaResult(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	payload := models.ResAttrs{
		"sla": models.ResAttrs{
			"minutes": 60,
		},
	}

	metTicket := models.CloudItsmTicket{
		Status:         models.CloudItsmTicketStatusResolved,
		RequestPayload: payload,
	}
	metTicket.CreatedAt = models.Time(now.Add(-50 * time.Minute))
	metTicket.ClosedAt = models.Time(now.Add(-10 * time.Minute))
	if result := cloudItsmTicketSlaResult(metTicket, now); !result.Counted || !result.Met || result.Breached {
		t.Fatalf("expected SLA met, got %#v", result)
	}

	breachedTicket := models.CloudItsmTicket{
		Status:         models.CloudItsmTicketStatusClosed,
		RequestPayload: payload,
	}
	breachedTicket.CreatedAt = models.Time(now.Add(-2 * time.Hour))
	breachedTicket.ClosedAt = models.Time(now.Add(-10 * time.Minute))
	if result := cloudItsmTicketSlaResult(breachedTicket, now); !result.Counted || result.Met || !result.Breached {
		t.Fatalf("expected SLA breached, got %#v", result)
	}

	openTicket := models.CloudItsmTicket{
		Status:         models.CloudItsmTicketStatusInProgress,
		RequestPayload: payload,
	}
	openTicket.CreatedAt = models.Time(now.Add(-2 * time.Hour))
	if result := cloudItsmTicketSlaResult(openTicket, now); !result.Counted || result.Met || !result.Breached {
		t.Fatalf("expected open breached SLA, got %#v", result)
	}

	inWindowTicket := models.CloudItsmTicket{
		Status:         models.CloudItsmTicketStatusInProgress,
		RequestPayload: payload,
	}
	inWindowTicket.CreatedAt = models.Time(now.Add(-30 * time.Minute))
	if result := cloudItsmTicketSlaResult(inWindowTicket, now); !result.Counted || result.Met || result.Breached {
		t.Fatalf("expected open ticket still in SLA window, got %#v", result)
	}

	noSlaTicket := models.CloudItsmTicket{
		Status: models.CloudItsmTicketStatusResolved,
	}
	noSlaTicket.CreatedAt = models.Time(now.Add(-30 * time.Minute))
	if result := cloudItsmTicketSlaResult(noSlaTicket, now); result.Counted {
		t.Fatalf("expected non-SLA ticket ignored, got %#v", result)
	}
}

func TestCloudItsmDimensionMetricAddTicket(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	metric := &resps.CloudItsmDimensionMetricResp{
		DimensionType:             "project",
		DimensionId:               "p-1",
		DimensionName:             "核心业务",
		SelfServiceCoverageTarget: 80,
		TicketAutomationTarget:    60,
		SelfServiceSlaTarget:      90,
		WindowDays:                30,
	}
	metPayload := models.ResAttrs{
		"operation": models.ResAttrs{
			"operationType": models.CloudOperationTypeSelfService,
			"action":        models.CloudOperationActionGitOpsIacChange,
			"params": models.ResAttrs{
				"selfService": true,
			},
		},
		"robotProcessed": true,
		"sla": models.ResAttrs{
			"minutes": 60,
		},
	}
	metTicket := models.CloudItsmTicket{
		Status:         models.CloudItsmTicketStatusResolved,
		RequestPayload: metPayload,
	}
	metTicket.CreatedAt = models.Time(now.Add(-50 * time.Minute))
	metTicket.ClosedAt = models.Time(now.Add(-5 * time.Minute))

	manualTicket := models.CloudItsmTicket{
		Status: models.CloudItsmTicketStatusPending,
		RequestPayload: models.ResAttrs{
			"operation": models.ResAttrs{
				"operationType": models.CloudOperationTypeAction,
				"action":        models.CloudOperationActionStartInstance,
			},
		},
	}
	manualTicket.CreatedAt = models.Time(now.Add(-20 * time.Minute))

	breachedPayload := models.ResAttrs{
		"operation": models.ResAttrs{
			"operationType": models.CloudOperationTypeSelfService,
			"action":        models.CloudOperationActionDriftRemediation,
			"params": models.ResAttrs{
				"selfService": true,
			},
		},
		"robot": models.ResAttrs{
			"processed": true,
		},
		"sla": models.ResAttrs{
			"minutes": 30,
		},
	}
	breachedTicket := models.CloudItsmTicket{
		Status:         models.CloudItsmTicketStatusInProgress,
		RequestPayload: breachedPayload,
	}
	breachedTicket.CreatedAt = models.Time(now.Add(-2 * time.Hour))

	cloudItsmDimensionMetricAddTicket(metric, metTicket, now)
	cloudItsmDimensionMetricAddTicket(metric, manualTicket, now)
	cloudItsmDimensionMetricAddTicket(metric, breachedTicket, now)

	if metric.TicketTotal != 3 || metric.SelfServiceTicketTotal != 2 {
		t.Fatalf("expected 2/3 self service tickets, got %#v", metric)
	}
	if metric.AutomatedTicketTotal != 2 || metric.RobotProcessedTicketTotal != 2 {
		t.Fatalf("expected 2 automated robot tickets, got %#v", metric)
	}
	if metric.SelfServiceSlaTicketTotal != 2 || metric.SelfServiceSlaMetTotal != 1 || metric.SelfServiceSlaBreached != 1 {
		t.Fatalf("expected SLA 1 met / 1 breached, got %#v", metric)
	}
	if metric.SelfServiceCoverageRate < 66 || metric.SelfServiceCoverageRate > 67 {
		t.Fatalf("expected self service coverage around 66.7, got %#v", metric.SelfServiceCoverageRate)
	}
	if metric.TicketAutomationRate < 66 || metric.TicketAutomationRate > 67 {
		t.Fatalf("expected automation rate around 66.7, got %#v", metric.TicketAutomationRate)
	}
	if metric.SelfServiceSlaMetRate != 50 {
		t.Fatalf("expected SLA met rate 50, got %#v", metric.SelfServiceSlaMetRate)
	}
}

func TestCloudItsmTicketTeamDimension(t *testing.T) {
	explicitTeamTicket := models.CloudItsmTicket{
		RequestPayload: models.ResAttrs{
			"operation": models.ResAttrs{
				"params": models.ResAttrs{
					"team": models.ResAttrs{
						"id":   "team-platform",
						"name": "平台团队",
					},
				},
			},
		},
	}
	explicitTeam := cloudItsmTicketTeamDimension(nil, explicitTeamTicket, "p-1", "核心项目", nil)
	if explicitTeam.Id != "team-platform" || explicitTeam.Name != "平台团队" {
		t.Fatalf("expected explicit team dimension, got %#v", explicitTeam)
	}

	departmentTicket := models.CloudItsmTicket{
		RequestPayload: models.ResAttrs{
			"requester": models.ResAttrs{
				"department": "SRE 平台部",
			},
		},
	}
	departmentTeam := cloudItsmTicketTeamDimension(nil, departmentTicket, "", "", nil)
	if departmentTeam.Id != "SRE 平台部" || departmentTeam.Name != "SRE 平台部" {
		t.Fatalf("expected requester department team, got %#v", departmentTeam)
	}

	projectFallbackTicket := models.CloudItsmTicket{}
	projectTeam := cloudItsmTicketTeamDimension(nil, projectFallbackTicket, "p-prod", "生产业务项目", nil)
	if projectTeam.Id != "project:p-prod" || projectTeam.Name != "生产业务项目" {
		t.Fatalf("expected project fallback team, got %#v", projectTeam)
	}

	unassignedTeam := cloudItsmTicketTeamDimension(nil, models.CloudItsmTicket{}, "", "未关联项目", nil)
	if unassignedTeam.Id != "unassigned" || unassignedTeam.Name != "未标记团队" {
		t.Fatalf("expected unassigned team fallback, got %#v", unassignedTeam)
	}
}

func TestNormalizeCloudItsmSelfServiceGitOpsGate(t *testing.T) {
	params := models.ResAttrs{
		"gitOpsRepository":     "gitlab.example.com/platform/iac",
		"gitOpsBranch":         "feature/network-change",
		"gitOpsTargetBranch":   "main",
		"gitOpsPullRequestUrl": "https://gitlab.example.com/platform/iac/-/merge_requests/18",
		"gitOpsReviewStatus":   "approved",
		"gitOpsPipelineUrl":    "https://gitlab.example.com/platform/iac/-/pipelines/42",
		"gitOpsPipelineStatus": "success",
	}
	normalized, err := normalizeCloudItsmSelfServiceParams(models.CloudOperationActionGitOpsIacChange, params)
	if err != nil {
		t.Fatalf("unexpected normalize error: %v", err)
	}
	gate := modelResAttrs(normalized["gitOpsGate"])
	if gate["status"] != cloudItsmGitOpsGateStatusPassed {
		t.Fatalf("expected passed gitops gate, got %#v", gate)
	}
	if gate["reviewStatus"] != "approved" || gate["pipelineStatus"] != "passed" {
		t.Fatalf("expected normalized review and pipeline statuses, got %#v", gate)
	}
	if normalized["gateStatus"] != cloudItsmGitOpsGateStatusPassed {
		t.Fatalf("expected top-level gate status, got %#v", normalized["gateStatus"])
	}
}

func TestNormalizeCloudItsmSelfServiceGitOpsGateRequiresPullRequest(t *testing.T) {
	_, err := normalizeCloudItsmSelfServiceParams(models.CloudOperationActionGitOpsIacChange, models.ResAttrs{
		"gitOpsReviewStatus":   "approved",
		"gitOpsPipelineStatus": "passed",
	})
	if err == nil {
		t.Fatalf("expected missing pull request url error")
	}
}

func TestCloudItsmGitOpsGateStatus(t *testing.T) {
	cases := []struct {
		name            string
		reviewStatus    string
		pipelineStatus  string
		pipelinePresent bool
		want            string
	}{
		{name: "passed", reviewStatus: "approved", pipelineStatus: "passed", pipelinePresent: true, want: cloudItsmGitOpsGateStatusPassed},
		{name: "waiting review", reviewStatus: "pending", pipelineStatus: "passed", pipelinePresent: true, want: cloudItsmGitOpsGateStatusWaiting},
		{name: "waiting pipeline url", reviewStatus: "approved", pipelineStatus: "passed", pipelinePresent: false, want: cloudItsmGitOpsGateStatusWaiting},
		{name: "blocked review", reviewStatus: "changes_requested", pipelineStatus: "passed", pipelinePresent: true, want: cloudItsmGitOpsGateStatusBlocked},
		{name: "blocked pipeline", reviewStatus: "approved", pipelineStatus: "failed", pipelinePresent: true, want: cloudItsmGitOpsGateStatusBlocked},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cloudItsmGitOpsGateStatus(tc.reviewStatus, tc.pipelineStatus, tc.pipelinePresent); got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}
