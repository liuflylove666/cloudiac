output "vpc_id"              { value = alicloud_vpc.main.id }
output "public_vswitch_id"   { value = alicloud_vswitch.public.id }
output "private_vswitch_id"  { value = alicloud_vswitch.private.id }
output "web_sg_id"           { value = alicloud_security_group.web.id }