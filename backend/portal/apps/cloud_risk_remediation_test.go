package apps

import (
	"strings"
	"testing"
	"time"

	"cloudiac/portal/consts"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
)

func TestCloudRiskRemediationParams(t *testing.T) {
	finding := models.CloudRiskFinding{
		ProjectId:      models.Id("p-1"),
		EnvId:          models.Id("env-1"),
		AssetId:        models.Id("ci-1"),
		CloudAccountId: models.Id("ca-1"),
		Provider:       "aws",
		AccountId:      "123456789012",
		Region:         "us-east-1",
		ResourceType:   "compute_instance",
		ResourceId:     "i-123",
		ResourceName:   "web-1",
		Source:         models.CloudRiskSourceCloudConfig,
		RuleKey:        "public_ingress_security_rule",
		RuleName:       "公网入方向规则",
		RiskLevel:      models.CloudOperationRiskCritical,
		Status:         models.CloudRiskStatusOpen,
		Evidence: models.ResAttrs{
			"cidr": "0.0.0.0/0",
			"port": "22",
		},
		Recommendation: "收敛公网规则范围",
	}
	finding.Id = models.Id("crf-1")

	params := cloudRiskRemediationParams(finding, models.ResAttrs{"custom": "keep"})
	if params["riskRemediation"] != true || params["custom"] != "keep" {
		t.Fatalf("expected remediation params with custom value, got %#v", params)
	}
	if params["riskId"] != "crf-1" || params["riskLevel"] != models.CloudOperationRiskCritical {
		t.Fatalf("unexpected risk identity params: %#v", params)
	}
	evidence := modelResAttrs(params["evidence"])
	if evidence["cidr"] != "0.0.0.0/0" || evidence["port"] != "22" {
		t.Fatalf("expected evidence in remediation params, got %#v", evidence)
	}
}

func TestCloudRiskRemediationParamsForDrift(t *testing.T) {
	finding := models.CloudRiskFinding{
		Source:    models.CloudRiskSourceDrift,
		RuleKey:   "terraform_drift_detected",
		RiskLevel: models.CloudOperationRiskHigh,
		Status:    models.CloudRiskStatusOpen,
	}
	finding.Id = models.Id("crf-drift")

	params := cloudRiskRemediationParams(finding, nil)
	if params["driftRemediation"] != true || params["targetState"] != "iac_desired_state" {
		t.Fatalf("expected drift remediation target state, got %#v", params)
	}
}

func TestCloudRiskRemediationTitleAndDescription(t *testing.T) {
	finding := models.CloudRiskFinding{
		Provider:       "oci",
		Region:         "ap-singapore-1",
		ResourceName:   "prod-subnet",
		RuleName:       "IaC 漂移",
		RiskLevel:      models.CloudOperationRiskHigh,
		Source:         models.CloudRiskSourceDrift,
		Recommendation: "重新执行 plan/apply",
	}
	title := cloudRiskRemediationTitle(finding)
	if !strings.Contains(title, "IaC 漂移") || !strings.Contains(title, "prod-subnet") {
		t.Fatalf("unexpected title: %s", title)
	}
	description := cloudRiskRemediationDescription(finding)
	for _, want := range []string{"风险等级：high", "风险来源：drift", "oci / ap-singapore-1", "重新执行 plan/apply"} {
		if !strings.Contains(description, want) {
			t.Fatalf("expected description to contain %q, got %s", want, description)
		}
	}
}

func TestNormalizeCloudItsmSelfServiceRiskRemediationParams(t *testing.T) {
	normalized, err := normalizeCloudItsmSelfServiceParams(models.CloudOperationActionRiskRemediation, models.ResAttrs{
		"riskId": "crf-1",
	})
	if err != nil {
		t.Fatalf("unexpected normalize error: %v", err)
	}
	if normalized["riskRemediationRequired"] != true {
		t.Fatalf("expected risk remediation flag, got %#v", normalized)
	}
	if normalized["targetState"] != "risk_remediated_state" {
		t.Fatalf("expected default remediation target state, got %#v", normalized["targetState"])
	}
}

func TestCloudItsmRiskStatusForTicketStatus(t *testing.T) {
	cases := []struct {
		status string
		want   string
		ok     bool
	}{
		{status: models.CloudItsmTicketStatusSubmitted, want: models.CloudRiskStatusInProgress, ok: true},
		{status: models.CloudItsmTicketStatusInProgress, want: models.CloudRiskStatusInProgress, ok: true},
		{status: models.CloudItsmTicketStatusResolved, want: models.CloudRiskStatusResolved, ok: true},
		{status: models.CloudItsmTicketStatusClosed, want: models.CloudRiskStatusResolved, ok: true},
		{status: models.CloudItsmTicketStatusFailed, want: models.CloudRiskStatusOpen, ok: true},
		{status: models.CloudItsmTicketStatusCanceled, want: models.CloudRiskStatusOpen, ok: true},
		{status: models.CloudItsmTicketStatusPending, ok: false},
	}
	for _, tc := range cases {
		got, ok := cloudItsmRiskStatusForTicketStatus(tc.status)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("status %s expected (%s,%v), got (%s,%v)", tc.status, tc.want, tc.ok, got, ok)
		}
	}
}

