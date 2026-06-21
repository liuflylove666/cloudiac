// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/models"
)

func (a cloudComputeInstanceOperationAdapter) Execute(request cloudProviderOperationRequest, state cloudProviderResourceState, account *cmdbCloudAccount) (models.ResAttrs, e.Error) {
	result := models.ResAttrs{
		"providerAdapter": a.Key(),
		"request":         cloudProviderOperationRequestAttrs(request, state),
		"preState":        cloudProviderResourceStateAttrs(state),
	}
	if !cloudOperationProviderWriteEnabled() {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   fmt.Sprintf("%s is not enabled", cloudOperationProviderWriteEnv),
		})
		return result, e.New(e.BadParam, fmt.Errorf("云端写操作保护开关 %s 未开启", cloudOperationProviderWriteEnv))
	}
	if request.ReadMode != cloudOperationProviderReadModeLive || state.ReadMode != cloudOperationProviderReadModeLive {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   fmt.Sprintf("provider write requires %s=live", cloudOperationProviderReadModeEnv),
		})
		return result, e.New(e.BadParam, fmt.Errorf("云端写操作要求 %s=live", cloudOperationProviderReadModeEnv))
	}
	if state.ErrorCode != "" {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "provider state read failed before write",
			"error": models.ResAttrs{
				"code":      state.ErrorCode,
				"message":   state.ErrorMessage,
				"retryable": state.Retryable,
			},
		})
		return result, e.New(e.BadParam, fmt.Errorf("provider 状态读取失败，拒绝执行写操作：%s", firstNonEmpty(state.ErrorMessage, state.ErrorCode)))
	}
	if account == nil || !account.Ready {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "cloud account is not ready",
		})
		return result, e.New(e.BadParam, fmt.Errorf("云账号不可用，拒绝执行 provider 写操作"))
	}
	if request.Action == models.CloudOperationActionResizeInstance {
		return a.executeInstanceResize(request, state, account, result)
	}
	if cloudProviderComputeActionNoop(request, state) {
		result["providerAdapterExecuted"] = false
		result["idempotentNoop"] = true
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed":        false,
			"reason":          "resource already in target state",
			"normalizedState": state.NormalizedState,
			"targetState":     request.TargetState,
		})
		result["postState"] = cloudProviderResourceStateAttrs(state)
		result["message"] = "实例已处于目标状态，本次 provider 写操作按幂等空操作处理"
		return result, nil
	}

	writeResp, err := cloudProviderExecuteComputeAction(account, request)
	result["providerAdapterExecuted"] = writeResp["executed"]
	result["providerWriteRequest"] = cloudProviderSanitizeAttrs(modelResAttrs(writeResp["request"]))
	result["rawResponse"] = cloudProviderSanitizeAttrs(writeResp)
	if err != nil {
		return result, err
	}

	poll := cloudProviderPollComputeState(request, account)
	result["postState"] = cloudProviderResourceStateAttrs(poll.FinalState)
	result["statePolling"] = poll.Attrs()
	if poll.Err != nil {
		result["message"] = fmt.Sprintf("provider adapter %s 已提交 %s 写操作，但最终态确认失败", a.Key(), request.Action)
		return result, e.New(e.BadParam, poll.Err)
	}
	result["message"] = fmt.Sprintf("provider adapter %s 已提交 %s 写操作，资源已到达目标状态 %s", a.Key(), request.Action, request.TargetState)
	return result, nil
}

func (a cloudComputeInstanceOperationAdapter) executeInstanceResize(request cloudProviderOperationRequest, state cloudProviderResourceState, account *cmdbCloudAccount, result models.ResAttrs) (models.ResAttrs, e.Error) {
	targetSpec, ok := cloudProviderInstanceTargetSpec(request)
	if !ok {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "params.targetInstanceType is required",
		})
		return result, e.New(e.BadParam, fmt.Errorf("实例规格调整动作需要提交 params.targetInstanceType"))
	}
	currentSpec := cloudProviderComputeCurrentSpec(state)
	if currentSpec != "" && currentSpec == targetSpec {
		result["providerAdapterExecuted"] = false
		result["idempotentNoop"] = true
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed":            false,
			"reason":              "instance already uses target spec",
			"currentInstanceType": currentSpec,
			"targetInstanceType":  targetSpec,
		})
		result["postState"] = cloudProviderResourceStateAttrs(state)
		result["message"] = "实例已处于目标规格，本次 provider 写操作按幂等空操作处理"
		return result, nil
	}
	if state.NormalizedState != "stopped" {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed":        false,
			"reason":          "instance must be stopped before resize",
			"normalizedState": state.NormalizedState,
			"rawState":        state.RawState,
		})
		return result, e.New(e.BadParam, fmt.Errorf("实例规格调整要求实例处于停止状态，当前状态为 %s", firstNonEmpty(state.RawState, state.NormalizedState, "unknown")))
	}

	writeResp, err := cloudProviderExecuteInstanceResize(account, request, targetSpec)
	result["providerAdapterExecuted"] = writeResp["executed"]
	result["providerWriteRequest"] = cloudProviderSanitizeAttrs(modelResAttrs(writeResp["request"]))
	result["rawResponse"] = cloudProviderSanitizeAttrs(writeResp)
	if err != nil {
		return result, err
	}

	poll := cloudProviderPollComputeState(request, account)
	result["postState"] = cloudProviderResourceStateAttrs(poll.FinalState)
	result["statePolling"] = poll.Attrs()
	if poll.Err != nil {
		result["message"] = fmt.Sprintf("provider adapter %s 已提交实例规格调整写操作，但最终规格确认失败", a.Key())
		return result, e.New(e.BadParam, poll.Err)
	}
	result["message"] = fmt.Sprintf("provider adapter %s 已提交实例规格调整写操作，实例规格已达到 %s", a.Key(), targetSpec)
	return result, nil
}

