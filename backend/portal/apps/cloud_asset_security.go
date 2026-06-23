// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"strings"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
)

func CloudAssetSecurityRules(c *ctx.ServiceContext, form *forms.CmdbAssetParam) (*resps.CloudAssetSecurityRulesResp, e.Error) {
	asset := models.CmdbAsset{}
	if err := buildCmdbAssetQuery(c).
		Where("iac_cmdb_asset.id = ?", form.Id).
		First(&asset); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}

	rules := cloudAssetSecurityRulesFromAsset(&asset)
	publicCount := 0
	for _, rule := range rules {
		if rule.PublicExposure {
			publicCount++
		}
	}
	return &resps.CloudAssetSecurityRulesResp{
		AssetId:         asset.Id,
		Provider:        asset.Provider,
		AssetType:       asset.AssetType,
		NativeType:      asset.NativeType,
		RuleCount:       len(rules),
		PublicRuleCount: publicCount,
		Rules:           rules,
	}, nil
}

func cloudAssetSecurityRulesFromAsset(asset *models.CmdbAsset) []resps.CloudAssetSecurityRuleResp {
	if asset == nil {
		return nil
	}
	rules := make([]resps.CloudAssetSecurityRuleResp, 0)
	for _, raw := range cloudSecurityRuleList(asset.Attributes, "ingressSecurityRules", "ingress_security_rules", "ingressRules", "ingress", "ipPermissions") {
		rules = append(rules, cloudSecurityRuleResp("ingress", raw))
	}
	for _, raw := range cloudSecurityRuleList(asset.Attributes, "egressSecurityRules", "egress_security_rules", "egressRules", "egress", "ipPermissionsEgress") {
		rules = append(rules, cloudSecurityRuleResp("egress", raw))
	}
	return rules
}

func cloudSecurityRuleList(attrs models.ResAttrs, keys ...string) []models.ResAttrs {
	for _, key := range keys {
		if list := cloudSecurityRuleListValue(attrs[key]); len(list) > 0 {
			return list
		}
	}
	return nil
}

func cloudSecurityRuleListValue(value interface{}) []models.ResAttrs {
	switch typed := value.(type) {
	case []models.ResAttrs:
		return typed
	case []map[string]interface{}:
		result := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			result = append(result, models.ResAttrs(item))
		}
		return result
	case []interface{}:
		result := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			if attrs := modelResAttrs(item); attrs != nil {
				result = append(result, attrs)
			}
		}
		return result
	case models.ResAttrs:
		return []models.ResAttrs{typed}
	case map[string]interface{}:
		return []models.ResAttrs{models.ResAttrs(typed)}
	default:
		return nil
	}
}

func cloudSecurityRuleResp(direction string, raw models.ResAttrs) resps.CloudAssetSecurityRuleResp {
	source := firstNonEmpty(
		attrString(raw, "source"),
		attrString(raw, "sourceCidr"),
		attrString(raw, "sourceCidrBlock"),
		attrString(raw, "cidrBlock"),
		attrString(raw, "ipv6CidrBlock"),
	)
	destination := firstNonEmpty(
		attrString(raw, "destination"),
		attrString(raw, "destinationCidr"),
		attrString(raw, "destinationCidrBlock"),
	)
	return resps.CloudAssetSecurityRuleResp{
		Direction:      direction,
		Protocol:       firstNonEmpty(attrString(raw, "protocol"), attrString(raw, "ipProtocol")),
		Source:         source,
		Destination:    destination,
		PortRange:      cloudSecurityRulePortRange(raw),
		Description:    cloudSecurityRuleDescription(raw),
		PublicExposure: cloudSecurityRulePublicExposure(source, destination),
		Raw:            sanitizeCmdbAssetAttrs(raw),
	}
}

func cloudSecurityRulePortRange(raw models.ResAttrs) string {
	if port := firstNonEmpty(attrString(raw, "portRange"), attrString(raw, "port")); port != "" {
		return port
	}
	if fromPort := attrString(raw, "fromPort"); fromPort != "" {
		toPort := firstNonEmpty(attrString(raw, "toPort"), fromPort)
		if fromPort == toPort {
			return fromPort
		}
		return fmt.Sprintf("%s-%s", fromPort, toPort)
	}
	for _, optionKey := range []string{"tcpOptions", "udpOptions"} {
		if portRange := cloudSecurityNestedPortRange(modelResAttrs(raw[optionKey])); portRange != "" {
			return portRange
		}
	}
	return "all"
}

func cloudSecurityNestedPortRange(options models.ResAttrs) string {
	if len(options) == 0 {
		return ""
	}
	for _, key := range []string{"destinationPortRange", "sourcePortRange", "portRange"} {
		if portRange := modelResAttrs(options[key]); len(portRange) > 0 {
			minPort := firstNonEmpty(attrString(portRange, "min"), attrString(portRange, "from"))
			maxPort := firstNonEmpty(attrString(portRange, "max"), attrString(portRange, "to"), minPort)
			if minPort == "" {
				continue
			}
			if minPort == maxPort {
				return minPort
			}
			return fmt.Sprintf("%s-%s", minPort, maxPort)
		}
	}
	return ""
}

func cloudSecurityRuleDescription(raw models.ResAttrs) string {
	description := firstNonEmpty(attrString(raw, "description"), attrString(raw, "ruleDescription"))
	if stateless := attrString(raw, "isStateless"); stateless != "" {
		if description != "" {
			return fmt.Sprintf("%s；stateless=%s", description, stateless)
		}
		return fmt.Sprintf("stateless=%s", stateless)
	}
	return description
}

func cloudSecurityRulePublicExposure(values ...string) bool {
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(normalized, "0.0.0.0/0") || strings.Contains(normalized, "::/0") {
			return true
		}
	}
	return false
}