func TestCloudItsmOperationRiskRemediationDetection(t *testing.T) {
	operation := models.CloudOperation{
		OperationType: models.CloudOperationTypeSelfService,
		Action:        models.CloudOperationActionRiskRemediation,
		Params: models.ResAttrs{
			"riskId": "crf-1",
		},
	}
	if !cloudItsmOperationIsRiskRemediation(operation, modelResAttrs(operation.Params)) {
		t.Fatalf("expected risk remediation operation")
	}
	if got := cloudItsmOperationRiskId(modelResAttrs(operation.Params)); got != "crf-1" {
		t.Fatalf("expected risk id crf-1, got %s", got)
	}

	operation.Action = models.CloudOperationActionPermissionRequest
	operation.Params = models.ResAttrs{
		"requestType":     models.CloudOperationActionRiskRemediation,
		"riskFindingId":   "crf-2",
		"riskRemediation": true,
	}
	if !cloudItsmOperationIsRiskRemediation(operation, modelResAttrs(operation.Params)) {
		t.Fatalf("expected risk remediation by params")
	}
	if got := cloudItsmOperationRiskId(modelResAttrs(operation.Params)); got != "crf-2" {
		t.Fatalf("expected risk id crf-2, got %s", got)
	}
}

func TestCloudItsmRiskRemediationEvidence(t *testing.T) {
	ticket := &models.CloudItsmTicket{
		ExternalId:  "ext-1",
		ExternalKey: "ITSM-1",
		ExternalUrl: "https://itsm.example.com/tickets/ITSM-1",
		Status:      models.CloudItsmTicketStatusResolved,
	}
	ticket.Id = models.Id("cit-1")
	ticket.ConnectorId = models.Id("citc-1")
	operation := &models.CloudOperation{
		Name: "风险整改申请",
	}
	operation.Id = models.Id("cop-1")
	form := &forms.UpdateCloudItsmTicketStatusForm{
		Comment: "外部 ITSM 已解决",
		Payload: models.ResAttrs{"resolution": "fixed"},
	}
	syncedAt := models.Time(time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC))

	evidence := cloudItsmRiskRemediationEvidence(ticket, operation, form, models.CloudRiskStatusResolved, syncedAt)
	if evidence["lastItsmTicketId"] != "cit-1" || evidence["lastItsmOperationId"] != "cop-1" {
		t.Fatalf("unexpected ticket evidence: %#v", evidence)
	}
	if evidence["lastItsmRiskStatus"] != models.CloudRiskStatusResolved || evidence["lastItsmStatusComment"] != "外部 ITSM 已解决" {
		t.Fatalf("unexpected status evidence: %#v", evidence)
	}
	payload := modelResAttrs(evidence["lastItsmStatusPayload"])
	if payload["resolution"] != "fixed" {
		t.Fatalf("expected status payload, got %#v", payload)
	}
}

func TestCloudItsmRiskFindingIsDriftRemediation(t *testing.T) {
	if !cloudItsmRiskFindingIsDriftRemediation(models.CloudRiskFinding{Source: models.CloudRiskSourceDrift}, nil) {
		t.Fatalf("expected drift source to be treated as drift remediation")
	}
	if !cloudItsmRiskFindingIsDriftRemediation(models.CloudRiskFinding{RuleKey: "terraform_drift_detected"}, nil) {
		t.Fatalf("expected terraform drift rule to be treated as drift remediation")
	}
	if !cloudItsmRiskFindingIsDriftRemediation(models.CloudRiskFinding{}, models.ResAttrs{"driftRemediation": true}) {
		t.Fatalf("expected driftRemediation flag to be treated as drift remediation")
	}
	if !cloudItsmRiskFindingIsDriftRemediation(models.CloudRiskFinding{}, models.ResAttrs{"requestType": models.CloudOperationActionDriftRemediation}) {
		t.Fatalf("expected drift remediation request type to be treated as drift remediation")
	}
	if !cloudItsmRiskFindingIsDriftRemediation(models.CloudRiskFinding{}, models.ResAttrs{"targetState": "iac_desired_state"}) {
		t.Fatalf("expected iac_desired_state target to be treated as drift remediation")
	}
	if cloudItsmRiskFindingIsDriftRemediation(models.CloudRiskFinding{Source: models.CloudRiskSourceCloudConfig}, models.ResAttrs{"targetState": "risk_remediated_state"}) {
		t.Fatalf("did not expect cloud config remediation to be treated as drift remediation")
	}
}

func TestCloudItsmDriftAutoRepairEnvId(t *testing.T) {
	params := models.ResAttrs{"envId": "env-param"}
	operation := models.CloudOperation{EnvId: models.Id("env-operation")}
	finding := models.CloudRiskFinding{EnvId: models.Id("env-finding")}
	if got := cloudItsmDriftAutoRepairEnvId(finding, operation, params); got != "env-finding" {
		t.Fatalf("expected finding env priority, got %s", got)
	}
	finding.EnvId = ""
	if got := cloudItsmDriftAutoRepairEnvId(finding, operation, params); got != "env-operation" {
		t.Fatalf("expected operation env fallback, got %s", got)
	}
	operation.EnvId = ""
	if got := cloudItsmDriftAutoRepairEnvId(finding, operation, params); got != "env-param" {
		t.Fatalf("expected params env fallback, got %s", got)
	}
}

