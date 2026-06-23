package apps

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"cloudiac/portal/models"
)

func TestOciAPIErrorIncludesNativeMetadata(t *testing.T) {
	err := newOciAPIError(&http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     "429 Too Many Requests",
		Header: http.Header{
			"Opc-Request-Id": []string{"req-1"},
			"Retry-After":    []string{"30"},
		},
	}, "iaas", "/20160918/instances", []byte(`{"code":"TooManyRequests","message":"Rate exceeded"}`))
	err.Attempts = 3

	message := err.Error()
	for _, expected := range []string{
		"provider=oci",
		"status=429",
		"code=TooManyRequests",
		"requestId=req-1",
		"retryAfter=30",
		"service=iaas",
		"path=/20160918/instances",
		"attempts=3",
		"Rate exceeded",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected %q in %q", expected, message)
		}
	}
}

func TestCmdbSyncFailureDetailsExtractsProviderMetadata(t *testing.T) {
	err := newOciAPIError(&http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     "429 Too Many Requests",
		Header: http.Header{
			"Opc-Request-Id": []string{"req-1"},
			"Retry-After":    []string{"30"},
		},
	}, "iaas", "/20160918/instances", []byte(`{"code":"TooManyRequests","message":"Rate exceeded"}`))
	err.Attempts = 3

	details := cmdbSyncFailureDetails([]string{err.Error()})
	if len(details) != 1 {
		t.Fatalf("expected one failure detail, got %#v", details)
	}
	detail := details[0]
	if detail["category"] != "rate_limit" {
		t.Fatalf("expected rate_limit category, got %#v", detail["category"])
	}
	if detail["retryable"] != true {
		t.Fatalf("expected retryable detail, got %#v", detail["retryable"])
	}
	if detail["provider"] != "oci" {
		t.Fatalf("expected oci provider, got %#v", detail["provider"])
	}
	if detail["httpStatus"] != http.StatusTooManyRequests {
		t.Fatalf("expected http status 429, got %#v", detail["httpStatus"])
	}
	if detail["providerCode"] != "TooManyRequests" {
		t.Fatalf("expected provider code, got %#v", detail["providerCode"])
	}
	if detail["requestId"] != "req-1" {
		t.Fatalf("expected request id, got %#v", detail["requestId"])
	}
	if detail["retryAfter"] != "30" {
		t.Fatalf("expected retry after, got %#v", detail["retryAfter"])
	}
	if detail["providerAttempts"] != 3 {
		t.Fatalf("expected provider attempts, got %#v", detail["providerAttempts"])
	}

	details = cmdbSyncFailureDetails([]string{"provider=oci region=ap-singapore-1 assetType=instance status=429 code=TooManyRequests message=rate"})
	if len(details) != 1 {
		t.Fatalf("expected one scoped failure detail, got %#v", details)
	}
	if details[0]["region"] != "ap-singapore-1" || details[0]["assetType"] != "instance" {
		t.Fatalf("expected scoped metadata, got %#v", details[0])
	}
}

func TestCmdbSyncClassifyOciNativeCodes(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		category  string
		retryable bool
	}{
		{
			name:      "permission",
			message:   "provider=oci status=403 code=NotAuthorizedOrNotFound message=not allowed",
			category:  "permission",
			retryable: false,
		},
		{
			name:      "credential",
			message:   "provider=oci status=401 code=NotAuthenticated message=bad signature",
			category:  "credential",
			retryable: false,
		},
		{
			name:      "server error",
			message:   "provider=oci status=500 code=InternalError message=temporary unavailable",
			category:  "network",
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, retryable, _ := cmdbSyncClassifyFailure(tt.message)
			if category != tt.category || retryable != tt.retryable {
				t.Fatalf("expected %s/%v, got %s/%v", tt.category, tt.retryable, category, retryable)
			}
			accountCategory, accountRetryable, _ := cloudAccountClassifySyncFailure(tt.message)
			if accountCategory != tt.category || accountRetryable != tt.retryable {
				t.Fatalf("cloud account classifier expected %s/%v, got %s/%v", tt.category, tt.retryable, accountCategory, accountRetryable)
			}
		})
	}
}

