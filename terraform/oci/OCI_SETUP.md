# OCI Setup — Hackathon Fast Path

Get from zero to `terraform apply` in ~15 minutes.

## Step 1: Sign up / log in
Go to https://cloud.oracle.com/free — sign up or log in.

## Step 2: Generate API signing key (in OCI Console)

1. Top-right corner → click your profile avatar → **My Profile**
2. Under **Resources** → **API Keys** → **Add API Key**
3. Select **Generate API Key Pair** → **Download Private Key**
4. Save the private key file (shown as `*.pem`) → move to `~/.oci/oci_api_key.pem`
5. Click **Add** — it shows you the config snippet. Copy it.

```bash
mkdir -p ~/.oci
chmod 700 ~/.oci
# Paste the downloaded private key:
mv ~/Downloads/*.pem ~/.oci/oci_api_key.pem
chmod 600 ~/.oci/oci_api_key.pem
```

## Step 3: Create ~/.oci/config

Paste the config snippet from Step 2 into `~/.oci/config`. It looks like:

```ini
[DEFAULT]
user=ocid1.user.oc1..XXXXXXXXXX
fingerprint=xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx
tenancy=ocid1.tenancy.oc1..XXXXXXXXXX
region=us-ashburn-1
key_file=~/.oci/oci_api_key.pem
```

## Step 4: Get your compartment OCID

- OCI Console → **Identity & Security** → **Compartments**
- Use the **root compartment** OCID (or create a new one for this project)
- Format: `ocid1.compartment.oc1..XXXXXXXXXX` (or `ocid1.tenancy.oc1..XXXXXXXXXX` for root)

## Step 5: Get your availability domain

- OCI Console → **Compute** → **Instances** → **Create Instance**
- Look at the **Placement** section → copy the Availability Domain name
- Format: `XXXX:US-ASHBURN-AD-1` (varies by region)

## Step 6: Create terraform.tfvars

```bash
cp terraform/oci/terraform.tfvars.example terraform/oci/terraform.tfvars
# Edit with your values:
nano terraform/oci/terraform.tfvars
```

For free tier (recommended — uses Ampere ARM cores):
```hcl
compartment_id      = "ocid1.compartment.oc1..YOUR_OCID"
availability_domain = "XXXX:US-ASHBURN-AD-1"
region              = "us-ashburn-1"
shape               = "VM.Standard.A1.Flex"
shape_ocpus         = 2
shape_memory_gbs    = 8
```

## Step 7: Deploy

```bash
# Make sure you have an SSH key pair
ls ~/.ssh/id_ed25519.pub || ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519

cd terraform/oci
terraform init
terraform apply
```

Takes ~10-15 minutes (VM provision + Avalanche L1 bootstrap).

## Step 8: Deploy contracts

Once `terraform apply` completes:

```bash
cd ../..   # back to repo root
./run.sh --oci
```

This opens an SSH tunnel and deploys contracts through it.

## Troubleshooting

- **Auth error**: Check `~/.oci/config` — fingerprint and key_file path must match
- **Shape not available**: Try `us-ashburn-1` (most shapes available there), or change to `VM.Standard.E2.1.Micro` (smallest free tier)
- **bootstrap.sh fails**: SSH into the VM and check `/tmp/claw1-bootstrap.log`