func TestCloudItsmDriftAutoRepairEvidenceAndTriggerFlag(t *testing.T) {
	checkedAt := models.Time(time.Date(2026, 6, 22, 11, 0, 0, 0, time.UTC))
	evidence := cloudItsmDriftAutoRepairEvidence(checkedAt, models.Id("env-1"), false, "cron_drift_disabled")
	if evidence["driftAutoRepairTriggered"] != false || evidence["driftAutoRepairSkippedReason"] != "cron_drift_disabled" {
		t.Fatalf("unexpected skipped evidence: %#v", evidence)
	}
	if evidence["driftAutoRepairEnvId"] != "env-1" || evidence["driftAutoRepairSource"] != "itsm_risk_remediation_resolved" {
		t.Fatalf("unexpected evidence identity: %#v", evidence)
	}
	if cloudItsmDriftAutoRepairTriggered(evidence) {
		t.Fatalf("did not expect skipped evidence to be treated as triggered")
	}
	evidence["driftAutoRepairTriggered"] = "true"
	if !cloudItsmDriftAutoRepairTriggered(evidence) {
		t.Fatalf("expected string true evidence to be treated as triggered")
	}
}

func TestCloudItsmDriftAutoRepairApprovalPolicyDefaults(t *testing.T) {
	env := models.Env{AutoApproval: true}
	repairPolicy := cloudItsmDriftAutoRepairApprovalPolicyForEnv(env, cloudItsmDriftAutoRepairApprovalContextRepair)
	if !repairPolicy.AutoApprove || repairPolicy.RequiresApproval || repairPolicy.Mode != cloudItsmDriftAutoRepairApprovalModeInheritEnv {
		t.Fatalf("expected initial auto repair to inherit env auto approval, got %#v", repairPolicy)
	}

	retryPolicy := cloudItsmDriftAutoRepairApprovalPolicyForEnv(env, cloudItsmDriftAutoRepairApprovalContextRetry)
	if retryPolicy.AutoApprove || !retryPolicy.RequiresApproval || retryPolicy.Mode != cloudItsmDriftAutoRepairApprovalModeRetryRequireOnly {
		t.Fatalf("expected retry to require approval by default, got %#v", retryPolicy)
	}
}

func TestCloudItsmDriftAutoRepairApprovalPolicyFromEnvExtraData(t *testing.T) {
	env := models.Env{
		AutoApproval: true,
		ExtraData: models.JSON(`{
			"driftAutoRepairApprovalPolicy": {
				"mode": "require_approval",
				"approverRoles": ["ops", "sre"]
			}
		}`),
	}
	policy := cloudItsmDriftAutoRepairApprovalPolicyForEnv(env, cloudItsmDriftAutoRepairApprovalContextRepair)
	if policy.AutoApprove || !policy.RequiresApproval || policy.Source != "env_extra_data" {
		t.Fatalf("expected env extraData policy to require approval, got %#v", policy)
	}
	if len(policy.ApproverRoles) != 2 || policy.ApproverRoles[0] != "ops" || policy.ApproverRoles[1] != "sre" {
		t.Fatalf("unexpected approver roles: %#v", policy.ApproverRoles)
	}
	evidence := cloudItsmDriftAutoRepairApprovalPolicyEvidence(policy)
	if evidence["driftAutoRepairApprovalPolicyMode"] != cloudItsmDriftAutoRepairApprovalModeRequireApproval || evidence["driftAutoRepairApprovalAutoApprove"] != false {
		t.Fatalf("unexpected approval policy evidence: %#v", evidence)
	}
}

func TestCloudItsmDriftAutoRepairApprovalPolicyRetryOverride(t *testing.T) {
	env := models.Env{
		ExtraData: models.JSON(`{
			"driftAutoRepairApprovalPolicy": {
				"mode": "retry_require_approval",
				"retryAutoApprove": true
			}
		}`),
	}
	policy := cloudItsmDriftAutoRepairApprovalPolicyForEnv(env, cloudItsmDriftAutoRepairApprovalContextRetry)
	if !policy.AutoApprove || policy.RequiresApproval || policy.Reason != "policy_retry_auto_approve_override" {
		t.Fatalf("expected retry auto approve override, got %#v", policy)
	}
}

func TestCloudItsmDriftAutoRepairApprovalPolicyInvalidModeFailsClosed(t *testing.T) {
	env := models.Env{
		AutoApproval: true,
		ExtraData: models.JSON(`{
			"driftAutoRepairApprovalPolicy": {
				"mode": "unknown"
			}
		}`),
	}
	policy := cloudItsmDriftAutoRepairApprovalPolicyForEnv(env, cloudItsmDriftAutoRepairApprovalContextRepair)
	if policy.AutoApprove || !policy.RequiresApproval || policy.Reason != "invalid_policy_mode_requires_approval" {
		t.Fatalf("expected invalid policy mode to require approval, got %#v", policy)
	}
}

