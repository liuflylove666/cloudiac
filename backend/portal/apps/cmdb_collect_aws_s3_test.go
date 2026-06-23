package apps

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"testing"

	"cloudiac/portal/models"
)

func TestAwsS3BucketConfigAttrsExtractsGovernanceFields(t *testing.T) {
	tagging := awsS3BucketTaggingResponse{}
	if err := xml.Unmarshal([]byte(`
<Tagging>
  <TagSet>
    <Tag>
      <Key>application</Key>
      <Value>billing</Value>
    </Tag>
    <Tag>
      <Key>cost-center</Key>
      <Value>finops</Value>
    </Tag>
  </TagSet>
</Tagging>`), &tagging); err != nil {
		t.Fatalf("unmarshal tagging: %v", err)
	}
	tagAttrs := awsS3BucketTagsToAttrs(tagging.Tags)
	if tagAttrs["application"] != "billing" || tagAttrs["cost-center"] != "finops" {
		t.Fatalf("unexpected tag attrs: %#v", tagAttrs)
	}

	encryption := awsS3BucketEncryptionResponse{}
	if err := xml.Unmarshal([]byte(`
<ServerSideEncryptionConfiguration>
  <Rule>
    <ApplyServerSideEncryptionByDefault>
      <SSEAlgorithm>aws:kms</SSEAlgorithm>
      <KMSMasterKeyID>arn:aws:kms:ap-southeast-1:123:key/demo</KMSMasterKeyID>
    </ApplyServerSideEncryptionByDefault>
    <BucketKeyEnabled>true</BucketKeyEnabled>
  </Rule>
</ServerSideEncryptionConfiguration>`), &encryption); err != nil {
		t.Fatalf("unmarshal encryption: %v", err)
	}
	encryptionAttrs := awsS3BucketEncryptionAttrs(encryption)
	if len(encryptionAttrs) != 1 || encryptionAttrs[0]["sseAlgorithm"] != "aws:kms" || encryptionAttrs[0]["bucketKeyEnabled"] != true {
		t.Fatalf("unexpected encryption attrs: %#v", encryptionAttrs)
	}

	versioning := awsS3BucketVersioningResponse{}
	if err := xml.Unmarshal([]byte(`
<VersioningConfiguration>
  <Status>Enabled</Status>
  <MfaDelete>Disabled</MfaDelete>
</VersioningConfiguration>`), &versioning); err != nil {
		t.Fatalf("unmarshal versioning: %v", err)
	}
	versioningAttrs := awsS3BucketVersioningAttrs(versioning)
	if versioningAttrs["enabled"] != true || versioningAttrs["mfaDelete"] != "Disabled" {
		t.Fatalf("unexpected versioning attrs: %#v", versioningAttrs)
	}

	publicAccess := awsS3PublicAccessBlockResponse{}
	if err := xml.Unmarshal([]byte(`
<PublicAccessBlockConfiguration>
  <BlockPublicAcls>true</BlockPublicAcls>
  <IgnorePublicAcls>true</IgnorePublicAcls>
  <BlockPublicPolicy>true</BlockPublicPolicy>
  <RestrictPublicBuckets>true</RestrictPublicBuckets>
</PublicAccessBlockConfiguration>`), &publicAccess); err != nil {
		t.Fatalf("unmarshal public access block: %v", err)
	}
	if !awsS3PublicAccessBlockEnabled(publicAccess) {
		t.Fatalf("expected public access block to be fully enabled")
	}
}