func TestOciAPIRetryPolicy(t *testing.T) {
	if !ociAPIShouldRetry(http.StatusTooManyRequests, nil) {
		t.Fatal("expected 429 to be retryable")
	}
	if !ociAPIShouldRetry(http.StatusBadGateway, nil) {
		t.Fatal("expected 502 to be retryable")
	}
	if ociAPIShouldRetry(http.StatusForbidden, nil) {
		t.Fatal("expected 403 to be non-retryable")
	}
	if ociAPIShouldRetry(0, context.Canceled) {
		t.Fatal("expected context cancellation to be non-retryable")
	}
	if ociAPIShouldRetry(0, context.DeadlineExceeded) {
		t.Fatal("expected deadline exceeded from HTTP client to be non-retryable")
	}
}

func TestOciAPIRetryDelay(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	if got := ociAPIRetryDelay("2", 1, now); got != 2*time.Second {
		t.Fatalf("expected Retry-After seconds, got %s", got)
	}
	retryAt := now.Add(3 * time.Second).Format(http.TimeFormat)
	if got := ociAPIRetryDelay(retryAt, 1, now); got != 3*time.Second {
		t.Fatalf("expected Retry-After date, got %s", got)
	}
	if got := ociAPIRetryDelay("", 3, now); got != 2*time.Second {
		t.Fatalf("expected exponential delay, got %s", got)
	}
	if got := ociAPIRetryDelay("120", 1, now); got != ociAPIRequestMaxRetryDelay {
		t.Fatalf("expected capped retry delay, got %s", got)
	}
}

func TestOciAPIMetricRecorderAndSummary(t *testing.T) {
	recorder := &ociAPIMetricsRecorder{}
	ctx := context.WithValue(context.Background(), ociAPIMetricsContextKey{}, recorder)
	apiErr := &ociAPIError{
		StatusCode: http.StatusTooManyRequests,
		Code:       "TooManyRequests",
		Message:    "rate exceeded",
		RequestId:  "req-1",
		RetryAfter: "1",
		Service:    "iaas",
		Path:       "/20160918/instances",
		Attempts:   2,
	}
	metric := ociBuildAPIMetric(
		"ap-singapore-1",
		"iaas",
		"/20160918/instances",
		url.Values{"compartmentId": []string{"ocid1.compartment"}, "page": []string{"token-1"}},
		time.Now().Add(-10*time.Millisecond),
		2,
		http.StatusTooManyRequests,
		"req-1",
		"1",
		apiErr,
	)
	ociRecordAPIMetric(ctx, metric)

	items := recorder.List()
	if len(items) != 1 {
		t.Fatalf("expected one api metric, got %#v", items)
	}
	item := items[0]
	if item["status"] != "failed" || item["errorCategory"] != "rate_limit" || item["retryable"] != true {
		t.Fatalf("unexpected failure metric: %#v", item)
	}
	if item["retryCount"] != 1 || item["attempts"] != 2 {
		t.Fatalf("expected retry count and attempts, got %#v", item)
	}
	if item["pageTokenUsed"] != true || item["compartmentId"] != "ocid1.compartment" {
		t.Fatalf("expected query scope metadata, got %#v", item)
	}

	summary := ociAPIMetricSummary(map[string][]models.ResAttrs{"ap-singapore-1": items})
	if summary["total"] != 1 || summary["failed"] != 1 || summary["retried"] != 1 || summary["maxAttempts"] != 2 {
		t.Fatalf("unexpected api metric summary: %#v", summary)
	}
	serviceCounts, ok := summary["serviceCounts"].(map[string]int)
	if !ok || serviceCounts["iaas"] != 1 {
		t.Fatalf("expected service count for iaas, got %#v", summary["serviceCounts"])
	}
	if !strings.Contains(summary["slowestEndpoint"].(string), "/20160918/instances") {
		t.Fatalf("expected slowest endpoint, got %#v", summary["slowestEndpoint"])
	}
}