func TestCloudItsmDriftAutoRepairTimelineForCompletedTask(t *testing.T) {
	evidence := models.ResAttrs{
		"driftAutoRepairCheckedAt":             "2026-06-22T10:00:00Z",
		"driftAutoRepairTriggered":             true,
		"driftAutoRepairEnvId":                 "env-1",
		"driftAutoRepairSourceTaskId":          "run-source",
		"driftAutoRepairApprovalPolicyMode":    cloudItsmDriftAutoRepairApprovalModeInheritEnv,
		"driftAutoRepairApprovalPolicySource":  "env_auto_approval",
		"driftAutoRepairApprovalPolicyContext": cloudItsmDriftAutoRepairApprovalContextRepair,
		"driftAutoRepairApprovalAutoApprove":   true,
		"driftAutoRepairApprovalReason":        "env_auto_approval_enabled",
		"driftAutoRepairTaskId":                "run-repair",
		"driftAutoRepairTaskType":              models.TaskTypeApply,
		"driftAutoRepairTaskStatus":            models.TaskComplete,
		"driftAutoRepairTaskAutoApprove":       true,
		"driftAutoRepairTaskStartedAt":         "2026-06-22T10:01:00Z",
		"driftAutoRepairTaskEndedAt":           "2026-06-22T10:03:00Z",
		"driftAutoRepairTaskLastSyncedAt":      "2026-06-22T10:04:00Z",
		"driftAutoRepairTaskRiskStatus":        models.CloudRiskStatusResolved,
		"driftAutoRepairRollbackEvaluatedAt":   "2026-06-22T10:04:00Z",
		"driftAutoRepairRollbackRequired":      false,
		"driftAutoRepairRollbackStrategy":      "no_platform_rollback_required",
		"driftAutoRepairRollbackSafetyLevel":   "low",
		"driftAutoRepairRollbackTaskId":        "run-repair",
	}

	timeline := cloudItsmDriftAutoRepairTimeline(evidence)
	for _, stage := range []string{"trigger_check", "source_task", "approval_policy", "repair_task_created", "task_started", "task_finished", "risk_result_synced", "rollback_evaluated"} {
		if cloudItsmTestTimelineStage(t, timeline, stage) == nil {
			t.Fatalf("expected stage %s in timeline %#v", stage, timeline)
		}
	}
	if got := cloudItsmTestTimelineStage(t, timeline, "approval_policy")["status"]; got != "auto_approved" {
		t.Fatalf("expected auto approved policy stage, got %#v", got)
	}
	if got := cloudItsmTestTimelineStage(t, timeline, "rollback_evaluated")["status"]; got != "no_rollback_required" {
		t.Fatalf("expected no rollback required stage, got %#v", got)
	}
}

func TestCloudItsmDriftAutoRepairTimelineForRetryAndRollbackReview(t *testing.T) {
	evidence := models.ResAttrs{
		"driftAutoRepairCheckedAt":             "2026-06-22T11:00:00Z",
		"driftAutoRepairTriggered":             true,
		"driftAutoRepairEnvId":                 "env-1",
		"driftAutoRepairSourceTaskId":          "run-source",
		"driftAutoRepairTaskId":                "run-retry",
		"driftAutoRepairTaskStatus":            models.TaskApproving,
		"driftAutoRepairTaskMessage":           "自动修复失败，已创建待审批重试任务",
		"driftAutoRepairTaskEndedAt":           "2026-06-22T11:03:00Z",
		"driftAutoRepairTaskLastSyncedAt":      "2026-06-22T11:04:00Z",
		"driftAutoRepairTaskRiskStatus":        models.CloudRiskStatusOpen,
		"driftAutoRepairLastFailedTaskId":      "run-failed",
		"driftAutoRepairLastFailedTaskStatus":  models.TaskFailed,
		"driftAutoRepairRetryCheckedAt":        "2026-06-22T11:04:00Z",
		"driftAutoRepairRetryTaskId":           "run-retry",
		"driftAutoRepairRetryApprovalStatus":   "pending",
		"driftAutoRepairRetryAttempt":          1,
		"driftAutoRepairRetryMaxAttempts":      2,
		"driftAutoRepairApprovalPolicyMode":    cloudItsmDriftAutoRepairApprovalModeRetryRequireOnly,
		"driftAutoRepairApprovalPolicySource":  "default_retry_guardrail",
		"driftAutoRepairApprovalPolicyContext": cloudItsmDriftAutoRepairApprovalContextRetry,
		"driftAutoRepairApprovalAutoApprove":   false,
		"driftAutoRepairApprovalReason":        "retry_after_failed_auto_repair_requires_approval",
		"driftAutoRepairRollbackEvaluatedAt":   "2026-06-22T11:04:00Z",
		"driftAutoRepairRollbackRequired":      true,
		"driftAutoRepairRollbackStrategy":      "gitops_iac_change_or_manual_review_required",
		"driftAutoRepairRollbackSafetyLevel":   "high",
		"driftAutoRepairRollbackTaskId":        "run-failed",
	}

	timeline := cloudItsmDriftAutoRepairTimeline(evidence)
	if got := cloudItsmTestTimelineStage(t, timeline, "last_failed_task_finished")["taskId"]; got != "run-failed" {
		t.Fatalf("expected failed task id in timeline, got %#v", got)
	}
	retryStage := cloudItsmTestTimelineStage(t, timeline, "retry")
	if retryStage["taskId"] != "run-retry" || retryStage["status"] != "pending" {
		t.Fatalf("unexpected retry timeline stage: %#v", retryStage)
	}
	if got := cloudItsmTestTimelineStage(t, timeline, "rollback_evaluated")["status"]; got != "rollback_review_required" {
		t.Fatalf("expected rollback review required stage, got %#v", got)
	}
}

