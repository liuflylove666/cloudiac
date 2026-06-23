package apps

import (
	"os"
	"testing"
	"time"

	"cloudiac/portal/models"
)

func TestCloudBudgetEvaluationDue(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	if !cloudBudgetEvaluationDue(models.CloudBudget{EvaluationInterval: 3600}, now) {
		t.Fatal("expected never-evaluated budget to be due")
	}
	if !cloudBudgetEvaluationDue(models.CloudBudget{
		EvaluationInterval: 3600,
		LastEvaluatedAt:    models.Time(now.Add(-3600 * time.Second)),
	}, now) {
		t.Fatal("expected budget evaluated at interval boundary to be due")
	}
	if cloudBudgetEvaluationDue(models.CloudBudget{
		EvaluationInterval: 3600,
		LastEvaluatedAt:    models.Time(now.Add(-3599 * time.Second)),
	}, now) {
		t.Fatal("expected budget evaluated inside interval to be skipped")
	}
	if !cloudBudgetEvaluationDue(models.CloudBudget{
		LastEvaluatedAt: models.Time(now.Add(-3600 * time.Second)),
	}, now) {
		t.Fatal("expected default evaluation interval to be 3600 seconds")
	}
}

func TestCloudBudgetEvaluationWorkerInterval(t *testing.T) {
	const key = "CLOUDIAC_BUDGET_EVALUATION_WORKER_INTERVAL_SECONDS"
	original, hadOriginal := os.LookupEnv(key)
	defer func() {
		if hadOriginal {
			_ = os.Setenv(key, original)
		} else {
			_ = os.Unsetenv(key)
		}
	}()

	cases := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{name: "default", raw: "", want: cloudBudgetEvaluationWorkerDefaultInterval},
		{name: "invalid", raw: "abc", want: cloudBudgetEvaluationWorkerDefaultInterval},
		{name: "min", raw: "1", want: cloudBudgetEvaluationWorkerMinInterval},
		{name: "custom", raw: "120", want: 120 * time.Second},
		{name: "max", raw: "999999", want: cloudBudgetEvaluationWorkerMaxInterval},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.raw == "" {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, tc.raw)
			}
			if got := cloudBudgetEvaluationWorkerInterval(); got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestCloudBudgetEvaluateResultCount(t *testing.T) {
	result := models.ResAttrs{
		"int":     2,
		"int64":   int64(3),
		"float64": float64(4),
		"string":  "5",
	}
	if got := cloudBudgetEvaluateResultCount(result, "int"); got != 2 {
		t.Fatalf("expected int count 2, got %d", got)
	}
	if got := cloudBudgetEvaluateResultCount(result, "int64"); got != 3 {
		t.Fatalf("expected int64 count 3, got %d", got)
	}
	if got := cloudBudgetEvaluateResultCount(result, "float64"); got != 4 {
		t.Fatalf("expected float64 count 4, got %d", got)
	}
	if got := cloudBudgetEvaluateResultCount(result, "string"); got != 0 {
		t.Fatalf("expected unsupported count to be 0, got %d", got)
	}
	cloudBudgetAddEvaluateResultCount(result, "int", 5)
	if got := cloudBudgetEvaluateResultCount(result, "int"); got != 7 {
		t.Fatalf("expected accumulated count 7, got %d", got)
	}
}

func TestCloudBudgetRespAmounts(t *testing.T) {
	budget := models.CloudBudget{
		LimitAmount:      100,
		LastAmount:       125,
		ThresholdPercent: 80,
	}
	resp := cloudBudgetResp(nil, budget)
	if resp.CurrentAmount != 125 {
		t.Fatalf("expected current amount 125, got %f", resp.CurrentAmount)
	}
	if resp.UsagePercent != 125 {
		t.Fatalf("expected usage percent 125, got %f", resp.UsagePercent)
	}
	if resp.RemainingAmount != 0 {
		t.Fatalf("expected remaining amount to be capped at 0, got %f", resp.RemainingAmount)
	}
	if !resp.ThresholdReached || !resp.BudgetExceeded {
		t.Fatalf("expected threshold and budget exceeded flags, got %#v", resp)
	}
}

func TestCloudBudgetEvaluationLockName(t *testing.T) {
	if got := cloudBudgetEvaluationLockName("", "2026-06", "CNY"); got != "cloudiac:budget_evaluation:unknown" {
		t.Fatalf("unexpected empty org lock name: %s", got)
	}
	got := cloudBudgetEvaluationLockName(models.Id("org-1"), "2026-06", "CNY")
	want := "cloudiac:budget_evaluation:org-1:2026-06:CNY"
	if got != want {
		t.Fatalf("expected lock name %s, got %s", want, got)
	}
}