func TestCmdbSyncTaskAPIMetricAttrsList(t *testing.T) {
	stats := models.ResAttrs{
		"apiMetrics": map[string]interface{}{
			"ap-singapore-1": []interface{}{
				map[string]interface{}{
					"provider":   "oci",
					"service":    "iaas",
					"path":       "/20160918/instances",
					"durationMs": float64(1200),
					"attempts":   float64(2),
					"retryCount": float64(1),
					"status":     "failed",
				},
			},
		},
	}

	items := cmdbSyncTaskAPIMetricAttrsList(stats)
	if len(items) != 1 {
		t.Fatalf("expected one api metric, got %#v", items)
	}
	item := items[0]
	if item["region"] != "ap-singapore-1" {
		t.Fatalf("expected region to be injected, got %#v", item)
	}
	if cmdbSyncTaskMetricInt64(item["durationMs"]) != 1200 {
		t.Fatalf("expected numeric duration, got %#v", item["durationMs"])
	}
	if cmdbSyncTaskMetricText(item["path"]) != "/20160918/instances" {
		t.Fatalf("expected metric path, got %#v", item["path"])
	}
}

func TestCmdbSyncTaskAPIMetricTrendList(t *testing.T) {
	rangeStart := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	trendRange := cmdbSyncTaskTrendRange{
		Start: rangeStart,
		End:   rangeStart.AddDate(0, 0, 2),
		Days:  3,
	}
	acc := map[string]*cmdbSyncTaskAPIMetricTrendAccumulator{
		"2026-06-21": {
			Date:            "2026-06-21",
			CallCount:       2,
			FailedCount:     1,
			RetriedCount:    1,
			TotalDurationMs: 3000,
			MaxDurationMs:   2000,
		},
	}

	trend := cmdbSyncTaskAPIMetricTrendList(acc, trendRange)
	if len(trend) != 3 {
		t.Fatalf("expected three trend points, got %#v", trend)
	}
	if trend[0].Date != "2026-06-20" || trend[0].CallCount != 0 {
		t.Fatalf("expected empty first day, got %#v", trend[0])
	}
	if trend[1].CallCount != 2 || trend[1].FailedCount != 1 || trend[1].RetriedCount != 1 {
		t.Fatalf("unexpected populated day, got %#v", trend[1])
	}
	if trend[1].AvgDurationMs != 1500 || trend[1].MaxDurationMs != 2000 {
		t.Fatalf("unexpected duration stats, got %#v", trend[1])
	}
}

func TestCmdbSyncTaskSlowAPISummary(t *testing.T) {
	stats := models.ResAttrs{
		"apiMetrics": []interface{}{
			map[string]interface{}{
				"provider":   "oci",
				"region":     "ap-singapore-1",
				"service":    "iaas",
				"path":       "/20160918/instances",
				"durationMs": float64(2500),
				"attempts":   float64(2),
				"retryCount": float64(1),
				"status":     "failed",
				"requestId":  "req-slow",
			},
			map[string]interface{}{
				"provider":   "oci",
				"region":     "ap-singapore-1",
				"service":    "identity",
				"path":       "/20160918/compartments",
				"durationMs": float64(1200),
				"attempts":   float64(1),
				"status":     "complete",
			},
			map[string]interface{}{
				"provider":   "oci",
				"region":     "ap-singapore-1",
				"service":    "objectstorage",
				"path":       "/n/namespace/b",
				"durationMs": float64(900),
				"attempts":   float64(1),
				"status":     "complete",
			},
		},
	}

	summary, ok := cmdbSyncTaskSlowAPISummary(stats, 1000)
	if !ok {
		t.Fatal("expected slow api summary")
	}
	if cmdbSyncTaskMetricInt64(summary["slowCount"]) != 2 {
		t.Fatalf("expected two slow calls, got %#v", summary["slowCount"])
	}
	if cmdbSyncTaskMetricInt64(summary["failedCount"]) != 1 || cmdbSyncTaskMetricInt64(summary["retriedCount"]) != 1 {
		t.Fatalf("unexpected failed/retried counts: %#v", summary)
	}
	slowest := attrResAttrs(summary, "slowest")
	if cmdbSyncTaskMetricText(slowest["path"]) != "/20160918/instances" {
		t.Fatalf("expected slowest instances endpoint, got %#v", slowest)
	}
	if slowest["requestId"] != "req-slow" {
		t.Fatalf("expected request id to be kept, got %#v", slowest)
	}
	top := cmdbSyncTaskMetricAttrsList(summary["top"])
	if len(top) != 2 {
		t.Fatalf("expected two top slow calls, got %#v", top)
	}
	if cmdbSyncTaskMetricText(top[0]["path"]) != "/20160918/instances" {
		t.Fatalf("expected top list sorted by duration, got %#v", top)
	}
}