func TestAwsS3BucketLifecycleAttrsExtractsRules(t *testing.T) {
	lifecycle := awsS3BucketLifecycleResponse{}
	if err := xml.Unmarshal([]byte(`
<LifecycleConfiguration>
  <Rule>
    <ID>archive-logs</ID>
    <Status>Enabled</Status>
    <Filter>
      <And>
        <Prefix>logs/</Prefix>
        <Tag>
          <Key>class</Key>
          <Value>audit</Value>
        </Tag>
        <ObjectSizeGreaterThan>1024</ObjectSizeGreaterThan>
      </And>
    </Filter>
    <Transition>
      <Days>30</Days>
      <StorageClass>STANDARD_IA</StorageClass>
    </Transition>
    <NoncurrentVersionTransition>
      <NoncurrentDays>60</NoncurrentDays>
      <StorageClass>GLACIER</StorageClass>
    </NoncurrentVersionTransition>
    <Expiration>
      <Days>365</Days>
    </Expiration>
    <NoncurrentVersionExpiration>
      <NoncurrentDays>730</NoncurrentDays>
      <NewerNoncurrentVersions>2</NewerNoncurrentVersions>
    </NoncurrentVersionExpiration>
    <AbortIncompleteMultipartUpload>
      <DaysAfterInitiation>7</DaysAfterInitiation>
    </AbortIncompleteMultipartUpload>
  </Rule>
</LifecycleConfiguration>`), &lifecycle); err != nil {
		t.Fatalf("unmarshal lifecycle: %v", err)
	}

	attrs := awsS3BucketLifecycleRuleAttrs(lifecycle.Rules)
	if len(attrs) != 1 || attrs[0]["id"] != "archive-logs" || attrs[0]["enabled"] != true {
		t.Fatalf("unexpected lifecycle attrs: %#v", attrs)
	}
	if attrs[0]["prefix"] != "logs/" {
		t.Fatalf("expected lifecycle prefix, got %#v", attrs[0]["prefix"])
	}
	transitions, ok := attrs[0]["transitions"].([]models.ResAttrs)
	if !ok || len(transitions) != 1 || transitions[0]["storageClass"] != "STANDARD_IA" {
		t.Fatalf("unexpected transitions: %#v", attrs[0]["transitions"])
	}
	abort, ok := attrs[0]["abortIncompleteMultipartUpload"].(models.ResAttrs)
	if !ok || abort["daysAfterInitiation"] != "7" {
		t.Fatalf("unexpected abort config: %#v", attrs[0]["abortIncompleteMultipartUpload"])
	}
}

func TestAwsS3BucketOptionalConfigMissing(t *testing.T) {
	if !awsS3BucketOptionalConfigMissing(errors.New("404 Not Found: <Code>NoSuchLifecycleConfiguration</Code>")) {
		t.Fatalf("expected lifecycle missing error to be optional")
	}
	if !awsS3BucketOptionalConfigMissing(errors.New("404 Not Found: <Code>NoSuchBucketPolicy</Code>")) {
		t.Fatalf("expected bucket policy missing error to be optional")
	}
	if !awsS3BucketOptionalConfigMissing(errors.New("404 Not Found: <Code>ObjectLockConfigurationNotFoundError</Code>")) {
		t.Fatalf("expected object lock missing error to be optional")
	}
	if awsS3BucketOptionalConfigMissing(errors.New("403 Forbidden: AccessDenied")) {
		t.Fatalf("access denied should remain visible")
	}
}

