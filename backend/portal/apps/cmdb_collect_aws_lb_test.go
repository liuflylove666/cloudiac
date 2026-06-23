package apps

import (
	"encoding/xml"
	"testing"

	"cloudiac/portal/models"
)

func TestAwsV2ListenerAttrsExtractsTargetsAndCertificates(t *testing.T) {
	listener := awsV2Listener{
		ListenerArn:     "arn:aws:elasticloadbalancing:ap-southeast-1:123:listener/app/demo/1/2",
		LoadBalancerArn: "arn:aws:elasticloadbalancing:ap-southeast-1:123:loadbalancer/app/demo/1",
		Port:            "443",
		Protocol:        "HTTPS",
		SslPolicy:       "ELBSecurityPolicy-TLS13-1-2-2021-06",
	}
	listener.Certificates = append(listener.Certificates, struct {
		CertificateArn string `xml:"CertificateArn"`
		IsDefault      string `xml:"IsDefault"`
	}{CertificateArn: "arn:aws:acm:cert/default", IsDefault: "true"})
	listener.DefaultActions = []awsV2ListenerAction{
		{
			Type:           "forward",
			TargetGroupArn: "arn:aws:elasticloadbalancing:targetgroup/app-a/1",
		},
	}
	listener.DefaultActions[0].ForwardConfig.TargetGroups = append(listener.DefaultActions[0].ForwardConfig.TargetGroups, struct {
		TargetGroupArn string `xml:"TargetGroupArn"`
		Weight         string `xml:"Weight"`
	}{TargetGroupArn: "arn:aws:elasticloadbalancing:targetgroup/app-b/2", Weight: "20"})

	attrs := awsV2ListenerAttrs([]awsV2Listener{listener})
	if len(attrs) != 1 {
		t.Fatalf("expected one listener attr, got %#v", attrs)
	}
	if got := attrString(attrs[0], "protocol"); got != "HTTPS" {
		t.Fatalf("expected HTTPS protocol, got %q", got)
	}
	targetArns := awsV2ListenerTargetGroupArns([]awsV2Listener{listener})
	if len(targetArns) != 2 || targetArns[0] != "arn:aws:elasticloadbalancing:targetgroup/app-a/1" {
		t.Fatalf("unexpected target group arns: %#v", targetArns)
	}
	certs, ok := attrs[0]["certificates"].([]models.ResAttrs)
	if !ok || len(certs) != 1 || certs[0]["default"] != true {
		t.Fatalf("unexpected certificates: %#v", attrs[0]["certificates"])
	}
}

func TestAwsV2TargetGroupAttrsExtractsTargetRefs(t *testing.T) {
	group := awsV2TargetGroup{
		TargetGroupArn:      "arn:aws:elasticloadbalancing:targetgroup/app/1",
		TargetGroupName:     "app",
		Protocol:            "HTTP",
		Port:                "80",
		VpcId:               "vpc-123",
		TargetType:          "instance",
		HealthCheckEnabled:  "true",
		HealthCheckProtocol: "HTTP",
	}
	targets := []awsV2TargetHealthDescription{
		awsV2TargetHealth("i-123", "80", "healthy"),
		awsV2TargetHealth("10.0.1.10", "8080", "unhealthy"),
		awsV2TargetHealth("i-123", "80", "healthy"),
	}

	attrs := awsV2TargetGroupAttrs(group, targets)
	instanceIds, ok := attrs["targetInstanceIds"].([]string)
	if !ok || len(instanceIds) != 1 || instanceIds[0] != "i-123" {
		t.Fatalf("unexpected instance ids: %#v", attrs["targetInstanceIds"])
	}
	ipTargets, ok := attrs["targetIpAddresses"].([]string)
	if !ok || len(ipTargets) != 1 || ipTargets[0] != "10.0.1.10" {
		t.Fatalf("unexpected ip targets: %#v", attrs["targetIpAddresses"])
	}
	if attrs["healthyTargetCount"] != 2 || attrs["unhealthyTargetCount"] != 1 {
		t.Fatalf("unexpected target counts: %#v", attrs)
	}
	targetRows, ok := attrs["targets"].([]models.ResAttrs)
	if !ok || len(targetRows) != 3 {
		t.Fatalf("unexpected target rows: %#v", attrs["targets"])
	}
}