func TestCloudItsmDriftAutoRepairRecommendationsForSkippedConfig(t *testing.T) {
	evidence := models.ResAttrs{
		"driftAutoRepairCheckedAt":        "2026-06-22T12:00:00Z",
		"driftAutoRepairTriggered":        false,
		"driftAutoRepairEnvId":            "env-1",
		"driftAutoRepairSkippedReason":    "auto_repair_disabled",
		"driftAutoRepairOpenCronDrift":    true,
		"driftAutoRepairEnabled":          false,
		"driftAutoRepairApprovalRequired": true,
	}

	recommendations := cloudItsmDriftAutoRepairRecommendations(evidence)
	if cloudItsmTestRecommendationAction(t, recommendations, "enable_auto_repair") == nil {
		t.Fatalf("expected enable_auto_repair recommendation, got %#v", recommendations)
	}
	if got := cloudItsmTestRecommendationAction(t, recommendations, "open_env")["targetId"]; got != "env-1" {
		t.Fatalf("expected env target recommendation, got %#v", got)
	}
}

func TestCloudItsmDriftAutoRepairRecommendationsForRetryRollback(t *testing.T) {
	evidence := models.ResAttrs{
		"driftAutoRepairTriggered":             true,
		"driftAutoRepairEnvId":                 "env-1",
		"driftAutoRepairTaskId":                "run-retry",
		"driftAutoRepairRetryTaskId":           "run-retry",
		"driftAutoRepairLastFailedTaskId":      "run-failed",
		"driftAutoRepairTaskFailed":            false,
		"driftAutoRepairRetryApprovalRequired": true,
		"driftAutoRepairRollbackRequired":      true,
		"driftAutoRepairRollbackTaskId":        "run-failed",
	}

	recommendations := cloudItsmDriftAutoRepairRecommendations(evidence)
	if got := cloudItsmTestRecommendationAction(t, recommendations, "approve_retry_task")["targetId"]; got != "run-retry" {
		t.Fatalf("expected retry approval target, got %#v", got)
	}
	if got := cloudItsmTestRecommendationAction(t, recommendations, "review_rollback")["priority"]; got != "critical" {
		t.Fatalf("expected critical rollback recommendation, got %#v", got)
	}
	if cloudItsmTestRecommendationAction(t, recommendations, "create_gitops_pr") == nil {
		t.Fatalf("expected GitOps PR recommendation, got %#v", recommendations)
	}
}

func TestCloudRiskRecommendationAdoptionEvidence(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-06-22T12:00:00Z")
	base := models.ResAttrs{
		"driftAutoRepairRecommendations": []models.ResAttrs{
			{
				"action":     "approve_retry_task",
				"title":      "审批失败后的漂移修复重试任务",
				"priority":   "high",
				"targetType": "task",
				"targetId":   "run-retry",
				"reason":     "retry_approval_required",
			},
			{
				"action":     "create_gitops_pr",
				"title":      "通过 GitOps/IaC PR 提交修复或回滚变更",
				"priority":   "high",
				"targetType": "env",
				"targetId":   "env-1",
			},
		},
	}
	evidence, adoption, err := cloudRiskRecommendationAdoptionEvidence(base, forms.AdoptCloudRiskRecommendationForm{
		Action:     "approve_retry_task",
		TargetType: "task",
		TargetId:   "run-retry",
		Comment:    "已提交审批",
	}, models.Id("u-1"), now)
	if err != nil {
		t.Fatalf("unexpected adoption error: %v", err)
	}
	if adoption["adoptedBy"] != "u-1" || adoption["adoptedAt"] != "2026-06-22T12:00:00Z" {
		t.Fatalf("unexpected adoption audit: %#v", adoption)
	}
	if evidence["driftAutoRepairRecommendationAdoptedTotal"] != 1 {
		t.Fatalf("expected one adopted recommendation, got %#v", evidence["driftAutoRepairRecommendationAdoptedTotal"])
	}
	if got := evidence["driftAutoRepairRecommendationAdoptionRate"]; got != 50.0 {
		t.Fatalf("expected 50 percent adoption rate, got %#v", got)
	}
	recommendations := cloudRiskRecommendationAttrs(evidence["driftAutoRepairRecommendations"])
	approved := cloudItsmTestRecommendationAction(t, recommendations, "approve_retry_task")
	if approved["adopted"] != true || approved["adoptionComment"] != "已提交审批" {
		t.Fatalf("expected adopted recommendation marker, got %#v", approved)
	}
	if cloudItsmTestRecommendationAction(t, recommendations, "create_gitops_pr")["adopted"] == true {
		t.Fatalf("did not expect unrelated recommendation to be adopted: %#v", recommendations)
	}
}

