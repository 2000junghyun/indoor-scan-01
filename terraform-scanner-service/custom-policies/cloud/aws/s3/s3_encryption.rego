# METADATA
# title: Ensure S3 bucket has encryption enabled
# description: S3 buckets should be encrypted to protect data at rest
# scope: package
# custom:
#   id: CUSTOM-S3-001
#   avd_id: CUSTOM-S3-001
#   provider: AWS
#   service: S3
#   severity: HIGH
#   recommended_action: Enable encryption on S3 buckets
package custom.s3.encryption

import rego.v1

# Check for aws_s3_bucket resources without server_side_encryption_configuration
deny contains result if {
	some bucket_name, bucket_config in input.resource.aws_s3_bucket
	not bucket_config.server_side_encryption_configuration
	
	result := {
		"msg": sprintf("S3 bucket '%s' does not have encryption enabled", [bucket_name]),
		"startline": object.get(bucket_config, "__startline__", 0),
		"endline": object.get(bucket_config, "__endline__", 0),
		"filepath": "",
	}
}