func (a cloudBlockVolumeOperationAdapter) Execute(request cloudProviderOperationRequest, state cloudProviderResourceState, account *cmdbCloudAccount) (models.ResAttrs, e.Error) {
	result := models.ResAttrs{
		"providerAdapter": a.Key(),
		"request":         cloudProviderOperationRequestAttrs(request, state),
		"preState":        cloudProviderResourceStateAttrs(state),
	}
	if !cloudOperationProviderWriteEnabled() {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   fmt.Sprintf("%s is not enabled", cloudOperationProviderWriteEnv),
		})
		return result, e.New(e.BadParam, fmt.Errorf("云端写操作保护开关 %s 未开启", cloudOperationProviderWriteEnv))
	}
	if request.ReadMode != cloudOperationProviderReadModeLive || state.ReadMode != cloudOperationProviderReadModeLive {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   fmt.Sprintf("provider write requires %s=live", cloudOperationProviderReadModeEnv),
		})
		return result, e.New(e.BadParam, fmt.Errorf("云端写操作要求 %s=live", cloudOperationProviderReadModeEnv))
	}
	if state.ErrorCode != "" {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "provider state read failed before write",
			"error": models.ResAttrs{
				"code":      state.ErrorCode,
				"message":   state.ErrorMessage,
				"retryable": state.Retryable,
			},
		})
		return result, e.New(e.BadParam, fmt.Errorf("provider 状态读取失败，拒绝执行磁盘扩容：%s", firstNonEmpty(state.ErrorMessage, state.ErrorCode)))
	}
	if account == nil || !account.Ready {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "cloud account is not ready",
		})
		return result, e.New(e.BadParam, fmt.Errorf("云账号不可用，拒绝执行 provider 磁盘扩容"))
	}
	if request.Action == models.CloudOperationActionCreateSnapshot {
		return a.executeVolumeSnapshot(request, state, account, result)
	}
	targetSize, ok := cloudProviderVolumeTargetSizeGiB(request)
	if !ok {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "params.targetSizeGiB is required",
		})
		return result, e.New(e.BadParam, fmt.Errorf("磁盘扩容动作需要提交 params.targetSizeGiB 正整数"))
	}
	currentSize := cloudProviderVolumeCurrentSizeGiB(request, state)
	if currentSize <= 0 {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "current volume size is unknown",
		})
		return result, e.New(e.BadParam, fmt.Errorf("未读取到当前磁盘容量，拒绝执行 provider 扩容"))
	}
	if currentSize == targetSize {
		result["providerAdapterExecuted"] = false
		result["idempotentNoop"] = true
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed":       false,
			"reason":         "volume already at target size",
			"currentSizeGiB": currentSize,
			"targetSizeGiB":  targetSize,
		})
		result["postState"] = cloudProviderResourceStateAttrs(state)
		result["message"] = "磁盘已处于目标容量，本次 provider 写操作按幂等空操作处理"
		return result, nil
	}
	if currentSize > targetSize {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed":       false,
			"reason":         "volume shrink is not supported",
			"currentSizeGiB": currentSize,
			"targetSizeGiB":  targetSize,
		})
		return result, e.New(e.BadParam, fmt.Errorf("磁盘扩容目标容量 %d GiB 必须大于当前容量 %d GiB", targetSize, currentSize))
	}

	writeResp, err := cloudProviderExecuteVolumeResize(account, request, targetSize)
	result["providerAdapterExecuted"] = writeResp["executed"]
	result["providerWriteRequest"] = cloudProviderSanitizeAttrs(modelResAttrs(writeResp["request"]))
	result["rawResponse"] = cloudProviderSanitizeAttrs(writeResp)
	if err != nil {
		return result, err
	}

	poll := cloudProviderPollVolumeState(request, account)
	result["postState"] = cloudProviderResourceStateAttrs(poll.FinalState)
	result["statePolling"] = poll.Attrs()
	if poll.Err != nil {
		result["message"] = fmt.Sprintf("provider adapter %s 已提交磁盘扩容写操作，但最终容量确认失败", a.Key())
		return result, e.New(e.BadParam, poll.Err)
	}
	result["message"] = fmt.Sprintf("provider adapter %s 已提交磁盘扩容写操作，容量已达到 %d GiB", a.Key(), targetSize)
	return result, nil
}

func (a cloudBlockVolumeOperationAdapter) executeVolumeSnapshot(request cloudProviderOperationRequest, state cloudProviderResourceState, account *cmdbCloudAccount, result models.ResAttrs) (models.ResAttrs, e.Error) {
	snapshotName := cloudProviderSnapshotName(request)
	writeResp, err := cloudProviderExecuteVolumeSnapshot(account, request, snapshotName)
	result["providerAdapterExecuted"] = writeResp["executed"]
	result["providerWriteRequest"] = cloudProviderSanitizeAttrs(modelResAttrs(writeResp["request"]))
	result["rawResponse"] = cloudProviderSanitizeAttrs(writeResp)
	result["postState"] = cloudProviderResourceStateAttrs(state)
	if err != nil {
		return result, err
	}
	result["message"] = fmt.Sprintf("provider adapter %s 已提交快照/备份创建请求 %s", a.Key(), snapshotName)
	return result, nil
}

func (a cloudSecurityGroupOperationAdapter) Execute(request cloudProviderOperationRequest, state cloudProviderResourceState, account *cmdbCloudAccount) (models.ResAttrs, e.Error) {
	result := models.ResAttrs{
		"providerAdapter": a.Key(),
		"request":         cloudProviderOperationRequestAttrs(request, state),
		"preState":        cloudProviderResourceStateAttrs(state),
	}
	if !cloudOperationProviderWriteEnabled() {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   fmt.Sprintf("%s is not enabled", cloudOperationProviderWriteEnv),
		})
		return result, e.New(e.BadParam, fmt.Errorf("云端写操作保护开关 %s 未开启", cloudOperationProviderWriteEnv))
	}
	if request.ReadMode != cloudOperationProviderReadModeLive || state.ReadMode != cloudOperationProviderReadModeLive {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   fmt.Sprintf("provider write requires %s=live", cloudOperationProviderReadModeEnv),
		})
		return result, e.New(e.BadParam, fmt.Errorf("云端写操作要求 %s=live", cloudOperationProviderReadModeEnv))
	}
	if state.ErrorCode != "" {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "provider state read failed before write",
			"error": models.ResAttrs{
				"code":      state.ErrorCode,
				"message":   state.ErrorMessage,
				"retryable": state.Retryable,
			},
		})
		return result, e.New(e.BadParam, fmt.Errorf("provider 状态读取失败，拒绝执行安全组规则写操作：%s", firstNonEmpty(state.ErrorMessage, state.ErrorCode)))
	}
	if account == nil || !account.Ready {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   "cloud account is not ready",
		})
		return result, e.New(e.BadParam, fmt.Errorf("云账号不可用，拒绝执行 provider 安全组规则写操作"))
	}
	rule, errs := cloudSecurityRuleOperationFromParams(request.Params)
	if len(errs) > 0 {
		result["providerAdapterExecuted"] = false
		result["rawResponse"] = cloudProviderSanitizeAttrs(models.ResAttrs{
			"executed": false,
			"reason":   strings.Join(errs, "；"),
		})
		return result, e.New(e.BadParam, fmt.Errorf("更新安全组规则参数无效：%s", strings.Join(errs, "；")))
	}

	writeResp, err := cloudProviderExecuteSecurityRuleUpdate(account, request, rule)
	result["providerAdapterExecuted"] = writeResp["executed"]
	result["providerWriteRequest"] = cloudProviderSanitizeAttrs(modelResAttrs(writeResp["request"]))
	result["rawResponse"] = cloudProviderSanitizeAttrs(writeResp)
	result["postState"] = cloudProviderResourceStateAttrs(state)
	if err != nil {
		return result, err
	}
	result["message"] = fmt.Sprintf("provider adapter %s 已提交安全组规则 %s 请求", a.Key(), rule.Operation)
	return result, nil
}

type cloudProviderStatePollResult struct {
	Attempts    []models.ResAttrs
	FinalState  cloudProviderResourceState
	TargetState string
	Matched     bool
	TimedOut    bool
	Err         error
}

func (r cloudProviderStatePollResult) Attrs() models.ResAttrs {
	attempts := r.Attempts
	if attempts == nil {
		attempts = []models.ResAttrs{}
	}
	return models.ResAttrs{
		"attempts":    attempts,
		"targetState": r.TargetState,
		"matched":     r.Matched,
		"timedOut":    r.TimedOut,
		"finalState":  cloudProviderResourceStateAttrs(r.FinalState),
		"error":       cloudProviderPollErrorAttrs(r.Err),
	}
}

