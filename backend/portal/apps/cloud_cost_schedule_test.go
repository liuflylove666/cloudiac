package apps

import (
	"testing"
	"time"

	"cloudiac/portal/models"
)

func TestCloudCostSyncScheduleNormalizeNotificationParams(t *testing.T) {
	params := cloudCostSyncScheduleNormalizeNotificationParams(models.ResAttrs{
		"notificationOwner":            " finops-oncall ",
		"notificationRoutes":           "finops, cloud-platform",
		"notificationAssignees":        []interface{}{"alice", "alice", " team-a "},
		"notificationSilenceMinutes":   float64(20000),
		"notificationWindows":          "09:00-18:00, invalid, 22:00-01:00",
		"notificationFailureRoutes":    "permission: iam, security\nrate_limit=cloud-platform",
		"notificationEscalationAt":     200,
		"notificationEscalationRoutes": "finops-lead sre-manager",
	})
	if params["notificationOwner"] != "finops-oncall" {
		t.Fatalf("expected owner trimmed, got %#v", params["notificationOwner"])
	}
	routes := cloudCostSyncScheduleNotificationValues(params, "notificationRoutes")
	if len(routes) != 2 || routes[0] != "finops" || routes[1] != "cloud-platform" {
		t.Fatalf("unexpected routes: %#v", routes)
	}
	assignees := cloudCostSyncScheduleNotificationValues(params, "notificationAssignees")
	if len(assignees) != 2 || assignees[0] != "alice" || assignees[1] != "team-a" {
		t.Fatalf("unexpected assignees: %#v", assignees)
	}
	if got := cloudCostSyncScheduleNotificationSilenceMinutes(params); got != cloudCostSyncScheduleMaxSilenceMins {
		t.Fatalf("expected silence capped at %d, got %d", cloudCostSyncScheduleMaxSilenceMins, got)
	}
	windows := cloudCostSyncScheduleNotificationWindows(params)
	if len(windows) != 2 || windows[0] != "09:00-18:00" || windows[1] != "22:00-01:00" {
		t.Fatalf("unexpected windows: %#v", windows)
	}
	failureRoutes := cloudCostSyncScheduleNotificationFailureRoutes(params)
	permissionRoutes := cloudSyncPolicyNormalizeRoutingValues(failureRoutes["permission"])
	if len(permissionRoutes) != 2 || permissionRoutes[0] != "iam" || permissionRoutes[1] != "security" {
		t.Fatalf("unexpected permission routes: %#v", failureRoutes)
	}
	if got := cloudCostSyncScheduleNotificationEscalationAt(params); got != cloudCostSyncScheduleMaxEscalateAt {
		t.Fatalf("expected escalation threshold capped at %d, got %d", cloudCostSyncScheduleMaxEscalateAt, got)
	}
	escalationRoutes := cloudCostSyncScheduleNotificationValues(params, "notificationEscalationRoutes")
	if len(escalationRoutes) != 2 || escalationRoutes[0] != "finops-lead" || escalationRoutes[1] != "sre-manager" {
		t.Fatalf("unexpected escalation routes: %#v", escalationRoutes)
	}

	params = cloudCostSyncScheduleNormalizeNotificationParams(models.ResAttrs{
		"notificationOwner":            "",
		"notificationRoutes":           []string{},
		"notificationAssignees":        "",
		"notificationSilenceMinutes":   0,
		"notificationWindows":          "bad",
		"notificationFailureRoutes":    "bad-line",
		"notificationEscalationAt":     0,
		"notificationEscalationRoutes": "",
	})
	for _, key := range []string{"notificationOwner", "notificationRoutes", "notificationAssignees", "notificationSilenceMinutes", "notificationWindows", "notificationFailureRoutes", "notificationEscalationAt", "notificationEscalationRoutes"} {
		if _, ok := params[key]; ok {
			t.Fatalf("expected empty %s to be removed, got %#v", key, params)
		}
	}
}