func TestCloudRiskRecommendationAdoptionEvidenceRejectsUnknownAction(t *testing.T) {
	_, _, err := cloudRiskRecommendationAdoptionEvidence(models.ResAttrs{
		"driftAutoRepairRecommendations": []models.ResAttrs{
			{"action": "open_env", "targetType": "env", "targetId": "env-1"},
		},
	}, forms.AdoptCloudRiskRecommendationForm{Action: "missing"}, models.Id("u-1"), time.Now())
	if err == nil {
		t.Fatalf("expected unknown recommendation action error")
	}
}

func TestCloudItsmDriftAutoRepairSlaPolicyFromEnvExtraData(t *testing.T) {
	env := models.Env{
		ExtraData: models.JSON(`{
			"driftAutoRepairSlaPolicy": {
				"approvalDueMinutes": 45,
				"rollbackReviewDueMinutes": 120,
				"dueSoonMinutes": 10,
				"notificationRoutes": ["iac-approval", "sre"],
				"notificationAssignees": ["user-1"],
				"ownerRoles": ["ops"],
				"autoTicket": true
			}
		}`),
	}

	policy := cloudItsmDriftAutoRepairSlaPolicyForEnv(env)
	if policy.Source != "env_extra_data" || policy.ApprovalDueMinutes != 45 || policy.RollbackReviewDueMinutes != 120 || policy.DueSoonMinutes != 10 {
		t.Fatalf("unexpected SLA policy: %#v", policy)
	}
	if !policy.AutoTicket || len(policy.NotificationRoutes) != 2 || policy.NotificationRoutes[0] != "iac-approval" {
		t.Fatalf("unexpected SLA notification policy: %#v", policy)
	}
	evidence := cloudItsmDriftAutoRepairSlaPolicyEvidence(policy)
	if evidence["driftAutoRepairSlaApprovalDueMinutes"] != 45 || evidence["driftAutoRepairSlaAutoTicketOnEscalation"] != true {
		t.Fatalf("unexpected SLA policy evidence: %#v", evidence)
	}
}

func TestCloudItsmDriftAutoRepairSlaForApprovalDueSoon(t *testing.T) {
	evidence := models.ResAttrs{
		"driftAutoRepairCheckedAt":                 "2026-06-22T10:00:00Z",
		"driftAutoRepairTaskId":                    "run-approval",
		"driftAutoRepairApprovalRequired":          true,
		"driftAutoRepairApprovalAutoApprove":       false,
		"driftAutoRepairSlaApprovalDueMinutes":     60,
		"driftAutoRepairSlaDueSoonMinutes":         15,
		"driftAutoRepairSlaNotificationRoutes":     []string{"iac-approval"},
		"driftAutoRepairSlaNotificationOwnerRoles": []string{"sre"},
	}

	now := time.Date(2026, 6, 22, 10, 50, 0, 0, time.UTC)
	cloudItsmDriftAutoRepairAppendSlaEvidence(evidence, now)
	if evidence["driftAutoRepairSlaAction"] != "approve_repair" || evidence["driftAutoRepairSlaStatus"] != "due_soon" {
		t.Fatalf("unexpected approval SLA evidence: %#v", evidence)
	}
	if evidence["driftAutoRepairSlaTargetId"] != "run-approval" || evidence["driftAutoRepairSlaMinutesRemaining"] != 10 {
		t.Fatalf("unexpected approval SLA target/due: %#v", evidence)
	}
	payload := cloudItsmDriftAutoRepairSlaEventPayload(evidence)
	if !cloudItsmTestStringSliceContains(cloudSyncPolicyAttrStringSlice(payload["notificationRoutes"]), "drift_auto_repair_approve_repair") {
		t.Fatalf("expected action notification route, got %#v", payload)
	}
}

func TestCloudItsmDriftAutoRepairSlaEscalationPayload(t *testing.T) {
	evidence := models.ResAttrs{
		"driftAutoRepairRollbackRequired":            true,
		"driftAutoRepairRollbackTaskId":              "run-failed",
		"driftAutoRepairRollbackEvaluatedAt":         "2026-06-22T10:00:00Z",
		"driftAutoRepairSlaRollbackReviewDueMinutes": 30,
		"driftAutoRepairSlaNotificationRoutes":       []string{"rollback-review"},
		"driftAutoRepairSlaAutoTicketOnEscalation":   true,
	}

	now := time.Date(2026, 6, 22, 11, 0, 0, 0, time.UTC)
	cloudItsmDriftAutoRepairAppendSlaEvidence(evidence, now)
	if evidence["driftAutoRepairSlaAction"] != "review_rollback" || evidence["driftAutoRepairSlaStatus"] != "breached" {
		t.Fatalf("unexpected rollback SLA evidence: %#v", evidence)
	}
	if evidence["driftAutoRepairSlaEscalationRequired"] != true {
		t.Fatalf("expected SLA escalation, got %#v", evidence)
	}
	payload := cloudItsmDriftAutoRepairSlaEventPayload(evidence)
	routes := cloudSyncPolicyAttrStringSlice(payload["notificationRoutes"])
	if !cloudItsmTestStringSliceContains(routes, "sla_breach") || !cloudItsmTestStringSliceContains(routes, "drift_auto_repair_review_rollback") {
		t.Fatalf("expected SLA escalation routes, got %#v", payload)
	}
	if payload["itsmAutoTicket"] != true || payload["notificationEscalationReason"] != "review_rollback_sla_breached" {
		t.Fatalf("expected SLA escalation payload, got %#v", payload)
	}
}