func cloudProviderPollComputeState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderStatePollResult {
	maxAttempts := cloudOperationProviderPollAttempts()
	delay := cloudOperationProviderPollInterval()
	maxDelay := cloudOperationProviderPollMaxDelay()
	targetState := strings.TrimSpace(request.TargetState)
	targetSpec := ""
	if request.Action == models.CloudOperationActionResizeInstance {
		if value, ok := cloudProviderInstanceTargetSpec(request); ok {
			targetSpec = value
			targetState = value
		}
	}
	result := cloudProviderStatePollResult{
		Attempts:    make([]models.ResAttrs, 0, maxAttempts),
		TargetState: targetState,
	}
	if targetState == "" {
		result.Err = fmt.Errorf("provider 写操作缺少目标状态，无法确认最终态")
		return result
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(delay)
			delay = cloudProviderNextPollDelay(delay, maxDelay)
		}

		state := cloudProviderLiveComputeState(request, account)
		result.FinalState = state
		currentSpec := cloudProviderComputeCurrentSpec(state)
		matched := state.NormalizedState == targetState
		if request.Action == models.CloudOperationActionResizeInstance {
			matched = targetSpec != "" && currentSpec == targetSpec
		}
		result.Attempts = append(result.Attempts, models.ResAttrs{
			"attempt":             attempt,
			"rawState":            state.RawState,
			"normalizedState":     state.NormalizedState,
			"currentInstanceType": currentSpec,
			"targetState":         targetState,
			"matched":             matched,
			"readError": models.ResAttrs{
				"code":      state.ErrorCode,
				"message":   state.ErrorMessage,
				"retryable": state.Retryable,
			},
		})
		if matched {
			result.Matched = true
			return result
		}
		if state.ErrorCode != "" && !state.Retryable {
			result.Err = fmt.Errorf("provider 最终态确认失败：%s", firstNonEmpty(state.ErrorMessage, state.ErrorCode))
			return result
		}
	}

	result.TimedOut = true
	if request.Action == models.CloudOperationActionResizeInstance {
		result.Err = fmt.Errorf("provider 写操作已提交，但实例未在 %d 次轮询内达到目标规格 %s，最后规格为 %s",
			maxAttempts, targetSpec, firstNonEmpty(cloudProviderComputeCurrentSpec(result.FinalState), "unknown"))
		return result
	}
	result.Err = fmt.Errorf("provider 写操作已提交，但资源未在 %d 次轮询内到达目标状态 %s，最后状态为 %s",
		maxAttempts, targetState, firstNonEmpty(result.FinalState.NormalizedState, result.FinalState.RawState, "unknown"))
	return result
}

func cloudProviderPollVolumeState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderStatePollResult {
	maxAttempts := cloudOperationProviderPollAttempts()
	delay := cloudOperationProviderPollInterval()
	maxDelay := cloudOperationProviderPollMaxDelay()
	targetSize, ok := cloudProviderVolumeTargetSizeGiB(request)
	result := cloudProviderStatePollResult{
		Attempts:    make([]models.ResAttrs, 0, maxAttempts),
		TargetState: cloudVolumeStateValue(targetSize),
	}
	if !ok {
		result.Err = fmt.Errorf("provider 写操作缺少目标容量，无法确认最终态")
		return result
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(delay)
			delay = cloudProviderNextPollDelay(delay, maxDelay)
		}

		state := cloudProviderLiveVolumeState(request, account)
		result.FinalState = state
		currentSize := cloudProviderVolumeCurrentSizeGiB(request, state)
		matched := currentSize >= targetSize
		result.Attempts = append(result.Attempts, models.ResAttrs{
			"attempt":         attempt,
			"rawState":        state.RawState,
			"normalizedState": state.NormalizedState,
			"currentSizeGiB":  currentSize,
			"targetSizeGiB":   targetSize,
			"matched":         matched,
			"readError": models.ResAttrs{
				"code":      state.ErrorCode,
				"message":   state.ErrorMessage,
				"retryable": state.Retryable,
			},
		})
		if matched {
			result.Matched = true
			return result
		}
		if state.ErrorCode != "" && !state.Retryable {
			result.Err = fmt.Errorf("provider 容量确认失败：%s", firstNonEmpty(state.ErrorMessage, state.ErrorCode))
			return result
		}
	}

	result.TimedOut = true
	result.Err = fmt.Errorf("provider 写操作已提交，但磁盘未在 %d 次轮询内达到目标容量 %d GiB，最后容量为 %d GiB",
		maxAttempts, targetSize, cloudProviderVolumeCurrentSizeGiB(request, result.FinalState))
	return result
}

func cloudProviderExecuteComputeAction(account *cmdbCloudAccount, request cloudProviderOperationRequest) (models.ResAttrs, e.Error) {
	writeCtx, cancel := context.WithTimeout(context.Background(), cloudOperationProviderWriteTimeout())
	defer cancel()
	switch normalizeProvider(request.Provider) {
	case "aws":
		return cloudProviderExecuteAwsComputeAction(writeCtx, account, request)
	case "oci":
		return cloudProviderExecuteOciComputeAction(writeCtx, account, request)
	case "alicloud":
		return cloudProviderExecuteAlicloudComputeAction(writeCtx, account, request)
	case "azure":
		return cloudProviderExecuteAzureComputeAction(writeCtx, account, request)
	case "gcp":
		return cloudProviderExecuteGcpComputeAction(writeCtx, account, request)
	default:
		return models.ResAttrs{"executed": false, "provider": request.Provider}, e.New(e.BadParam, fmt.Errorf("暂不支持 %s provider 写操作", request.Provider))
	}
}

func cloudProviderExecuteVolumeResize(account *cmdbCloudAccount, request cloudProviderOperationRequest, targetSizeGiB int) (models.ResAttrs, e.Error) {
	writeCtx, cancel := context.WithTimeout(context.Background(), cloudOperationProviderWriteTimeout())
	defer cancel()
	switch normalizeProvider(request.Provider) {
	case "aws":
		return cloudProviderExecuteAwsVolumeResize(writeCtx, account, request, targetSizeGiB)
	case "oci":
		return cloudProviderExecuteOciVolumeResize(writeCtx, account, request, targetSizeGiB)
	case "alicloud":
		return cloudProviderExecuteAlicloudVolumeResize(writeCtx, account, request, targetSizeGiB)
	default:
		return models.ResAttrs{"executed": false, "provider": request.Provider}, e.New(e.BadParam, fmt.Errorf("暂不支持 %s provider 磁盘扩容", request.Provider))
	}
}

func cloudProviderExecuteInstanceResize(account *cmdbCloudAccount, request cloudProviderOperationRequest, targetSpec string) (models.ResAttrs, e.Error) {
	writeCtx, cancel := context.WithTimeout(context.Background(), cloudOperationProviderWriteTimeout())
	defer cancel()
	switch normalizeProvider(request.Provider) {
	case "aws":
		return cloudProviderExecuteAwsInstanceResize(writeCtx, account, request, targetSpec)
	case "oci":
		return cloudProviderExecuteOciInstanceResize(writeCtx, account, request, targetSpec)
	case "alicloud":
		return cloudProviderExecuteAlicloudInstanceResize(writeCtx, account, request, targetSpec)
	default:
		return models.ResAttrs{"executed": false, "provider": request.Provider}, e.New(e.BadParam, fmt.Errorf("暂不支持 %s provider 实例规格调整", request.Provider))
	}
}

