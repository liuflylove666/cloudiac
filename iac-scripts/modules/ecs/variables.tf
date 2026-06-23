variable "project"           { type = string }
variable "env"               { type = string }
variable "instance_count"    { 
    type = number
    default = 1 
    }
variable "cpu_count"         { 
    type = number  
    default = 1 
    }
variable "memory_size"       { 
    type = number  
    default = 2 
    }
variable "disk_size"         { 
    type = number  
    default = 40 
    }
variable "bandwidth"         { 
    type = number  
    default = 0 
    }
variable "vswitch_id"        { type = string }
variable "security_group_id" { type = string }
variable "public_key"        { type = string }
variable "role"              { 
    type = string  
    default = "web" 
    } 