func TestCmdbSyncTaskSlowAPISummaryUsesPersistedThreshold(t *testing.T) {
	stats := models.ResAttrs{
		"slowApiThresholdMs": float64(2000),
		"apiMetrics": []interface{}{
			map[string]interface{}{
				"provider":   "oci",
				"region":     "ap-singapore-1",
				"service":    "identity",
				"path":       "/20160918/compartments",
				"durationMs": float64(1500),
				"status":     "complete",
			},
			map[string]interface{}{
				"provider":   "oci",
				"region":     "ap-singapore-1",
				"service":    "iaas",
				"path":       "/20160918/instances",
				"durationMs": float64(2500),
				"status":     "complete",
			},
		},
	}

	summary, ok := cmdbSyncTaskSlowAPISummary(stats, cmdbSyncTaskSlowAPIThresholdFromStats(stats))
	if !ok {
		t.Fatal("expected slow api summary")
	}
	if cmdbSyncTaskMetricInt64(summary["thresholdMs"]) != 2000 {
		t.Fatalf("expected persisted threshold, got %#v", summary["thresholdMs"])
	}
	top := cmdbSyncTaskMetricAttrsList(summary["top"])
	if len(top) != 1 || cmdbSyncTaskMetricText(top[0]["path"]) != "/20160918/instances" {
		t.Fatalf("expected only endpoint above persisted threshold, got %#v", top)
	}
}

func TestCloudSyncPolicyNormalizeSlowAPIThreshold(t *testing.T) {
	params := cloudSyncPolicyNormalizeParams(models.ResAttrs{
		"slowApiThresholdMs":    float64(2500),
		"slowApiSilenceMinutes": float64(30),
		"autoRetryFailedScopes": true,
		"autoRetryMaxScopes":    float64(12),
	})
	if got := cmdbSyncTaskMetricInt64(params["slowApiThresholdMs"]); got != 2500 {
		t.Fatalf("expected persisted threshold 2500, got %#v", params["slowApiThresholdMs"])
	}
	if got := cmdbSyncTaskMetricInt64(params["slowApiSilenceMinutes"]); got != 30 {
		t.Fatalf("expected persisted silence window 30, got %#v", params["slowApiSilenceMinutes"])
	}
	if params["autoRetryFailedScopes"] != true {
		t.Fatalf("expected auto retry failed scopes enabled, got %#v", params["autoRetryFailedScopes"])
	}
	if got := cmdbSyncTaskMetricInt64(params["autoRetryMaxScopes"]); got != 12 {
		t.Fatalf("expected auto retry max scopes 12, got %#v", params["autoRetryMaxScopes"])
	}

	params = cloudSyncPolicyNormalizeParams(models.ResAttrs{
		"slowApiThresholdMs":    float64(900000),
		"slowApiSilenceMinutes": float64(20000),
		"autoRetryFailedScopes": "true",
		"autoRetryMaxScopes":    float64(200),
	})
	if got := cmdbSyncTaskMetricInt64(params["slowApiThresholdMs"]); got != cmdbSyncTaskSlowAPIThresholdMaxMs {
		t.Fatalf("expected threshold capped at %d, got %#v", cmdbSyncTaskSlowAPIThresholdMaxMs, params["slowApiThresholdMs"])
	}
	if got := cmdbSyncTaskMetricInt64(params["slowApiSilenceMinutes"]); got != cmdbSyncTaskSlowAPIEventSilenceMaxMinutes {
		t.Fatalf("expected silence window capped at %d, got %#v", cmdbSyncTaskSlowAPIEventSilenceMaxMinutes, params["slowApiSilenceMinutes"])
	}
	if got := cmdbSyncTaskMetricInt64(params["autoRetryMaxScopes"]); got != cloudSyncPolicyMaxAutoRetryScopes {
		t.Fatalf("expected auto retry max scopes capped at %d, got %#v", cloudSyncPolicyMaxAutoRetryScopes, params["autoRetryMaxScopes"])
	}

	params = cloudSyncPolicyNormalizeParams(models.ResAttrs{
		"slowApiThresholdMs":    0,
		"slowApiSilenceMinutes": 0,
		"autoRetryFailedScopes": false,
		"autoRetryMaxScopes":    10,
	})
	if _, ok := params["slowApiThresholdMs"]; ok {
		t.Fatalf("expected zero threshold to be removed, got %#v", params)
	}
	if _, ok := params["slowApiSilenceMinutes"]; ok {
		t.Fatalf("expected zero silence window to be removed, got %#v", params)
	}
	if _, ok := params["autoRetryFailedScopes"]; ok {
		t.Fatalf("expected disabled auto retry flag to be removed, got %#v", params)
	}
	if _, ok := params["autoRetryMaxScopes"]; ok {
		t.Fatalf("expected disabled auto retry max scopes to be removed, got %#v", params)
	}
}