func cloudProviderExecuteVolumeSnapshot(account *cmdbCloudAccount, request cloudProviderOperationRequest, snapshotName string) (models.ResAttrs, e.Error) {
	writeCtx, cancel := context.WithTimeout(context.Background(), cloudOperationProviderWriteTimeout())
	defer cancel()
	switch normalizeProvider(request.Provider) {
	case "aws":
		return cloudProviderExecuteAwsVolumeSnapshot(writeCtx, account, request, snapshotName)
	case "oci":
		return cloudProviderExecuteOciVolumeBackup(writeCtx, account, request, snapshotName)
	case "alicloud":
		return cloudProviderExecuteAlicloudVolumeSnapshot(writeCtx, account, request, snapshotName)
	default:
		return models.ResAttrs{"executed": false, "provider": request.Provider}, e.New(e.BadParam, fmt.Errorf("暂不支持 %s provider 创建快照/备份", request.Provider))
	}
}

func cloudProviderExecuteSecurityRuleUpdate(account *cmdbCloudAccount, request cloudProviderOperationRequest, rule cloudSecurityRuleOperation) (models.ResAttrs, e.Error) {
	writeCtx, cancel := context.WithTimeout(context.Background(), cloudOperationProviderWriteTimeout())
	defer cancel()
	switch normalizeProvider(request.Provider) {
	case "aws":
		return cloudProviderExecuteAwsSecurityRuleUpdate(writeCtx, account, request, rule)
	case "oci":
		return cloudProviderExecuteOciSecurityRuleUpdate(writeCtx, account, request, rule)
	case "alicloud":
		return cloudProviderExecuteAlicloudSecurityRuleUpdate(writeCtx, account, request, rule)
	default:
		return models.ResAttrs{"executed": false, "provider": request.Provider}, e.New(e.BadParam, fmt.Errorf("暂不支持 %s provider 安全组规则写操作", request.Provider))
	}
}

func cloudProviderExecuteAwsComputeAction(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) (models.ResAttrs, e.Error) {
	action, ok := cloudProviderAwsComputeAction(request.Action)
	if !ok {
		return models.ResAttrs{"executed": false, "provider": "aws"}, e.New(e.BadParam, fmt.Errorf("AWS EC2 不支持动作 %s", request.Action))
	}
	values := url.Values{
		"Action":       []string{action},
		"Version":      []string{awsEC2APIVersion},
		"InstanceId.1": []string{request.ResourceId},
	}
	resp := awsEC2ActionResponse{}
	err := awsQueryAPIWithValues(c, account, "ec2", request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "aws",
		"service":  "ec2",
		"action":   action,
		"request": models.ResAttrs{
			"region":     request.Region,
			"instanceId": request.ResourceId,
		},
		"response": models.ResAttrs{
			"requestId": firstNonEmpty(resp.RequestId, resp.ResponseMetadata.RequestId),
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("AWS EC2 %s 失败：%s", action, err.Error()))
	}
	return raw, nil
}

type awsEC2ActionResponse struct {
	RequestId        string `xml:"requestId"`
	ResponseMetadata struct {
		RequestId string `xml:"requestId"`
	} `xml:"ResponseMetadata"`
}

type awsEC2ModifyVolumeResponse struct {
	RequestId          string `xml:"requestId"`
	VolumeModification struct {
		VolumeId          string `xml:"volumeId"`
		ModificationState string `xml:"modificationState"`
		TargetSize        int    `xml:"targetSize"`
		OriginalSize      int    `xml:"originalSize"`
		Progress          int    `xml:"progress"`
	} `xml:"volumeModification"`
}

type awsEC2CreateSnapshotResponse struct {
	RequestId   string `xml:"requestId"`
	SnapshotId  string `xml:"snapshotId"`
	VolumeId    string `xml:"volumeId"`
	Status      string `xml:"status"`
	StartTime   string `xml:"startTime"`
	Description string `xml:"description"`
}

