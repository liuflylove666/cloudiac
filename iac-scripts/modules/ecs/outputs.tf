output "instance_ids"      { value = alicloud_instance.main[*].id }
output "private_ips"       { value = alicloud_instance.main[*].private_ip }
output "public_ips"        { value = alicloud_eip_address.main[*].ip_address }
output "key_pair_name"     { value = alicloud_key_pair.main.key_pair_name }