func TestAwsV2RuleAttrsExtractsConditionsAndTargets(t *testing.T) {
	raw := []byte(`
<DescribeRulesResponse>
  <DescribeRulesResult>
    <Rules>
      <member>
        <RuleArn>arn:aws:elasticloadbalancing:listener-rule/app/demo/1/2/3</RuleArn>
        <Priority>10</Priority>
        <IsDefault>false</IsDefault>
        <Conditions>
          <member>
            <Field>host-header</Field>
            <HostHeaderConfig>
              <Values>
                <member>api.example.com</member>
              </Values>
            </HostHeaderConfig>
          </member>
          <member>
            <Field>path-pattern</Field>
            <PathPatternConfig>
              <Values>
                <member>/v1/*</member>
              </Values>
            </PathPatternConfig>
          </member>
          <member>
            <Field>query-string</Field>
            <QueryStringConfig>
              <Values>
                <member>
                  <Key>version</Key>
                  <Value>blue</Value>
                </member>
              </Values>
            </QueryStringConfig>
          </member>
        </Conditions>
        <Actions>
          <member>
            <Type>forward</Type>
            <TargetGroupArn>arn:aws:elasticloadbalancing:targetgroup/app-a/1</TargetGroupArn>
            <ForwardConfig>
              <TargetGroups>
                <member>
                  <TargetGroupArn>arn:aws:elasticloadbalancing:targetgroup/app-b/2</TargetGroupArn>
                  <Weight>10</Weight>
                </member>
              </TargetGroups>
            </ForwardConfig>
          </member>
        </Actions>
      </member>
    </Rules>
  </DescribeRulesResult>
</DescribeRulesResponse>`)
	resp := awsELBV2DescribeRulesResponse{}
	if err := xml.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal rules response: %v", err)
	}
	rules := resp.Result.Rules
	if len(rules) != 1 {
		t.Fatalf("expected one rule, got %#v", rules)
	}

	attrs := awsV2RuleAttrs(rules)
	if len(attrs) != 1 || attrs[0]["default"] != false {
		t.Fatalf("unexpected rule attrs: %#v", attrs)
	}
	targetArns := awsV2RuleTargetGroupArns(rules)
	if len(targetArns) != 2 || targetArns[0] != "arn:aws:elasticloadbalancing:targetgroup/app-a/1" {
		t.Fatalf("unexpected rule target groups: %#v", targetArns)
	}
	conditions, ok := attrs[0]["conditions"].([]models.ResAttrs)
	if !ok || len(conditions) != 3 {
		t.Fatalf("unexpected conditions: %#v", attrs[0]["conditions"])
	}
	hostValues, _ := conditions[0]["hostHeaderValues"].([]string)
	if len(hostValues) != 1 || hostValues[0] != "api.example.com" {
		t.Fatalf("unexpected host condition: %#v", conditions[0])
	}
	queryValues, ok := conditions[2]["queryStringValues"].([]models.ResAttrs)
	if !ok || len(queryValues) != 1 || queryValues[0]["key"] != "version" || queryValues[0]["value"] != "blue" {
		t.Fatalf("unexpected query condition: %#v", conditions[2])
	}

	listener := awsV2Listener{ListenerArn: "listener-1"}
	listenerAttrs := awsV2ListenerAttrs([]awsV2Listener{listener}, map[string][]awsV2Rule{"listener-1": rules})
	if len(listenerAttrs) != 1 || listenerAttrs[0]["ruleCount"] != 1 {
		t.Fatalf("unexpected listener rule attrs: %#v", listenerAttrs)
	}
}

