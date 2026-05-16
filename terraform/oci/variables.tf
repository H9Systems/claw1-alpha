variable "compartment_id" {
  description = "OCI compartment OCID."
  type        = string
}

variable "region" {
  description = "OCI region (e.g. us-ashburn-1)."
  type        = string
  default     = "us-ashburn-1"
}

variable "availability_domain" {
  description = "Availability domain name (e.g. aBCD:US-ASHBURN-AD-1)."
  type        = string
}

variable "ssh_public_key_path" {
  description = "Path to the SSH public key for the VM."
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

variable "ssh_private_key_path" {
  description = "Path to the SSH private key for provisioning (never stored in state)."
  type        = string
  default     = "~/.ssh/id_rsa"
}