func TestCloudSyncPolicyNormalizeNotificationRouting(t *testing.T) {
	params := cloudSyncPolicyNormalizeParams(models.ResAttrs{
		"notificationOwner":            " sre-oncall ",
		"notificationRoutes":           "sre, cloud-platform\ninfra",
		"notificationAssignees":        []interface{}{"alice", "alice", " team-a "},
		"notificationFailureRoutes":    "permission: iam, security\nrate-limit: cloud-platform",
		"notificationServiceRoutes":    "iaas: oci-team; ec2: compute-oncall",
		"notificationEscalationAt":     float64(200),
		"notificationEscalationRoutes": "sre-manager cloud-manager",
		"itsmAutoTicket":               "true",
		"itsmConnectorId":              "citc-1, citc-2",
		"itsmPriority":                 " high ",
	})
	if params["notificationOwner"] != "sre-oncall" {
		t.Fatalf("expected owner to be trimmed, got %#v", params["notificationOwner"])
	}
	routes := cloudSyncPolicyAttrStringSlice(params["notificationRoutes"])
	if len(routes) != 3 || routes[0] != "sre" || routes[1] != "cloud-platform" || routes[2] != "infra" {
		t.Fatalf("unexpected routes: %#v", routes)
	}
	assignees := cloudSyncPolicyAttrStringSlice(params["notificationAssignees"])
	if len(assignees) != 2 || assignees[0] != "alice" || assignees[1] != "team-a" {
		t.Fatalf("unexpected assignees: %#v", assignees)
	}
	failureRoutes := cloudSyncPolicyNotificationRouteMap(params["notificationFailureRoutes"], cloudSyncPolicyNormalizeFailureCategory)
	permissionRoutes := cloudSyncPolicyNormalizeRoutingValues(failureRoutes["permission"])
	if len(permissionRoutes) != 2 || permissionRoutes[0] != "iam" || permissionRoutes[1] != "security" {
		t.Fatalf("unexpected failure routes: %#v", failureRoutes)
	}
	serviceRoutes := cloudSyncPolicyNotificationRouteMap(params["notificationServiceRoutes"], cloudEventNotificationRouteKey)
	if got := cloudSyncPolicyNormalizeRoutingValues(serviceRoutes["iaas"]); len(got) != 1 || got[0] != "oci-team" {
		t.Fatalf("unexpected service routes: %#v", serviceRoutes)
	}
	if got := cloudSyncPolicyNotificationEscalationAt(params); got != cloudSyncPolicyMaxEscalationAt {
		t.Fatalf("expected escalation threshold capped at %d, got %#v", cloudSyncPolicyMaxEscalationAt, params["notificationEscalationAt"])
	}
	escalationRoutes := cloudSyncPolicyNormalizeRoutingValues(params["notificationEscalationRoutes"])
	if len(escalationRoutes) != 2 || escalationRoutes[0] != "sre-manager" || escalationRoutes[1] != "cloud-manager" {
		t.Fatalf("unexpected escalation routes: %#v", escalationRoutes)
	}
	if params["itsmAutoTicket"] != true || params["itsmPriority"] != "high" {
		t.Fatalf("unexpected ITSM attrs: %#v", params)
	}
	connectorIds := cloudSyncPolicyNormalizeRoutingValues(params["itsmConnectorIds"])
	if len(connectorIds) != 2 || connectorIds[0] != "citc-1" || connectorIds[1] != "citc-2" {
		t.Fatalf("unexpected ITSM connector ids: %#v", connectorIds)
	}

	params = cloudSyncPolicyNormalizeParams(models.ResAttrs{
		"notificationOwner":            " ",
		"notificationRoutes":           []string{},
		"notificationAssignees":        "",
		"notificationFailureRoutes":    "permission:",
		"notificationServiceRoutes":    map[string]interface{}{"": "x"},
		"notificationEscalationAt":     0,
		"notificationEscalationRoutes": "",
		"itsmAutoTicket":               false,
		"itsmConnectorIds":             "",
		"itsmPriority":                 "",
	})
	for _, key := range []string{"notificationOwner", "notificationRoutes", "notificationAssignees", "notificationFailureRoutes", "notificationServiceRoutes", "notificationEscalationAt", "notificationEscalationRoutes", "itsmAutoTicket", "itsmConnectorIds", "itsmPriority"} {
		if _, ok := params[key]; ok {
			t.Fatalf("expected empty %s to be removed, got %#v", key, params)
		}
	}
}

