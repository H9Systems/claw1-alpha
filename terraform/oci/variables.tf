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

variable "shape" {
  description = "OCI VM shape. Free tier: VM.Standard.A1.Flex (ARM) or VM.Standard.E2.1.Micro."
  type        = string
  default     = "VM.Standard.E4.Flex"
}

variable "shape_ocpus" {
  description = "OCPUs for flex shapes."
  type        = number
  default     = 1
}

variable "shape_memory_gbs" {
  description = "Memory in GBs for flex shapes."
  type        = number
  default     = 4
}

variable "ssh_public_key_path" {
  description = "Path to the SSH public key for the VM."
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to the SSH private key for provisioning (never stored in state)."
  type        = string
  default     = "~/.ssh/id_ed25519"
}