func TestAwsS3BucketGovernanceAttrsExtractsPolicyAclAndRetention(t *testing.T) {
	policy := map[string]interface{}{}
	if err := json.Unmarshal([]byte(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::demo/*"
    }
  ]
}`), &policy); err != nil {
		t.Fatalf("unmarshal policy: %v", err)
	}
	policyAttrs := awsS3BucketPolicyAttrs(policy)
	if policyAttrs["statementCount"] != 1 || policyAttrs["publicAllow"] != true {
		t.Fatalf("unexpected policy attrs: %#v", policyAttrs)
	}

	acl := awsS3BucketACLResponse{}
	if err := xml.Unmarshal([]byte(`
<AccessControlPolicy>
  <Owner>
    <ID>owner-id</ID>
    <DisplayName>owner</DisplayName>
  </Owner>
  <AccessControlList>
    <Grant>
      <Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="Group">
        <URI>http://acs.amazonaws.com/groups/global/AllUsers</URI>
      </Grantee>
      <Permission>READ</Permission>
    </Grant>
  </AccessControlList>
</AccessControlPolicy>`), &acl); err != nil {
		t.Fatalf("unmarshal acl: %v", err)
	}
	aclAttrs := awsS3BucketACLAttrs(acl)
	if aclAttrs["public"] != true || aclAttrs["owner"].(models.ResAttrs)["id"] != "owner-id" {
		t.Fatalf("unexpected acl attrs: %#v", aclAttrs)
	}

	objectLock := awsS3ObjectLockResponse{}
	if err := xml.Unmarshal([]byte(`
<ObjectLockConfiguration>
  <ObjectLockEnabled>Enabled</ObjectLockEnabled>
  <Rule>
    <DefaultRetention>
      <Mode>GOVERNANCE</Mode>
      <Days>30</Days>
    </DefaultRetention>
  </Rule>
</ObjectLockConfiguration>`), &objectLock); err != nil {
		t.Fatalf("unmarshal object lock: %v", err)
	}
	objectLockAttrs := awsS3ObjectLockAttrs(objectLock)
	retention := objectLockAttrs["defaultRetention"].(models.ResAttrs)
	if objectLockAttrs["enabled"] != true || retention["mode"] != "GOVERNANCE" || retention["days"] != "30" {
		t.Fatalf("unexpected object lock attrs: %#v", objectLockAttrs)
	}
}

func TestAwsS3BucketExtendedConfigAttrsExtractsOperationsFields(t *testing.T) {
	replication := awsS3BucketReplicationResponse{}
	if err := xml.Unmarshal([]byte(`
<ReplicationConfiguration>
  <Role>arn:aws:iam::123456789012:role/replication</Role>
  <Rule>
    <ID>replicate-logs</ID>
    <Status>Enabled</Status>
    <Priority>1</Priority>
    <Filter>
      <Prefix>logs/</Prefix>
    </Filter>
    <Destination>
      <Account>210987654321</Account>
      <Bucket>arn:aws:s3:::archive-bucket</Bucket>
      <StorageClass>STANDARD_IA</StorageClass>
      <EncryptionConfiguration>
        <ReplicaKmsKeyID>arn:aws:kms:ap-southeast-1:123:key/demo</ReplicaKmsKeyID>
      </EncryptionConfiguration>
      <Metrics>
        <Status>Enabled</Status>
        <EventThreshold><Minutes>15</Minutes></EventThreshold>
      </Metrics>
      <ReplicationTime>
        <Status>Enabled</Status>
        <Time><Minutes>15</Minutes></Time>
      </ReplicationTime>
    </Destination>
    <DeleteMarkerReplication><Status>Disabled</Status></DeleteMarkerReplication>
    <SourceSelectionCriteria>
      <SseKmsEncryptedObjects><Status>Enabled</Status></SseKmsEncryptedObjects>
    </SourceSelectionCriteria>
  </Rule>
</ReplicationConfiguration>`), &replication); err != nil {
		t.Fatalf("unmarshal replication: %v", err)
	}
	replicationAttrs := awsS3BucketReplicationAttrs(replication)
	if awsS3EnabledReplicationRuleCount(replication.Rules) != 1 {
		t.Fatalf("expected one enabled replication rule")
	}
	destinations := replicationAttrs["destinationBuckets"].([]string)
	if len(destinations) != 1 || destinations[0] != "arn:aws:s3:::archive-bucket" {
		t.Fatalf("unexpected replication destinations: %#v", destinations)
	}

	logging := awsS3BucketLoggingResponse{}
	if err := xml.Unmarshal([]byte(`
<BucketLoggingStatus>
  <LoggingEnabled>
    <TargetBucket>log-bucket</TargetBucket>
    <TargetPrefix>s3/</TargetPrefix>
  </LoggingEnabled>
</BucketLoggingStatus>`), &logging); err != nil {
		t.Fatalf("unmarshal logging: %v", err)
	}
	loggingAttrs := awsS3BucketLoggingAttrs(logging)
	if loggingAttrs["enabled"] != true || loggingAttrs["targetBucket"] != "log-bucket" {
		t.Fatalf("unexpected logging attrs: %#v", loggingAttrs)
	}

	notification := awsS3BucketNotificationResponse{}
	if err := xml.Unmarshal([]byte(`
<NotificationConfiguration>
  <TopicConfiguration>
    <Id>topic-rule</Id>
    <Topic>arn:aws:sns:ap-southeast-1:123:topic</Topic>
    <Event>s3:ObjectCreated:*</Event>
    <Filter>
      <S3Key>
        <FilterRule><Name>prefix</Name><Value>inbound/</Value></FilterRule>
      </S3Key>
    </Filter>
  </TopicConfiguration>
  <QueueConfiguration>
    <Id>queue-rule</Id>
    <Queue>arn:aws:sqs:ap-southeast-1:123:queue</Queue>
    <Event>s3:ObjectRemoved:*</Event>
  </QueueConfiguration>
  <EventBridgeConfiguration/>
</NotificationConfiguration>`), &notification); err != nil {
		t.Fatalf("unmarshal notification: %v", err)
	}
	if awsS3BucketNotificationCount(notification) != 3 {
		t.Fatalf("unexpected notification count: %#v", awsS3BucketNotificationAttrs(notification))
	}
}

func TestAwsS3BucketWebsiteAndInventoryAttrs(t *testing.T) {
	website := awsS3BucketWebsiteResponse{}
	if err := xml.Unmarshal([]byte(`
<WebsiteConfiguration>
  <IndexDocument><Suffix>index.html</Suffix></IndexDocument>
  <ErrorDocument><Key>error.html</Key></ErrorDocument>
  <RoutingRules>
    <RoutingRule>
      <Condition><HttpErrorCodeReturnedEquals>404</HttpErrorCodeReturnedEquals></Condition>
      <Redirect><ReplaceKeyWith>index.html</ReplaceKeyWith></Redirect>
    </RoutingRule>
  </RoutingRules>
</WebsiteConfiguration>`), &website); err != nil {
		t.Fatalf("unmarshal website: %v", err)
	}
	websiteAttrs := awsS3BucketWebsiteAttrs(website)
	if websiteAttrs["enabled"] != true || websiteAttrs["routingRuleCount"] != 1 {
		t.Fatalf("unexpected website attrs: %#v", websiteAttrs)
	}

	inventory := awsS3BucketInventoryResponse{}
	if err := xml.Unmarshal([]byte(`
<ListInventoryConfigurationsResult>
  <InventoryConfiguration>
    <Id>daily-inventory</Id>
    <IsEnabled>true</IsEnabled>
    <IncludedObjectVersions>All</IncludedObjectVersions>
    <Filter><Prefix>logs/</Prefix></Filter>
    <Destination>
      <S3BucketDestination>
        <AccountId>123456789012</AccountId>
        <Bucket>arn:aws:s3:::inventory-bucket</Bucket>
        <Format>Parquet</Format>
        <Prefix>inventory/</Prefix>
        <Encryption>
          <SSE-KMS><KeyId>arn:aws:kms:ap-southeast-1:123:key/demo</KeyId></SSE-KMS>
        </Encryption>
      </S3BucketDestination>
    </Destination>
    <OptionalFields><Field>Size</Field><Field>StorageClass</Field></OptionalFields>
    <Schedule><Frequency>Daily</Frequency></Schedule>
  </InventoryConfiguration>
</ListInventoryConfigurationsResult>`), &inventory); err != nil {
		t.Fatalf("unmarshal inventory: %v", err)
	}
	inventoryAttrs := awsS3BucketInventoryAttrs(inventory.InventoryConfigurations)
	if awsS3BucketInventoryEnabledCount(inventory.InventoryConfigurations) != 1 {
		t.Fatalf("expected one enabled inventory configuration")
	}
	if len(inventoryAttrs) != 1 || inventoryAttrs[0]["id"] != "daily-inventory" || inventoryAttrs[0]["scheduleFrequency"] != "Daily" {
		t.Fatalf("unexpected inventory attrs: %#v", inventoryAttrs)
	}
}