func TestCloudSyncPolicyNotificationRoutingAttrs(t *testing.T) {
	policy := &models.CloudSyncPolicy{
		Params: models.ResAttrs{
			"notificationOwner":            "sre-oncall",
			"notificationRoutes":           []string{"sre", "cloud-platform"},
			"notificationAssignees":        "alice team-a",
			"notificationFailureRoutes":    models.ResAttrs{"permission": []string{"iam"}},
			"notificationServiceRoutes":    "iaas: oci-team",
			"notificationEscalationAt":     2,
			"notificationEscalationRoutes": []string{"sre-manager"},
			"itsmAutoTicket":               true,
			"itsmConnectorIds":             []string{"citc-1"},
			"itsmPriority":                 "high",
		},
	}
	payload := cloudSyncPolicyApplyNotificationRoutingWithContext(policy, models.ResAttrs{
		"taskId":       "task-1",
		"failureCount": 2,
	}, "provider=oci region=ap-singapore-1 service=iaas status=403 code=NotAuthorized message=permission denied", nil, false)
	if payload["taskId"] != "task-1" || payload["notificationOwner"] != "sre-oncall" {
		t.Fatalf("unexpected routing payload: %#v", payload)
	}
	routes := cloudSyncPolicyAttrStringSlice(payload["notificationRoutes"])
	if len(routes) != 5 ||
		routes[0] != "sre" ||
		routes[1] != "cloud-platform" ||
		routes[2] != "iam" ||
		routes[3] != "oci-team" ||
		routes[4] != "sre-manager" {
		t.Fatalf("unexpected routing routes: %#v", routes)
	}
	assignees := cloudSyncPolicyAttrStringSlice(payload["notificationAssignees"])
	if len(assignees) != 2 || assignees[0] != "alice" || assignees[1] != "team-a" {
		t.Fatalf("unexpected routing assignees: %#v", assignees)
	}
	if payload["failureCategory"] != "permission" || payload["cloudService"] != "iaas" || payload["notificationEscalated"] != true {
		t.Fatalf("expected failure category, cloud service and escalation attrs, got %#v", payload)
	}
	if payload["notificationEscalationReason"] != "failure_count" {
		t.Fatalf("unexpected escalation reason: %#v", payload["notificationEscalationReason"])
	}
	if payload["itsmAutoTicket"] != true || payload["itsmPriority"] != "high" {
		t.Fatalf("expected ITSM attrs, got %#v", payload)
	}
	if connectorIds := cloudSyncPolicyAttrStringSlice(payload["itsmConnectorIds"]); len(connectorIds) != 1 || connectorIds[0] != "citc-1" {
		t.Fatalf("unexpected ITSM connector ids: %#v", payload["itsmConnectorIds"])
	}
}

func TestCloudEventNotificationCandidatesIncludesRouting(t *testing.T) {
	candidates := cloudEventNotificationCandidates(models.CloudEvent{
		Source:    models.CloudEventSourceSync,
		EventType: "cloud.sync.policy.failed",
		Payload: models.ResAttrs{
			"notificationOwner":            "SRE OnCall",
			"notificationRoutes":           []string{"Cloud Platform", "k8s/oncall"},
			"notificationAssignees":        "alice@example.com team-a",
			"failureCategory":              "permission",
			"cloudService":                 "OCI IaaS",
			"notificationEscalationReason": "failure_count",
		},
	})
	candidateSet := map[string]bool{}
	for _, candidate := range candidates {
		candidateSet[candidate] = true
	}
	for _, expected := range []string{
		"cloud.sync.policy.failed",
		"cloud.*",
		"sync.*",
		"cloud.route.cloud-platform",
		"cloud.route.k8s-oncall",
		"cloud.owner.sre-oncall",
		"cloud.assignee.alice-example.com",
		"cloud.assignee.team-a",
		"cloud.failure.permission",
		"sync.failure.permission",
		"cloud.service.oci-iaas",
		"sync.service.oci-iaas",
		"cloud.escalation.failure_count",
		"sync.escalation.failure_count",
	} {
		if !candidateSet[expected] {
			t.Fatalf("expected candidate %s in %#v", expected, candidates)
		}
	}
}