func TestAwsV2ListenerActionsExtractsAuthenticationConfigs(t *testing.T) {
	raw := []byte(`
<DescribeRulesResponse>
  <DescribeRulesResult>
    <Rules>
      <member>
        <RuleArn>arn:aws:elasticloadbalancing:listener-rule/app/demo/auth</RuleArn>
        <Priority>20</Priority>
        <Actions>
          <member>
            <Type>authenticate-oidc</Type>
            <Order>1</Order>
            <AuthenticateOidcConfig>
              <Issuer>https://idp.example.com</Issuer>
              <AuthorizationEndpoint>https://idp.example.com/oauth2/authorize</AuthorizationEndpoint>
              <TokenEndpoint>https://idp.example.com/oauth2/token</TokenEndpoint>
              <UserInfoEndpoint>https://idp.example.com/oauth2/userinfo</UserInfoEndpoint>
              <ClientId>client-1</ClientId>
              <ClientSecret>super-secret</ClientSecret>
              <SessionCookieName>AuthSession</SessionCookieName>
              <Scope>openid profile email</Scope>
              <SessionTimeout>3600</SessionTimeout>
              <AuthenticationRequestExtraParams>
                <entry>
                  <key>prompt</key>
                  <value>login</value>
                </entry>
              </AuthenticationRequestExtraParams>
              <OnUnauthenticatedRequest>authenticate</OnUnauthenticatedRequest>
              <UseExistingClientSecret>false</UseExistingClientSecret>
            </AuthenticateOidcConfig>
          </member>
          <member>
            <Type>authenticate-cognito</Type>
            <Order>2</Order>
            <AuthenticateCognitoConfig>
              <UserPoolArn>arn:aws:cognito-idp:ap-southeast-1:123:userpool/ap-southeast-1_demo</UserPoolArn>
              <UserPoolClientId>cognito-client</UserPoolClientId>
              <UserPoolDomain>auth.example.com</UserPoolDomain>
              <SessionCookieName>CognitoSession</SessionCookieName>
              <Scope>openid</Scope>
              <SessionTimeout>604800</SessionTimeout>
              <AuthenticationRequestExtraParams>
                <entry>
                  <key>lang</key>
                  <value>zh</value>
                </entry>
              </AuthenticationRequestExtraParams>
              <OnUnauthenticatedRequest>deny</OnUnauthenticatedRequest>
            </AuthenticateCognitoConfig>
          </member>
          <member>
            <Type>jwt-validation</Type>
            <Order>3</Order>
            <JwtValidationConfig>
              <Issuer>https://jwt.example.com</Issuer>
              <JwksEndpoint>https://jwt.example.com/.well-known/jwks.json</JwksEndpoint>
              <AdditionalClaims>
                <member>
                  <Name>aud</Name>
                  <Format>single-string</Format>
                  <Values>
                    <member>api</member>
                  </Values>
                </member>
              </AdditionalClaims>
            </JwtValidationConfig>
          </member>
        </Actions>
      </member>
    </Rules>
  </DescribeRulesResult>
</DescribeRulesResponse>`)
	resp := awsELBV2DescribeRulesResponse{}
	if err := xml.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal auth rules response: %v", err)
	}
	if len(resp.Result.Rules) != 1 || len(resp.Result.Rules[0].Actions) != 3 {
		t.Fatalf("unexpected auth actions: %#v", resp.Result.Rules)
	}

	actions := awsV2ListenerActions(resp.Result.Rules[0].Actions)
	if len(actions) != 3 {
		t.Fatalf("expected three action attrs, got %#v", actions)
	}
	oidc, ok := actions[0]["authenticateOidc"].(models.ResAttrs)
	if !ok {
		t.Fatalf("expected oidc attrs, got %#v", actions[0])
	}
	if oidc["clientSecret"] != cmdbAssetMaskedValue || oidc["clientSecretConfigured"] != true {
		t.Fatalf("expected masked oidc client secret, got %#v", oidc)
	}
	extraParams, ok := oidc["authenticationRequestExtraParams"].([]models.ResAttrs)
	if !ok || len(extraParams) != 1 || extraParams[0]["key"] != "prompt" || extraParams[0]["value"] != "login" {
		t.Fatalf("unexpected oidc extra params: %#v", oidc["authenticationRequestExtraParams"])
	}

	cognito, ok := actions[1]["authenticateCognito"].(models.ResAttrs)
	if !ok || cognito["userPoolClientId"] != "cognito-client" || cognito["onUnauthenticatedRequest"] != "deny" {
		t.Fatalf("unexpected cognito attrs: %#v", actions[1])
	}

	jwt, ok := actions[2]["jwtValidation"].(models.ResAttrs)
	if !ok || jwt["jwksEndpoint"] != "https://jwt.example.com/.well-known/jwks.json" {
		t.Fatalf("unexpected jwt attrs: %#v", actions[2])
	}
	claims, ok := jwt["additionalClaims"].([]models.ResAttrs)
	if !ok || len(claims) != 1 || claims[0]["name"] != "aud" {
		t.Fatalf("unexpected jwt claims: %#v", jwt["additionalClaims"])
	}
}

func TestInferCloudRelationsAddsLoadBalancerTargetsAndSecurity(t *testing.T) {
	lb := models.CmdbAsset{
		TimedModel: models.TimedModel{BaseModel: models.BaseModel{Id: "lb-1"}},
		AssetType:  models.CmdbAssetTypeLoadBalancer,
		NativeId:   "arn:aws:elasticloadbalancing:loadbalancer/app/demo/1",
		Attributes: models.ResAttrs{"securityGroupIds": []string{"sg-1"}, "targetInstanceIds": []string{"i-123"}},
	}
	sg := models.CmdbAsset{
		TimedModel: models.TimedModel{BaseModel: models.BaseModel{Id: "sg-1"}},
		AssetType:  models.CmdbAssetTypeNetworkSecurityGroup,
		NativeId:   "sg-1",
	}
	instance := models.CmdbAsset{
		TimedModel: models.TimedModel{BaseModel: models.BaseModel{Id: "ci-1"}},
		AssetType:  models.CmdbAssetTypeComputeInstance,
		NativeId:   "i-123",
	}
	assets := []models.CmdbAsset{lb, sg, instance}
	relations := inferCmdbCloudRelations("org-test", "aws", "123", assets, newCmdbCloudRelationIndex(assets))

	if !hasInferredRelation(relations, "lb-1", "sg-1", "lb_security") {
		t.Fatalf("expected lb security relation, got %#v", relations)
	}
	if !hasInferredRelation(relations, "lb-1", "ci-1", "lb_targets") {
		t.Fatalf("expected lb target relation, got %#v", relations)
	}
}

func awsV2TargetHealth(id, port, state string) awsV2TargetHealthDescription {
	target := awsV2TargetHealthDescription{}
	target.Target.Id = id
	target.Target.Port = port
	target.TargetHealth.State = state
	return target
}

func hasInferredRelation(relations []models.CmdbAssetRelation, sourceId, targetId, inferredBy string) bool {
	for _, relation := range relations {
		if relation.SourceAssetId == models.Id(sourceId) &&
			relation.TargetAssetId == models.Id(targetId) &&
			attrString(relation.Metadata, "inferredBy") == inferredBy {
			return true
		}
	}
	return false
}