func cloudProviderExecuteAwsInstanceResize(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, targetSpec string) (models.ResAttrs, e.Error) {
	values := url.Values{
		"Action":             []string{"ModifyInstanceAttribute"},
		"Version":            []string{awsEC2APIVersion},
		"InstanceId":         []string{request.ResourceId},
		"InstanceType.Value": []string{targetSpec},
	}
	resp := awsEC2ActionResponse{}
	err := awsQueryAPIWithValues(c, account, "ec2", request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "aws",
		"service":  "ec2",
		"action":   "ModifyInstanceAttribute",
		"request": models.ResAttrs{
			"region":             request.Region,
			"instanceId":         request.ResourceId,
			"targetInstanceType": targetSpec,
		},
		"response": models.ResAttrs{
			"requestId": firstNonEmpty(resp.RequestId, resp.ResponseMetadata.RequestId),
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("AWS EC2 ModifyInstanceAttribute 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteAwsVolumeResize(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, targetSizeGiB int) (models.ResAttrs, e.Error) {
	values := url.Values{
		"Action":   []string{"ModifyVolume"},
		"Version":  []string{awsEC2APIVersion},
		"VolumeId": []string{request.ResourceId},
		"Size":     []string{strconv.Itoa(targetSizeGiB)},
	}
	resp := awsEC2ModifyVolumeResponse{}
	err := awsQueryAPIWithValues(c, account, "ec2", request.Region, values, &resp)
	modification := resp.VolumeModification
	raw := models.ResAttrs{
		"executed": true,
		"provider": "aws",
		"service":  "ec2",
		"action":   "ModifyVolume",
		"request": models.ResAttrs{
			"region":        request.Region,
			"volumeId":      request.ResourceId,
			"targetSizeGiB": targetSizeGiB,
		},
		"response": models.ResAttrs{
			"requestId":         resp.RequestId,
			"volumeId":          firstNonEmpty(modification.VolumeId, request.ResourceId),
			"modificationState": modification.ModificationState,
			"originalSizeGiB":   modification.OriginalSize,
			"targetSizeGiB":     modification.TargetSize,
			"progress":          modification.Progress,
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("AWS EBS ModifyVolume 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteAwsVolumeSnapshot(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, snapshotName string) (models.ResAttrs, e.Error) {
	description := cloudProviderSnapshotDescription(request)
	values := url.Values{
		"Action":      []string{"CreateSnapshot"},
		"Version":     []string{awsEC2APIVersion},
		"VolumeId":    []string{request.ResourceId},
		"Description": []string{description},
	}
	if snapshotName != "" {
		values.Set("TagSpecification.1.ResourceType", "snapshot")
		values.Set("TagSpecification.1.Tag.1.Key", "Name")
		values.Set("TagSpecification.1.Tag.1.Value", snapshotName)
	}
	resp := awsEC2CreateSnapshotResponse{}
	err := awsQueryAPIWithValues(c, account, "ec2", request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "aws",
		"service":  "ec2",
		"action":   "CreateSnapshot",
		"request": models.ResAttrs{
			"region":       request.Region,
			"volumeId":     request.ResourceId,
			"snapshotName": snapshotName,
			"description":  description,
		},
		"response": models.ResAttrs{
			"requestId":   resp.RequestId,
			"snapshotId":  resp.SnapshotId,
			"volumeId":    firstNonEmpty(resp.VolumeId, request.ResourceId),
			"status":      resp.Status,
			"startTime":   resp.StartTime,
			"description": resp.Description,
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("AWS EBS CreateSnapshot 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteAwsSecurityRuleUpdate(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, rule cloudSecurityRuleOperation) (models.ResAttrs, e.Error) {
	action := "AuthorizeSecurityGroupIngress"
	if rule.Operation == "revoke" && rule.Direction == "ingress" {
		action = "RevokeSecurityGroupIngress"
	} else if rule.Operation == "authorize" && rule.Direction == "egress" {
		action = "AuthorizeSecurityGroupEgress"
	} else if rule.Operation == "revoke" && rule.Direction == "egress" {
		action = "RevokeSecurityGroupEgress"
	}
	values := url.Values{
		"Action":                     []string{action},
		"Version":                    []string{awsEC2APIVersion},
		"GroupId":                    []string{request.ResourceId},
		"IpPermissions.1.IpProtocol": []string{cloudSecurityRuleAwsProtocol(rule.Protocol)},
	}
	if rule.FromPort >= 0 && rule.ToPort >= 0 {
		values.Set("IpPermissions.1.FromPort", strconv.Itoa(rule.FromPort))
		values.Set("IpPermissions.1.ToPort", strconv.Itoa(rule.ToPort))
	}
	if strings.Contains(rule.Cidr, ":") {
		values.Set("IpPermissions.1.Ipv6Ranges.1.CidrIpv6", rule.Cidr)
		if rule.Description != "" && rule.Operation == "authorize" {
			values.Set("IpPermissions.1.Ipv6Ranges.1.Description", rule.Description)
		}
	} else {
		values.Set("IpPermissions.1.IpRanges.1.CidrIp", rule.Cidr)
		if rule.Description != "" && rule.Operation == "authorize" {
			values.Set("IpPermissions.1.IpRanges.1.Description", rule.Description)
		}
	}
	resp := awsEC2ActionResponse{}
	err := awsQueryAPIWithValues(c, account, "ec2", request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "aws",
		"service":  "ec2",
		"action":   action,
		"request": models.ResAttrs{
			"region":      request.Region,
			"groupId":     request.ResourceId,
			"operation":   rule.Operation,
			"direction":   rule.Direction,
			"protocol":    rule.Protocol,
			"cidr":        rule.Cidr,
			"portRange":   cloudSecurityRulePortText(rule),
			"description": rule.Description,
		},
		"response": models.ResAttrs{
			"requestId": firstNonEmpty(resp.RequestId, resp.ResponseMetadata.RequestId),
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("AWS EC2 %s 失败：%s", action, err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteOciComputeAction(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) (models.ResAttrs, e.Error) {
	action, ok := cloudProviderOciComputeAction(request.Action)
	if !ok {
		return models.ResAttrs{"executed": false, "provider": "oci"}, e.New(e.BadParam, fmt.Errorf("OCI Compute 不支持动作 %s", request.Action))
	}
	headers, err := ociInstanceActionAPI(c, account, request.Region, request.ResourceId, action)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "oci",
		"service":  "iaas",
		"action":   "InstanceAction",
		"request": models.ResAttrs{
			"region":     request.Region,
			"instanceId": request.ResourceId,
			"action":     action,
		},
		"response": models.ResAttrs{
			"requestId":     headers.Get("opc-request-id"),
			"workRequestId": headers.Get("opc-work-request-id"),
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("OCI Compute %s 失败：%s", action, err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteOciInstanceResize(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, targetSpec string) (models.ResAttrs, e.Error) {
	headers, body, err := ociUpdateInstanceShapeAPI(c, account, request.Region, request.ResourceId, targetSpec)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "oci",
		"service":  "iaas",
		"action":   "UpdateInstance",
		"request": models.ResAttrs{
			"region":      request.Region,
			"instanceId":  request.ResourceId,
			"targetShape": targetSpec,
		},
		"response": models.ResAttrs{
			"requestId":      headers.Get("opc-request-id"),
			"workRequestId":  headers.Get("opc-work-request-id"),
			"instanceId":     firstNonEmpty(ociAttrString(body, "id"), request.ResourceId),
			"lifecycleState": ociAttrString(body, "lifecycleState"),
			"shape":          ociAttrString(body, "shape"),
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("OCI Compute UpdateInstance 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteOciVolumeResize(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, targetSizeGiB int) (models.ResAttrs, e.Error) {
	headers, body, err := ociUpdateVolumeAPI(c, account, request.Region, request.ResourceId, targetSizeGiB)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "oci",
		"service":  "iaas",
		"action":   "UpdateVolume",
		"request": models.ResAttrs{
			"region":        request.Region,
			"volumeId":      request.ResourceId,
			"targetSizeGiB": targetSizeGiB,
		},
		"response": models.ResAttrs{
			"requestId":      headers.Get("opc-request-id"),
			"workRequestId":  headers.Get("opc-work-request-id"),
			"volumeId":       firstNonEmpty(ociAttrString(body, "id"), request.ResourceId),
			"lifecycleState": ociAttrString(body, "lifecycleState"),
			"sizeInGBs":      body["sizeInGBs"],
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("OCI Block Volume UpdateVolume 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteOciVolumeBackup(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, backupName string) (models.ResAttrs, e.Error) {
	backupType := cloudProviderSnapshotBackupType(request)
	headers, body, err := ociCreateVolumeBackupAPI(c, account, request.Region, request.ResourceId, backupName, backupType)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "oci",
		"service":  "iaas",
		"action":   "CreateVolumeBackup",
		"request": models.ResAttrs{
			"region":      request.Region,
			"volumeId":    request.ResourceId,
			"displayName": backupName,
			"backupType":  backupType,
		},
		"response": models.ResAttrs{
			"requestId":      headers.Get("opc-request-id"),
			"workRequestId":  headers.Get("opc-work-request-id"),
			"backupId":       ociAttrString(body, "id"),
			"volumeId":       firstNonEmpty(ociAttrString(body, "volumeId"), request.ResourceId),
			"displayName":    ociAttrString(body, "displayName"),
			"lifecycleState": ociAttrString(body, "lifecycleState"),
			"type":           ociAttrString(body, "type"),
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("OCI Block Volume CreateVolumeBackup 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteOciSecurityRuleUpdate(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, rule cloudSecurityRuleOperation) (models.ResAttrs, e.Error) {
	if strings.HasPrefix(strings.TrimSpace(request.ResourceId), "ocid1.securitylist") {
		return models.ResAttrs{
			"executed": false,
			"provider": "oci",
			"service":  "iaas",
			"action":   "SecurityListRuleUpdateBlocked",
			"request": models.ResAttrs{
				"region":         request.Region,
				"securityListId": request.ResourceId,
			},
		}, e.New(e.BadParam, fmt.Errorf("OCI Security List 规则写操作需要全量替换规则列表，当前仅开放 Network Security Group 规则写操作"))
	}
	headers, body, err := ociNetworkSecurityGroupRuleAPI(c, account, request.Region, request.ResourceId, rule)
	action := "AddNetworkSecurityGroupSecurityRules"
	if rule.Operation == "revoke" {
		action = "RemoveNetworkSecurityGroupSecurityRules"
	}
	raw := models.ResAttrs{
		"executed": true,
		"provider": "oci",
		"service":  "iaas",
		"action":   action,
		"request": models.ResAttrs{
			"region":      request.Region,
			"nsgId":       request.ResourceId,
			"operation":   rule.Operation,
			"direction":   rule.Direction,
			"protocol":    rule.Protocol,
			"cidr":        rule.Cidr,
			"portRange":   cloudSecurityRulePortText(rule),
			"ruleId":      rule.RuleId,
			"description": rule.Description,
		},
		"response": models.ResAttrs{
			"requestId":     headers.Get("opc-request-id"),
			"workRequestId": headers.Get("opc-work-request-id"),
			"body":          cloudProviderSanitizeAttrs(body),
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("OCI NSG %s 失败：%s", action, err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteAlicloudComputeAction(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) (models.ResAttrs, e.Error) {
	action, ok := cloudProviderAlicloudComputeAction(request.Action)
	if !ok {
		return models.ResAttrs{"executed": false, "provider": "alicloud"}, e.New(e.BadParam, fmt.Errorf("阿里云 ECS 不支持动作 %s", request.Action))
	}
	values := url.Values{
		"Action":     []string{action},
		"Version":    []string{"2014-05-26"},
		"RegionId":   []string{request.Region},
		"InstanceId": []string{request.ResourceId},
	}
	resp := struct {
		RequestId string `json:"RequestId"`
	}{}
	err := alicloudRPCAPI(c, account, request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "alicloud",
		"service":  "ecs",
		"action":   action,
		"request": models.ResAttrs{
			"region":     request.Region,
			"instanceId": request.ResourceId,
		},
		"response": models.ResAttrs{
			"requestId": resp.RequestId,
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("阿里云 ECS %s 失败：%s", action, err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteAzureComputeAction(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) (models.ResAttrs, e.Error) {
	action, ok := cloudProviderAzureComputeAction(request.Action)
	if !ok {
		return models.ResAttrs{"executed": false, "provider": "azure"}, e.New(e.BadParam, fmt.Errorf("Azure VM 不支持动作 %s", request.Action))
	}
	token, err := azureAccessToken(c, account)
	resourcePath := azureResourcePath(request.ResourceId)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "azure",
		"service":  "compute",
		"action":   action,
		"request": models.ResAttrs{
			"resourceId": request.ResourceId,
		},
		"response": models.ResAttrs{},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("Azure VM 获取访问令牌失败：%s", err.Error()))
	}
	if resourcePath == "" {
		err := fmt.Errorf("Azure VM 写操作要求资源 ID 为完整 ARM ID")
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, err)
	}
	rawURL := fmt.Sprintf("%s%s/%s?api-version=%s", azureManagementEndpoint, resourcePath, action, url.QueryEscape(azureComputeAPIVersion))
	req, err := http.NewRequestWithContext(c, http.MethodPost, rawURL, nil)
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("Azure VM %s 请求创建失败：%s", action, err.Error()))
	}
	req.Header.Set("Accept", "application/json")
	err = azureDoJSON(req, token, nil)
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("Azure VM %s 失败：%s", action, err.Error()))
	}
	raw["response"] = models.ResAttrs{"accepted": true}
	return raw, nil
}

func cloudProviderExecuteGcpComputeAction(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) (models.ResAttrs, e.Error) {
	action, ok := cloudProviderGcpComputeAction(request.Action)
	if !ok {
		return models.ResAttrs{"executed": false, "provider": "gcp"}, e.New(e.BadParam, fmt.Errorf("GCP Compute 不支持动作 %s", request.Action))
	}
	token, err := gcpAccessToken(c, account)
	projectId := gcpProjectId(account)
	zone, instanceName := gcpComputeInstanceTarget(request)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "gcp",
		"service":  "compute",
		"action":   action,
		"request": models.ResAttrs{
			"projectId":    projectId,
			"zone":         zone,
			"instanceName": instanceName,
			"resourceId":   request.ResourceId,
		},
		"response": models.ResAttrs{},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("GCP Compute 获取访问令牌失败：%s", err.Error()))
	}
	if projectId == "" || zone == "" || instanceName == "" {
		err := fmt.Errorf("GCP Compute 写操作要求 project、zone 和 instance name")
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, err)
	}
	rawURL := fmt.Sprintf("%s/projects/%s/zones/%s/instances/%s/%s",
		gcpComputeEndpoint, url.PathEscape(projectId), url.PathEscape(zone), url.PathEscape(instanceName), action)
	req, err := http.NewRequestWithContext(c, http.MethodPost, rawURL, nil)
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("GCP Compute %s 请求创建失败：%s", action, err.Error()))
	}
	req.Header.Set("Accept", "application/json")
	err = gcpDoJSON(req, token, nil)
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("GCP Compute %s 失败：%s", action, err.Error()))
	}
	raw["response"] = models.ResAttrs{"accepted": true}
	return raw, nil
}

func cloudProviderExecuteAlicloudInstanceResize(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, targetSpec string) (models.ResAttrs, e.Error) {
	values := url.Values{
		"Action":       []string{"ModifyInstanceSpec"},
		"Version":      []string{"2014-05-26"},
		"RegionId":     []string{request.Region},
		"InstanceId":   []string{request.ResourceId},
		"InstanceType": []string{targetSpec},
	}
	resp := struct {
		RequestId string `json:"RequestId"`
	}{}
	err := alicloudRPCAPI(c, account, request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "alicloud",
		"service":  "ecs",
		"action":   "ModifyInstanceSpec",
		"request": models.ResAttrs{
			"region":             request.Region,
			"instanceId":         request.ResourceId,
			"targetInstanceType": targetSpec,
		},
		"response": models.ResAttrs{
			"requestId": resp.RequestId,
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("阿里云 ECS ModifyInstanceSpec 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteAlicloudVolumeResize(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, targetSizeGiB int) (models.ResAttrs, e.Error) {
	values := url.Values{
		"Action":   []string{"ResizeDisk"},
		"Version":  []string{"2014-05-26"},
		"RegionId": []string{request.Region},
		"DiskId":   []string{request.ResourceId},
		"NewSize":  []string{strconv.Itoa(targetSizeGiB)},
	}
	resp := struct {
		RequestId string `json:"RequestId"`
	}{}
	err := alicloudRPCAPI(c, account, request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "alicloud",
		"service":  "ecs",
		"action":   "ResizeDisk",
		"request": models.ResAttrs{
			"region":        request.Region,
			"diskId":        request.ResourceId,
			"targetSizeGiB": targetSizeGiB,
		},
		"response": models.ResAttrs{
			"requestId": resp.RequestId,
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("阿里云 ECS ResizeDisk 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteAlicloudVolumeSnapshot(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, snapshotName string) (models.ResAttrs, e.Error) {
	description := cloudProviderSnapshotDescription(request)
	values := url.Values{
		"Action":       []string{"CreateSnapshot"},
		"Version":      []string{"2014-05-26"},
		"RegionId":     []string{request.Region},
		"DiskId":       []string{request.ResourceId},
		"SnapshotName": []string{snapshotName},
		"Description":  []string{description},
	}
	resp := struct {
		RequestId  string `json:"RequestId"`
		SnapshotId string `json:"SnapshotId"`
	}{}
	err := alicloudRPCAPI(c, account, request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "alicloud",
		"service":  "ecs",
		"action":   "CreateSnapshot",
		"request": models.ResAttrs{
			"region":       request.Region,
			"diskId":       request.ResourceId,
			"snapshotName": snapshotName,
			"description":  description,
		},
		"response": models.ResAttrs{
			"requestId":  resp.RequestId,
			"snapshotId": resp.SnapshotId,
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("阿里云 ECS CreateSnapshot 失败：%s", err.Error()))
	}
	return raw, nil
}

func cloudProviderExecuteAlicloudSecurityRuleUpdate(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest, rule cloudSecurityRuleOperation) (models.ResAttrs, e.Error) {
	action := "AuthorizeSecurityGroup"
	if rule.Operation == "revoke" && rule.Direction == "ingress" {
		action = "RevokeSecurityGroup"
	} else if rule.Operation == "authorize" && rule.Direction == "egress" {
		action = "AuthorizeSecurityGroupEgress"
	} else if rule.Operation == "revoke" && rule.Direction == "egress" {
		action = "RevokeSecurityGroupEgress"
	}
	values := url.Values{
		"Action":          []string{action},
		"Version":         []string{"2014-05-26"},
		"RegionId":        []string{request.Region},
		"SecurityGroupId": []string{request.ResourceId},
		"IpProtocol":      []string{cloudSecurityRuleAlicloudProtocol(rule.Protocol)},
		"PortRange":       []string{cloudSecurityRuleAlicloudPortRange(rule)},
	}
	if rule.Direction == "egress" {
		values.Set("DestCidrIp", rule.Cidr)
	} else {
		values.Set("SourceCidrIp", rule.Cidr)
	}
	if rule.Description != "" && rule.Operation == "authorize" {
		values.Set("Description", rule.Description)
	}
	resp := struct {
		RequestId string `json:"RequestId"`
	}{}
	err := alicloudRPCAPI(c, account, request.Region, values, &resp)
	raw := models.ResAttrs{
		"executed": true,
		"provider": "alicloud",
		"service":  "ecs",
		"action":   action,
		"request": models.ResAttrs{
			"region":          request.Region,
			"securityGroupId": request.ResourceId,
			"operation":       rule.Operation,
			"direction":       rule.Direction,
			"protocol":        rule.Protocol,
			"cidr":            rule.Cidr,
			"portRange":       cloudSecurityRulePortText(rule),
			"description":     rule.Description,
		},
		"response": models.ResAttrs{
			"requestId": resp.RequestId,
		},
	}
	if err != nil {
		raw["executed"] = false
		raw["error"] = cloudProviderActionErrorAttrs(err)
		return raw, e.New(e.BadParam, fmt.Errorf("阿里云 ECS %s 失败：%s", action, err.Error()))
	}
	return raw, nil
}

func ociInstanceActionAPI(c context.Context, account *cmdbCloudAccount, region, instanceId, action string) (http.Header, error) {
	query := url.Values{"action": []string{action}}
	u := url.URL{
		Scheme:   "https",
		Host:     ociServiceHost("iaas", region),
		Path:     "/" + strings.Trim(ociCoreAPIVersion, "/") + "/instances/" + url.PathEscape(instanceId),
		RawQuery: query.Encode(),
	}
	body := []byte{}
	req, err := http.NewRequestWithContext(c, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setOCIRequestBodyHeaders(req, body)
	if err := signOCIRequest(req, account, time.Now().UTC()); err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if len(strings.TrimSpace(string(respBody))) > 0 {
		providerErr := struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{}
		if err := json.Unmarshal(respBody, &providerErr); err == nil && providerErr.Code != "" {
			return resp.Header, fmt.Errorf("oci %s: %s", providerErr.Code, providerErr.Message)
		}
	}
	return resp.Header, nil
}

func ociUpdateInstanceShapeAPI(c context.Context, account *cmdbCloudAccount, region, instanceId, targetShape string) (http.Header, models.ResAttrs, error) {
	u := url.URL{
		Scheme: "https",
		Host:   ociServiceHost("iaas", region),
		Path:   "/" + strings.Trim(ociCoreAPIVersion, "/") + "/instances/" + url.PathEscape(instanceId),
	}
	body, err := json.Marshal(models.ResAttrs{"shape": targetShape})
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(c, http.MethodPut, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setOCIRequestBodyHeaders(req, body)
	if err := signOCIRequest(req, account, time.Now().UTC()); err != nil {
		return nil, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	item := models.ResAttrs{}
	if len(strings.TrimSpace(string(respBody))) > 0 {
		if err := json.Unmarshal(respBody, &item); err != nil {
			return resp.Header, nil, err
		}
	}
	if providerErr := ociProviderError(item); providerErr != nil {
		return resp.Header, item, providerErr
	}
	return resp.Header, item, nil
}

func ociUpdateVolumeAPI(c context.Context, account *cmdbCloudAccount, region, volumeId string, targetSizeGiB int) (http.Header, models.ResAttrs, error) {
	u := url.URL{
		Scheme: "https",
		Host:   ociServiceHost("iaas", region),
		Path:   "/" + strings.Trim(ociCoreAPIVersion, "/") + "/volumes/" + url.PathEscape(volumeId),
	}
	body, err := json.Marshal(models.ResAttrs{"sizeInGBs": targetSizeGiB})
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(c, http.MethodPut, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setOCIRequestBodyHeaders(req, body)
	if err := signOCIRequest(req, account, time.Now().UTC()); err != nil {
		return nil, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	item := models.ResAttrs{}
	if len(strings.TrimSpace(string(respBody))) > 0 {
		if err := json.Unmarshal(respBody, &item); err != nil {
			return resp.Header, nil, err
		}
	}
	if providerErr := ociProviderError(item); providerErr != nil {
		return resp.Header, item, providerErr
	}
	return resp.Header, item, nil
}

func ociCreateVolumeBackupAPI(c context.Context, account *cmdbCloudAccount, region, volumeId, displayName, backupType string) (http.Header, models.ResAttrs, error) {
	u := url.URL{
		Scheme: "https",
		Host:   ociServiceHost("iaas", region),
		Path:   "/" + strings.Trim(ociCoreAPIVersion, "/") + "/volumeBackups",
	}
	body, err := json.Marshal(models.ResAttrs{
		"volumeId":    volumeId,
		"displayName": displayName,
		"type":        backupType,
	})
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(c, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setOCIRequestBodyHeaders(req, body)
	if err := signOCIRequest(req, account, time.Now().UTC()); err != nil {
		return nil, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	item := models.ResAttrs{}
	if len(strings.TrimSpace(string(respBody))) > 0 {
		if err := json.Unmarshal(respBody, &item); err != nil {
			return resp.Header, nil, err
		}
	}
	if providerErr := ociProviderError(item); providerErr != nil {
		return resp.Header, item, providerErr
	}
	return resp.Header, item, nil
}

func ociNetworkSecurityGroupRuleAPI(c context.Context, account *cmdbCloudAccount, region, nsgId string, rule cloudSecurityRuleOperation) (http.Header, models.ResAttrs, error) {
	actionPath := "addSecurityRules"
	payload := models.ResAttrs{
		"securityRules": []models.ResAttrs{ociNetworkSecurityGroupRulePayload(rule)},
	}
	if rule.Operation == "revoke" {
		if strings.TrimSpace(rule.RuleId) == "" {
			return nil, nil, fmt.Errorf("OCI NSG 撤销规则需要 params.ruleId")
		}
		actionPath = "removeSecurityRules"
		payload = models.ResAttrs{"securityRuleIds": []string{rule.RuleId}}
	}
	u := url.URL{
		Scheme: "https",
		Host:   ociServiceHost("iaas", region),
		Path: "/" + strings.Trim(ociCoreAPIVersion, "/") + "/networkSecurityGroups/" +
			url.PathEscape(nsgId) + "/actions/" + actionPath,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(c, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setOCIRequestBodyHeaders(req, body)
	if err := signOCIRequest(req, account, time.Now().UTC()); err != nil {
		return nil, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	item := models.ResAttrs{}
	if len(strings.TrimSpace(string(respBody))) > 0 {
		if err := json.Unmarshal(respBody, &item); err != nil {
			return resp.Header, nil, err
		}
	}
	if providerErr := ociProviderError(item); providerErr != nil {
		return resp.Header, item, providerErr
	}
	return resp.Header, item, nil
}

func ociNetworkSecurityGroupRulePayload(rule cloudSecurityRuleOperation) models.ResAttrs {
	payload := models.ResAttrs{
		"direction":   strings.ToUpper(rule.Direction),
		"protocol":    cloudSecurityRuleOciProtocol(rule.Protocol),
		"description": rule.Description,
	}
	if rule.Direction == "egress" {
		payload["destination"] = rule.Cidr
		payload["destinationType"] = "CIDR_BLOCK"
	} else {
		payload["source"] = rule.Cidr
		payload["sourceType"] = "CIDR_BLOCK"
	}
	if rule.FromPort >= 0 && rule.ToPort >= 0 {
		switch strings.ToLower(rule.Protocol) {
		case "udp", "17":
			payload["udpOptions"] = models.ResAttrs{
				"destinationPortRange": models.ResAttrs{"min": rule.FromPort, "max": rule.ToPort},
			}
		default:
			payload["tcpOptions"] = models.ResAttrs{
				"destinationPortRange": models.ResAttrs{"min": rule.FromPort, "max": rule.ToPort},
			}
		}
	}
	return payload
}

func ociProviderError(item models.ResAttrs) error {
	if len(item) == 0 {
		return nil
	}
	code := ociAttrString(item, "code")
	if code == "" {
		return nil
	}
	return fmt.Errorf("oci %s: %s", code, ociAttrString(item, "message"))
}

func cloudProviderAwsComputeAction(action string) (string, bool) {
	switch action {
	case models.CloudOperationActionStartInstance:
		return "StartInstances", true
	case models.CloudOperationActionStopInstance:
		return "StopInstances", true
	case models.CloudOperationActionRestartInstance:
		return "RebootInstances", true
	default:
		return "", false
	}
}

func cloudProviderOciComputeAction(action string) (string, bool) {
	switch action {
	case models.CloudOperationActionStartInstance:
		return "START", true
	case models.CloudOperationActionStopInstance:
		return "STOP", true
	case models.CloudOperationActionRestartInstance:
		return "RESET", true
	default:
		return "", false
	}
}

func cloudProviderAlicloudComputeAction(action string) (string, bool) {
	switch action {
	case models.CloudOperationActionStartInstance:
		return "StartInstance", true
	case models.CloudOperationActionStopInstance:
		return "StopInstance", true
	case models.CloudOperationActionRestartInstance:
		return "RebootInstance", true
	default:
		return "", false
	}
}

func cloudProviderAzureComputeAction(action string) (string, bool) {
	switch action {
	case models.CloudOperationActionStartInstance:
		return "start", true
	case models.CloudOperationActionStopInstance:
		return "powerOff", true
	case models.CloudOperationActionRestartInstance:
		return "restart", true
	default:
		return "", false
	}
}

func cloudProviderGcpComputeAction(action string) (string, bool) {
	switch action {
	case models.CloudOperationActionStartInstance:
		return "start", true
	case models.CloudOperationActionStopInstance:
		return "stop", true
	case models.CloudOperationActionRestartInstance:
		return "reset", true
	default:
		return "", false
	}
}

func cloudProviderComputeActionNoop(request cloudProviderOperationRequest, state cloudProviderResourceState) bool {
	switch request.Action {
	case models.CloudOperationActionStartInstance:
		return state.NormalizedState == "running"
	case models.CloudOperationActionStopInstance:
		return state.NormalizedState == "stopped"
	default:
		return false
	}
}

func cloudProviderVolumeTargetSizeGiB(request cloudProviderOperationRequest) (int, bool) {
	targetSize, ok := cloudOperationIntParam(request.Params, "targetSizeGiB")
	if ok && targetSize > 0 {
		return targetSize, true
	}
	return 0, false
}

func cloudSecurityRuleAwsProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "all", "-1":
		return "-1"
	case "tcp", "6":
		return "tcp"
	case "udp", "17":
		return "udp"
	case "icmp", "1":
		return "icmp"
	default:
		return strings.ToLower(strings.TrimSpace(protocol))
	}
}

func cloudSecurityRuleAlicloudProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "all", "-1":
		return "all"
	case "tcp", "6":
		return "tcp"
	case "udp", "17":
		return "udp"
	case "icmp", "1":
		return "icmp"
	default:
		return strings.ToLower(strings.TrimSpace(protocol))
	}
}

func cloudSecurityRuleOciProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "all", "-1":
		return "all"
	case "tcp", "6":
		return "6"
	case "udp", "17":
		return "17"
	case "icmp", "1":
		return "1"
	default:
		return strings.ToLower(strings.TrimSpace(protocol))
	}
}

func cloudSecurityRuleAlicloudPortRange(rule cloudSecurityRuleOperation) string {
	if rule.FromPort < 0 || rule.ToPort < 0 {
		return "-1/-1"
	}
	return fmt.Sprintf("%d/%d", rule.FromPort, rule.ToPort)
}

func cloudProviderResourceStateAttrs(state cloudProviderResourceState) models.ResAttrs {
	return models.ResAttrs{
		"rawState":        state.RawState,
		"normalizedState": state.NormalizedState,
		"source":          state.Source,
		"readMode":        state.ReadMode,
		"rawResponse":     cloudProviderSanitizeAttrs(state.RawResponse),
		"error": models.ResAttrs{
			"code":      state.ErrorCode,
			"message":   state.ErrorMessage,
			"retryable": state.Retryable,
		},
	}
}

func cloudProviderActionErrorAttrs(err error) models.ResAttrs {
	if err == nil {
		return nil
	}
	return models.ResAttrs{
		"message":   err.Error(),
		"retryable": cloudProviderErrorRetryable(err),
	}
}

func cloudOperationProviderWriteTimeout() time.Duration {
	seconds := 30
	if value := strings.TrimSpace(os.Getenv(cloudOperationProviderWriteTimeoutEnv)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			seconds = parsed
		}
	}
	return time.Duration(seconds) * time.Second
}

func cloudOperationProviderPollAttempts() int {
	attempts := 3
	if value := strings.TrimSpace(os.Getenv(cloudOperationProviderPollAttemptsEnv)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			attempts = parsed
		}
	}
	return attempts
}

func cloudOperationProviderPollInterval() time.Duration {
	seconds := 2
	if value := strings.TrimSpace(os.Getenv(cloudOperationProviderPollIntervalEnv)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			seconds = parsed
		}
	}
	return time.Duration(seconds) * time.Second
}

func cloudOperationProviderPollMaxDelay() time.Duration {
	seconds := 10
	if value := strings.TrimSpace(os.Getenv(cloudOperationProviderPollMaxDelayEnv)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			seconds = parsed
		}
	}
	return time.Duration(seconds) * time.Second
}

func cloudProviderNextPollDelay(current time.Duration, maxDelay time.Duration) time.Duration {
	next := current * 2
	if next > maxDelay {
		return maxDelay
	}
	return next
}

func cloudProviderPollErrorAttrs(err error) models.ResAttrs {
	if err == nil {
		return nil
	}
	return models.ResAttrs{
		"message":   err.Error(),
		"retryable": cloudProviderErrorRetryable(err),
	}
}

func modelResAttrs(value interface{}) models.ResAttrs {
	switch typed := value.(type) {
	case models.ResAttrs:
		return typed
	case map[string]interface{}:
		return models.ResAttrs(typed)
	default:
		return nil
	}
}