func TestCmdbSyncTaskFailedRetryScopesFromDetails(t *testing.T) {
	stats := models.ResAttrs{
		"failureDetails": []models.ResAttrs{
			{
				"message":   "rate limited",
				"retryable": true,
				"region":    "ap-singapore-1",
				"assetType": "instance",
			},
			{
				"message":   "permission denied",
				"retryable": false,
				"region":    "ap-singapore-1",
				"assetType": "bucket",
			},
		},
	}

	scopes, source := cmdbSyncTaskFailedRetryScopes(stats, []string{"ap-singapore-1", "us-ashburn-1"}, []string{"instance", "bucket"}, 10)
	if source != "failure_details" {
		t.Fatalf("expected failure details source, got %q", source)
	}
	if len(scopes) != 1 || scopes[0].Region != "ap-singapore-1" || scopes[0].AssetType != "instance" {
		t.Fatalf("unexpected retry scopes: %#v", scopes)
	}
	regions, assetTypes := cmdbSyncTaskScopeLists(scopes)
	if len(regions) != 1 || regions[0] != "ap-singapore-1" || len(assetTypes) != 1 || assetTypes[0] != "instance" {
		t.Fatalf("unexpected retry scope lists: regions=%#v assetTypes=%#v", regions, assetTypes)
	}
}

func TestCmdbSyncTaskFailedRetryScopesFallbackToMetrics(t *testing.T) {
	stats := models.ResAttrs{
		"scopeMetrics": []models.ResAttrs{
			{"region": "ap-singapore-1", "assetType": "instance", "status": "complete"},
			{"region": "us-ashburn-1", "assetType": "bucket", "status": "partial_failed"},
		},
	}

	scopes, source := cmdbSyncTaskFailedRetryScopes(stats, []string{"ap-singapore-1", "us-ashburn-1"}, []string{"instance", "bucket"}, 10)
	if source != "scope_metrics" {
		t.Fatalf("expected scope metrics source, got %q", source)
	}
	if len(scopes) != 1 || scopes[0].Region != "us-ashburn-1" || scopes[0].AssetType != "bucket" {
		t.Fatalf("unexpected metric retry scopes: %#v", scopes)
	}
}

func TestCmdbSyncTaskSlowAPISilenceFromStats(t *testing.T) {
	stats := models.ResAttrs{"slowApiSilenceMinutes": float64(45)}
	if got := cmdbSyncTaskSlowAPISilenceFromStats(stats); got != 45 {
		t.Fatalf("expected silence window 45, got %d", got)
	}
	stats["slowApiSilenceMinutes"] = float64(20000)
	if got := cmdbSyncTaskSlowAPISilenceFromStats(stats); got != cmdbSyncTaskSlowAPIEventSilenceMaxMinutes {
		t.Fatalf("expected capped silence window %d, got %d", cmdbSyncTaskSlowAPIEventSilenceMaxMinutes, got)
	}
	stats["slowApiSilenceMinutes"] = float64(0)
	if got := cmdbSyncTaskSlowAPISilenceFromStats(stats); got != 0 {
		t.Fatalf("expected disabled silence window, got %d", got)
	}
}

func TestCmdbSyncTaskSlowAPIEventFingerprint(t *testing.T) {
	got := cmdbSyncTaskSlowAPIEventFingerprint("oracle", "acct-1", models.Id("csp-1"), "region-core", "ap-singapore-1 iaas /20160918/instances")
	want := "oci|acct-1|csp-1|region-core|ap-singapore-1 iaas /20160918/instances"
	if got != want {
		t.Fatalf("unexpected slow api fingerprint: %q", got)
	}
}
