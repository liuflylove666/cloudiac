variable "project"       { type = string }
variable "env"           { type = string }
variable "instance_type" { 
    type = string  
    default = "rds.mysql.s1.small" 
    }
variable "storage_size"  { 
    type = number  
    default = 20 
    }
variable "vswitch_id"    { type = string }
variable "allowed_ips"   { type = list(string) }
variable "db_name"       { 
    type = string  
    default = "appdb" 
    }
variable "db_user"       { 
    type = string  
    default = "appuser" 
    }
variable "db_password"   { 
    type = string  
    sensitive = true 
    }