// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"strings"

	"cloudiac/portal/libs/db"
)

type cmdbAssetDSLTerm struct {
	Key   string
	Value string
}

func applyCmdbAssetDSL(query *db.Session, dsl string) *db.Session {
	for _, term := range parseCmdbAssetDSL(dsl) {
		query = applyCmdbAssetDSLTerm(query, term)
	}
	return query
}

func parseCmdbAssetDSL(dsl string) []cmdbAssetDSLTerm {
	terms := make([]cmdbAssetDSLTerm, 0)
	for _, token := range splitCmdbAssetDSLTokens(dsl) {
		key, value, ok := strings.Cut(token, ":")
		if !ok {
			key, value, ok = strings.Cut(token, "=")
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = trimCmdbAssetDSLValue(value)
		if !ok || key == "" || value == "" {
			continue
		}
		terms = append(terms, cmdbAssetDSLTerm{Key: key, Value: value})
	}
	return terms
}

func splitCmdbAssetDSLTokens(dsl string) []string {
	tokens := make([]string, 0)
	var builder strings.Builder
	inQuote := rune(0)
	escaped := false
	for _, r := range dsl {
		switch {
		case escaped:
			builder.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case inQuote != 0:
			if r == inQuote {
				inQuote = 0
			} else {
				builder.WriteRune(r)
			}
		case r == '"' || r == '\'':
			inQuote = r
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if builder.Len() > 0 {
				tokens = append(tokens, builder.String())
				builder.Reset()
			}
		default:
			builder.WriteRune(r)
		}
	}
	if builder.Len() > 0 {
		tokens = append(tokens, builder.String())
	}
	return tokens
}

func trimCmdbAssetDSLValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return strings.TrimSpace(value)
}

func applyCmdbAssetDSLTerm(query *db.Session, term cmdbAssetDSLTerm) *db.Session {
	if strings.HasPrefix(term.Key, "tag.") {
		return applyCmdbAssetDSLJSONTerm(query, "iac_cmdb_asset.tags", strings.TrimPrefix(term.Key, "tag."), term.Value)
	}
	if strings.HasPrefix(term.Key, "attr.") {
		return applyCmdbAssetDSLJSONTerm(query, "iac_cmdb_asset.attributes", strings.TrimPrefix(term.Key, "attr."), term.Value)
	}

	column, matchMode, ok := cmdbAssetDSLColumn(term.Key)
	if !ok {
		return query
	}

	values := splitCmdbAssetDSLValues(term.Value)
	if len(values) == 0 {
		return query
	}
	if matchMode == "exact" {
		if len(values) == 1 {
			return query.Where(fmt.Sprintf("%s = ?", column), values[0])
		}
		return query.Where(fmt.Sprintf("%s in (?)", column), values)
	}

	likes := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values))
	for _, value := range values {
		likes = append(likes, fmt.Sprintf("%s like ?", column))
		args = append(args, fmt.Sprintf("%%%s%%", value))
	}
	return query.Where("("+strings.Join(likes, " or ")+")", args...)
}

func cmdbAssetDSLColumn(key string) (string, string, bool) {
	exactColumns := map[string]string{
		"provider":       "iac_cmdb_asset.provider",
		"account":        "iac_cmdb_asset.account_id",
		"accountid":      "iac_cmdb_asset.account_id",
		"region":         "iac_cmdb_asset.region",
		"zone":           "iac_cmdb_asset.zone",
		"type":           "iac_cmdb_asset.asset_type",
		"assettype":      "iac_cmdb_asset.asset_type",
		"source":         "iac_cmdb_asset.source",
		"status":         "iac_cmdb_asset.status",
		"project":        "iac_cmdb_asset.project_id",
		"projectid":      "iac_cmdb_asset.project_id",
		"env":            "iac_cmdb_asset.env_id",
		"envid":          "iac_cmdb_asset.env_id",
		"lifecycle":      "iac_cmdb_asset.lifecycle",
		"cost":           "iac_cmdb_asset.cost",
		"risk":           "iac_cmdb_asset.compliance_risk",
		"compliancerisk": "iac_cmdb_asset.compliance_risk",
	}
	if column, ok := exactColumns[key]; ok {
		return column, "exact", true
	}

	likeColumns := map[string]string{
		"id":           "iac_cmdb_asset.id",
		"name":         "iac_cmdb_asset.name",
		"nativeid":     "iac_cmdb_asset.native_id",
		"resourceid":   "iac_cmdb_asset.native_id",
		"nativetype":   "iac_cmdb_asset.native_type",
		"address":      "iac_cmdb_asset.address",
		"module":       "iac_cmdb_asset.module",
		"owner":        "iac_cmdb_asset.owner",
		"application":  "iac_cmdb_asset.application",
		"app":          "iac_cmdb_asset.application",
		"businessline": "iac_cmdb_asset.business_line",
		"biz":          "iac_cmdb_asset.business_line",
		"publicip":     "iac_cmdb_asset.public_ip",
		"privateip":    "iac_cmdb_asset.private_ip",
		"iacaddress":   "iac_cmdb_asset.iac_address",
	}
	if column, ok := likeColumns[key]; ok {
		return column, "like", true
	}
	return "", "", false
}

func splitCmdbAssetDSLValues(value string) []string {
	values := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			values = append(values, item)
		}
	}
	return values
}

func applyCmdbAssetDSLJSONTerm(query *db.Session, column, key, value string) *db.Session {
	if !validCmdbAssetDSLJSONKey(key) {
		return query
	}
	jsonPath := cmdbAssetDSLJSONPath(key)
	return query.Where(fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(%s, ?)) like ?", column), jsonPath, fmt.Sprintf("%%%s%%", value))
}

func validCmdbAssetDSLJSONKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == '/' {
			continue
		}
		return false
	}
	return true
}

func cmdbAssetDSLJSONPath(key string) string {
	parts := strings.Split(key, ".")
	var builder strings.Builder
	builder.WriteString("$")
	for _, part := range parts {
		part = strings.ReplaceAll(part, `\`, `\\`)
		part = strings.ReplaceAll(part, `"`, `\"`)
		builder.WriteString(`."`)
		builder.WriteString(part)
		builder.WriteString(`"`)
	}
	return builder.String()
}