func cloudItsmTestTimelineStage(t *testing.T, timeline []models.ResAttrs, stage string) models.ResAttrs {
	t.Helper()
	for _, item := range timeline {
		if item["stage"] == stage {
			return item
		}
	}
	return nil
}

func cloudItsmTestRecommendationAction(t *testing.T, recommendations []models.ResAttrs, action string) models.ResAttrs {
	t.Helper()
	for _, item := range recommendations {
		if item["action"] == action {
			return item
		}
	}
	return nil
}

func cloudItsmTestStringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestCloudItsmRiskStatusForDriftAutoRepairTaskStatus(t *testing.T) {
	cases := []struct {
		taskStatus string
		fallback   string
		want       string
	}{
		{taskStatus: models.TaskComplete, fallback: models.CloudRiskStatusOpen, want: models.CloudRiskStatusResolved},
		{taskStatus: models.TaskFailed, fallback: models.CloudRiskStatusResolved, want: models.CloudRiskStatusOpen},
		{taskStatus: models.TaskAborted, fallback: models.CloudRiskStatusResolved, want: models.CloudRiskStatusOpen},
		{taskStatus: models.TaskRejected, fallback: models.CloudRiskStatusResolved, want: models.CloudRiskStatusOpen},
		{taskStatus: models.TaskRunning, fallback: models.CloudRiskStatusInProgress, want: models.CloudRiskStatusInProgress},
		{taskStatus: models.TaskRunning, want: models.CloudRiskStatusInProgress},
	}
	for _, tc := range cases {
		got := cloudItsmRiskStatusForDriftAutoRepairTaskStatus(tc.taskStatus, tc.fallback)
		if got != tc.want {
			t.Fatalf("task status %s expected risk status %s, got %s", tc.taskStatus, tc.want, got)
		}
	}
}

func TestCloudItsmDriftAutoRepairTaskResultEvidence(t *testing.T) {
	startedAt := models.Time(time.Date(2026, 6, 22, 11, 10, 0, 0, time.UTC))
	endedAt := models.Time(time.Date(2026, 6, 22, 11, 20, 0, 0, time.UTC))
	syncedAt := models.Time(time.Date(2026, 6, 22, 11, 21, 0, 0, time.UTC))
	task := &models.Task{
		BaseTask: models.BaseTask{
			Status:  models.TaskFailed,
			Message: "terraform apply failed",
			StartAt: &startedAt,
			EndAt:   &endedAt,
		},
		IsDriftTask: true,
		Source:      consts.TaskSourceDriftApply,
	}
	task.Id = models.Id("t-drift-apply")
	task.Type = models.TaskTypeApply

	if !cloudItsmTaskIsDriftAutoRepair(task) {
		t.Fatalf("expected task to be detected as drift auto repair")
	}
	evidence := cloudItsmDriftAutoRepairTaskResultEvidence(task, models.CloudRiskStatusOpen, syncedAt)
	if evidence["driftAutoRepairTaskId"] != "t-drift-apply" || evidence["driftAutoRepairTaskStatus"] != models.TaskFailed {
		t.Fatalf("unexpected task result evidence: %#v", evidence)
	}
	if evidence["driftAutoRepairTaskFailed"] != true || evidence["driftAutoRepairTaskCompleted"] != false {
		t.Fatalf("unexpected completion flags: %#v", evidence)
	}
	if evidence["driftAutoRepairTaskMessage"] != "terraform apply failed" || evidence["driftAutoRepairTaskRiskStatus"] != models.CloudRiskStatusOpen {
		t.Fatalf("unexpected task message/risk status: %#v", evidence)
	}
}

func TestCloudItsmDriftAutoRepairTaskNeedsRetryApproval(t *testing.T) {
	for _, status := range []string{models.TaskFailed, models.TaskAborted, models.TaskRejected} {
		if !cloudItsmDriftAutoRepairTaskNeedsRetryApproval(&models.Task{BaseTask: models.BaseTask{Status: status}}) {
			t.Fatalf("expected status %s to need retry approval", status)
		}
	}
	for _, status := range []string{models.TaskComplete, models.TaskRunning, models.TaskPending} {
		if cloudItsmDriftAutoRepairTaskNeedsRetryApproval(&models.Task{BaseTask: models.BaseTask{Status: status}}) {
			t.Fatalf("did not expect status %s to need retry approval", status)
		}
	}
}

