# SOPS Encrypted Secrets

This directory contains Kubernetes secrets encrypted with [SOPS](https://github.com/getsops/sops) using [age](https://github.com/FiloSottile/age) encryption.

## Prerequisites

Install required tools:
```bash
# Install SOPS
# macOS
brew install sops

# Linux
# Download from: https://github.com/getsops/sops/releases

# Install age
# macOS
brew install age

# Linux
sudo apt install age  # or download from https://github.com/FiloSottile/age/releases
```

## Setup

### 1. Generate Age Key (First Time Only)

```bash
# Generate a new age key
mkdir -p ~/.config/sops/age
age-keygen -o ~/.config/sops/age/keys.txt

# View your public key
grep "public key:" ~/.config/sops/age/keys.txt
```

**Important**: Save your age private key securely! If you lose it, you cannot decrypt the secrets.

### 2. Configure SOPS

The project already has a `.sops.yaml` configuration file in the root directory that specifies:
- Which files to encrypt (k8s/secrets/\*.yaml)
- The age public key to use for encryption

If you're setting up a new environment or rotating keys, update `.sops.yaml` with your age public key:
```yaml
creation_rules:
  - path_regex: k8s/secrets/.*\.yaml$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## Usage

### Viewing Encrypted Secrets

```bash
# View decrypted content
sops k8s/secrets/pinshare-backend-secret.yaml

# Or use your editor with SOPS
EDITOR=vim sops k8s/secrets/pinshare-backend-secret.yaml
```

### Editing Secrets

```bash
# Edit and re-encrypt in place
sops k8s/secrets/pinshare-backend-secret.yaml

# This will:
# 1. Decrypt the file
# 2. Open it in your $EDITOR
# 3. Re-encrypt it when you save and close
```

### Applying Secrets to Kubernetes

```bash
# Decrypt and apply directly to Kubernetes
sops --decrypt k8s/secrets/pinshare-backend-secret.yaml | kubectl apply -f -
sops --decrypt k8s/secrets/oauth-broker-secret.yaml | kubectl apply -f -

# Or use kubectl with SOPS
kubectl apply -f <(sops --decrypt k8s/secrets/pinshare-backend-secret.yaml)
```

### Creating New Encrypted Secrets

```bash
# Create a plain YAML file
cat > k8s/secrets/my-new-secret.yaml <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: pinshare
type: Opaque
stringData:
  MY_KEY: "my-value"
EOF

# Encrypt it in place
sops --encrypt --in-place k8s/secrets/my-new-secret.yaml

# Now it's encrypted and safe to commit to git
```

## Tilt Integration

Tilt automatically decrypts SOPS-encrypted secrets when applying Kubernetes manifests. The Tiltfile is already configured to decrypt and apply secrets:

```python
# Decrypt and apply secrets
k8s_yaml(local('sops --decrypt k8s/secrets/pinshare-backend-secret.yaml'))
k8s_yaml(local('sops --decrypt k8s/secrets/oauth-broker-secret.yaml'))
```

**Just run `tilt up`** and secrets will be automatically decrypted and applied!

## Security Best Practices

### ✅ DO:
- **Commit encrypted secret files** to git (`.yaml` files in this directory)
- **Backup your age private key** securely (not in git!)
- **Share age public keys** with team members
- **Rotate secrets regularly**
- **Use different age keys** for different environments (dev/staging/prod)

### ❌ DON'T:
- **Don't commit `.sops.yaml`** if it contains private keys (it shouldn't)
- **Don't commit age private keys** to git
- **Don't share age private keys** via insecure channels
- **Don't use the same secrets** across environments

## Team Collaboration

### Adding a Team Member

1. Team member generates their own age key:
   ```bash
   age-keygen -o ~/.config/sops/age/keys.txt
   ```

2. They share their **public key** (starts with `age1...`)

3. Update `.sops.yaml` to include multiple recipients:
   ```yaml
   creation_rules:
     - path_regex: k8s/secrets/.*\.yaml$
       age: >-
         age1alice...,
         age1bob...,
         age1charlie...
   ```

4. Re-encrypt all secrets for new recipients:
   ```bash
   sops updatekeys k8s/secrets/pinshare-backend-secret.yaml
   sops updatekeys k8s/secrets/oauth-broker-secret.yaml
   ```

### Rotating Keys

```bash
# 1. Generate new age key
age-keygen -o ~/.config/sops/age/keys-new.txt

# 2. Update .sops.yaml with new public key

# 3. Re-encrypt with new key
sops updatekeys k8s/secrets/*.yaml

# 4. Securely delete old key after verifying
```

## Troubleshooting

### "no key could be found to decrypt the data"

Your age private key is not in the expected location or doesn't match the encrypted files.

**Solution**:
```bash
# Check your key location
ls -la ~/.config/sops/age/keys.txt

# Verify SOPS can find it
export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt
sops --decrypt k8s/secrets/pinshare-backend-secret.yaml
```

### "error loading config: no matching creation rules found"

The `.sops.yaml` path regex doesn't match your file path.

**Solution**: Ensure you're in the project root and the path in `.sops.yaml` matches your file structure.

### Encrypted file looks corrupted

**Solution**: Decrypt and re-encrypt:
```bash
# Backup first!
cp k8s/secrets/pinshare-backend-secret.yaml k8s/secrets/pinshare-backend-secret.yaml.bak

# Decrypt to temp file
sops --decrypt k8s/secrets/pinshare-backend-secret.yaml > /tmp/decrypted.yaml

# Re-encrypt
sops --encrypt --in-place /tmp/decrypted.yaml
mv /tmp/decrypted.yaml k8s/secrets/pinshare-backend-secret.yaml
```

## Environment Variables

### For Scripts/Automation

```bash
# Specify age key location
export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt

# Or use age key directly
export SOPS_AGE_KEY=$(cat ~/.config/sops/age/keys.txt | grep -v "public key")
```

## References

- [SOPS Documentation](https://github.com/getsops/sops)
- [Age Encryption](https://github.com/FiloSottile/age)
- [SOPS with Kubernetes](https://devopstales.github.io/kubernetes/sops/)
- [Best Practices for Kubernetes Secrets](https://kubernetes.io/docs/concepts/security/secrets-good-practices/)

## Secret Files

| File | Description |
|------|-------------|
| `pinshare-backend-secret.yaml` | Backend secrets: VirusTotal token, Google OAuth, encryption key |
| `oauth-broker-secret.yaml` | OAuth broker secrets: Google OAuth credentials, redirect URL |

All files are encrypted with SOPS and safe to commit to version control.