func TestCloudCostSyncScheduleNotificationRoutingPayload(t *testing.T) {
	schedule := &models.CloudCostSyncSchedule{}
	schedule.SetId("ccp-1")
	params := cloudCostSyncScheduleNormalizeNotificationParams(models.ResAttrs{
		"notificationOwner":          "finops-oncall",
		"notificationRoutes":         []string{"finops", "cloud-platform"},
		"notificationAssignees":      "alice team-a",
		"notificationSilenceMinutes": 30,
		"notificationWindows":        "09:00-18:00",
		"notificationFailureRoutes": models.ResAttrs{
			"permission": []string{"iam", "security"},
			"network":    "sre",
		},
		"notificationEscalationAt":     2,
		"notificationEscalationRoutes": []string{"finops-lead"},
	})
	payload := cloudCostSyncScheduleApplyNotificationRouting(schedule, params, models.ResAttrs{
		"scheduleId":   "ccp-1",
		"failureCount": 2,
	}, "AccessDenied: missing iam permission", false)
	if payload["notificationOwner"] != "finops-oncall" || payload["notificationSource"] != "cloud_cost_sync_schedule" {
		t.Fatalf("unexpected routing payload: %#v", payload)
	}
	routes := cloudCostSyncScheduleNotificationValues(payload, "notificationRoutes")
	if len(routes) != 5 || routes[0] != "finops" || routes[1] != "cloud-platform" || routes[2] != "iam" || routes[3] != "security" || routes[4] != "finops-lead" {
		t.Fatalf("unexpected payload routes: %#v", routes)
	}
	if payload["failureCategory"] != "permission" || payload["notificationEscalated"] != true || payload["notificationEscalationReason"] != "failure_count" {
		t.Fatalf("unexpected failure routing metadata: %#v", payload)
	}
	if cloudCostSyncScheduleNotificationSilenceMinutes(payload) != 30 {
		t.Fatalf("expected payload silence 30, got %#v", payload["notificationSilenceMinutes"])
	}
}

func TestCloudCostSyncScheduleFailureCategory(t *testing.T) {
	cases := map[string]string{
		"HTTP 429 Too Many Requests":              "rate_limit",
		"ExpiredToken: token expired":             "credential",
		"AccessDenied: not authorized":            "permission",
		"404 object not found":                    "not_found",
		"dial tcp timeout waiting for connection": "network",
		"unsupported file format":                 "config",
		"something odd happened":                  "unknown",
	}
	for reason, expected := range cases {
		if got := cloudCostSyncScheduleFailureCategory(reason); got != expected {
			t.Fatalf("expected %q to be %s, got %s", reason, expected, got)
		}
	}
}

func TestCloudCostSyncScheduleNotificationWindowActive(t *testing.T) {
	windows := []string{"09:00-18:00", "22:00-01:00"}
	if !cloudCostSyncScheduleNotificationWindowActive(windows, time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)) {
		t.Fatal("expected business-hour window to be active")
	}
	if !cloudCostSyncScheduleNotificationWindowActive(windows, time.Date(2026, 6, 22, 23, 0, 0, 0, time.UTC)) {
		t.Fatal("expected overnight window to be active")
	}
	if cloudCostSyncScheduleNotificationWindowActive(windows, time.Date(2026, 6, 22, 3, 0, 0, 0, time.UTC)) {
		t.Fatal("expected notification window to be inactive")
	}
}

func TestCloudEventNotificationCandidatesIncludesCostRouting(t *testing.T) {
	candidates := cloudEventNotificationCandidates(models.CloudEvent{
		Source:    models.CloudEventSourceCost,
		EventType: "cost.sync.schedule.failed",
		Payload: models.ResAttrs{
			"notificationOwner":            "FinOps OnCall",
			"notificationRoutes":           []string{"FinOps", "cloud-platform"},
			"notificationAssignees":        "alice@example.com team-a",
			"failureCategory":              "permission",
			"notificationEscalationReason": "failure_count",
		},
	})
	candidateSet := map[string]bool{}
	for _, candidate := range candidates {
		candidateSet[candidate] = true
	}
	for _, expected := range []string{
		"cost.sync.schedule.failed",
		"cost.*",
		"cloud.route.finops",
		"cost.route.finops",
		"cost.route.cloud-platform",
		"cost.owner.finops-oncall",
		"cost.assignee.alice-example.com",
		"cost.assignee.team-a",
		"cloud.failure.permission",
		"cost.failure.permission",
		"cloud.escalation.failure_count",
		"cost.escalation.failure_count",
	} {
		if !candidateSet[expected] {
			t.Fatalf("expected candidate %s in %#v", expected, candidates)
		}
	}
}