func TestCloudItsmDriftAutoRepairRetryAttemptAndMaxAttempts(t *testing.T) {
	if got := cloudItsmDriftAutoRepairRetryAttempt(models.ResAttrs{"driftAutoRepairRetryAttempt": "2"}); got != 2 {
		t.Fatalf("expected retry attempt 2, got %d", got)
	}
	if got := cloudItsmDriftAutoRepairRetryMaxAttempts(models.ResAttrs{}); got != cloudItsmDriftAutoRepairRetryDefaultMax {
		t.Fatalf("expected default max attempts, got %d", got)
	}
	if got := cloudItsmDriftAutoRepairRetryMaxAttempts(models.ResAttrs{"driftAutoRepairRetryMaxAttempts": 9}); got != cloudItsmDriftAutoRepairRetryMax {
		t.Fatalf("expected capped max attempts, got %d", got)
	}
	if got := cloudItsmDriftAutoRepairRetryMaxAttempts(models.ResAttrs{"driftAutoRepairMaxRetryAttempts": "2"}); got != 2 {
		t.Fatalf("expected legacy max attempts 2, got %d", got)
	}
}

func TestCloudItsmDriftAutoRepairRetryApprovalEvidence(t *testing.T) {
	checkedAt := models.Time(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	task := &models.Task{
		BaseTask: models.BaseTask{
			Status:  models.TaskFailed,
			Message: "terraform apply failed",
		},
		EnvId: models.Id("env-1"),
	}
	task.Id = models.Id("run-failed")

	evidence := cloudItsmDriftAutoRepairRetryApprovalEvidence(checkedAt, task, 1, 2, true, "")
	if !cloudItsmDriftAutoRepairRetryApprovalRequested(evidence) {
		t.Fatalf("expected retry approval requested, got %#v", evidence)
	}
	if evidence["driftAutoRepairRetryAttempt"] != 1 || evidence["driftAutoRepairRetryMaxAttempts"] != 2 {
		t.Fatalf("unexpected retry attempt evidence: %#v", evidence)
	}
	if evidence["driftAutoRepairRetryFromTaskId"] != "run-failed" || evidence["driftAutoRepairLastFailedTaskId"] != "run-failed" {
		t.Fatalf("unexpected failed task identity evidence: %#v", evidence)
	}
	if evidence["driftAutoRepairLastFailedTaskMessage"] != "terraform apply failed" || evidence["driftAutoRepairEnvId"] != "env-1" {
		t.Fatalf("unexpected failed task message/env evidence: %#v", evidence)
	}
}

func TestCloudItsmDriftAutoRepairRollbackEvidenceForCompleteTask(t *testing.T) {
	checkedAt := models.Time(time.Date(2026, 6, 22, 13, 0, 0, 0, time.UTC))
	task := &models.Task{
		BaseTask: models.BaseTask{
			Status: models.TaskComplete,
		},
	}
	task.Id = models.Id("run-complete")

	evidence := cloudItsmDriftAutoRepairRollbackEvidence(task, checkedAt)
	if cloudItsmDriftAutoRepairRollbackRequired(evidence) {
		t.Fatalf("did not expect completed task to require rollback, got %#v", evidence)
	}
	if evidence["driftAutoRepairRollbackStrategy"] != "no_platform_rollback_required" || evidence["driftAutoRepairRollbackSafetyLevel"] != "low" {
		t.Fatalf("unexpected completed rollback strategy: %#v", evidence)
	}
	if evidence["driftAutoRepairRollbackRequiresApproval"] != false {
		t.Fatalf("did not expect rollback approval for completed task: %#v", evidence)
	}
}

func TestCloudItsmDriftAutoRepairRollbackEvidenceForFailedPartialApply(t *testing.T) {
	checkedAt := models.Time(time.Date(2026, 6, 22, 13, 5, 0, 0, time.UTC))
	added := 1
	changed := 2
	task := &models.Task{
		BaseTask: models.BaseTask{
			Status:  models.TaskFailed,
			Message: "terraform apply failed after changes",
		},
		Result: models.TaskResult{
			ResAdded:   &added,
			ResChanged: &changed,
		},
	}
	task.Id = models.Id("run-failed-partial")

	evidence := cloudItsmDriftAutoRepairRollbackEvidence(task, checkedAt)
	if !cloudItsmDriftAutoRepairRollbackRequired(evidence) {
		t.Fatalf("expected failed partial task to require rollback review, got %#v", evidence)
	}
	if evidence["driftAutoRepairRollbackSafetyLevel"] != "high" || evidence["driftAutoRepairRollbackPartialApplySuspected"] != true {
		t.Fatalf("expected high safety rollback evidence, got %#v", evidence)
	}
	if evidence["driftAutoRepairRollbackRequiresApproval"] != true {
		t.Fatalf("expected rollback approval requirement, got %#v", evidence)
	}
	summary := modelResAttrs(evidence["driftAutoRepairRollbackApplyChanges"])
	if summary["added"] != 1 || summary["changed"] != 2 {
		t.Fatalf("unexpected rollback apply changes summary: %#v", summary)
	}
}
