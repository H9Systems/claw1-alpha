terraform {
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "~> 6.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

provider "oci" {
  region = var.region
  # Auth via OCI env vars or ~/.oci/config.
}

# ── Networking ────────────────────────────────────────────────────────────────

resource "oci_core_vcn" "claw1" {
  compartment_id = var.compartment_id
  cidr_blocks    = ["10.0.0.0/16"]
  display_name   = "claw1-vcn"
  dns_label      = "claw1vcn"
}

resource "oci_core_internet_gateway" "claw1" {
  compartment_id = var.compartment_id
  vcn_id         = oci_core_vcn.claw1.id
  display_name   = "claw1-igw"
  enabled        = true
}

resource "oci_core_default_route_table" "claw1" {
  manage_default_resource_id = oci_core_vcn.claw1.default_route_table_id

  route_rules {
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.claw1.id
  }
}

resource "oci_core_security_list" "claw1" {
  compartment_id = var.compartment_id
  vcn_id         = oci_core_vcn.claw1.id
  display_name   = "claw1-seclist"

  ingress_security_rules {
    protocol = "6" # TCP
    source   = "0.0.0.0/0"
    tcp_options {
      min = 22
      max = 22
    }
  }

  egress_security_rules {
    protocol    = "all"
    destination = "0.0.0.0/0"
  }
}

locals {
  ssh_public_key_path  = pathexpand(var.ssh_public_key_path)
  ssh_private_key_path = pathexpand(var.ssh_private_key_path)
}

resource "oci_core_subnet" "claw1" {
  compartment_id    = var.compartment_id
  vcn_id            = oci_core_vcn.claw1.id
  cidr_block        = "10.0.1.0/24"
  display_name      = "claw1-subnet"
  dns_label         = "claw1subnet"
  security_list_ids = [oci_core_security_list.claw1.id]
  route_table_id    = oci_core_vcn.claw1.default_route_table_id
}

# ── Ubuntu 22.04 image ────────────────────────────────────────────────────────

data "oci_core_images" "ubuntu_22_04" {
  compartment_id           = var.compartment_id
  operating_system         = "Canonical Ubuntu"
  operating_system_version = "22.04"
  shape                    = var.shape
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
}

# ── VM ────────────────────────────────────────────────────────────────────────

resource "oci_core_instance" "claw1" {
  compartment_id      = var.compartment_id
  availability_domain = var.availability_domain
  display_name        = "claw1-l1-node"
  shape               = var.shape

  shape_config {
    ocpus         = var.shape_ocpus
    memory_in_gbs = var.shape_memory_gbs
  }

  source_details {
    source_type = "image"
    source_id   = data.oci_core_images.ubuntu_22_04.images[0].id
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.claw1.id
    assign_public_ip = true
  }

  metadata = {
    ssh_authorized_keys = file(local.ssh_public_key_path)
  }
}

# ── Bootstrap: deploy Avalanche L1 on the VM ─────────────────────────────────

resource "null_resource" "bootstrap_l1" {
  depends_on = [oci_core_instance.claw1]

  triggers = {
    instance_id = oci_core_instance.claw1.id
  }

  connection {
    type        = "ssh"
    host        = oci_core_instance.claw1.public_ip
    user        = "ubuntu"
    private_key = file(local.ssh_private_key_path)
    timeout     = "10m"
  }

  provisioner "file" {
    source      = "${path.module}/scripts/bootstrap.sh"
    destination = "/tmp/bootstrap.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/bootstrap.sh",
      "/tmp/bootstrap.sh 2>&1 | tee /tmp/claw1-bootstrap.log",
    ]
  }

  # Pull network.json from the VM to local disk so run.sh --oci can use it.
  provisioner "local-exec" {
    command = <<-EOT
      mkdir -p "$HOME/.claw1/claw1demobank-oci"
      scp -o StrictHostKeyChecking=no \
          -i ${local.ssh_private_key_path} \
          ubuntu@${oci_core_instance.claw1.public_ip}:/home/ubuntu/.claw1/claw1demobank/network.json \
          "$HOME/.claw1/claw1demobank-oci/network.json"
    EOT
  }
}

output "oci_vm_ip" {
  value = oci_core_instance.claw1.public_ip
}

output "ssh_command" {
  value = "ssh ubuntu@${oci_core_instance.claw1.public_ip}"
}

output "local_network_json" {
  value = pathexpand("~/.claw1/claw1demobank-oci/network.json")
}

output "ssh_private_key_path" {
  value = local.ssh_private_key_path
}
