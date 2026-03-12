# stackctl get secret - Usage Examples

## Basic Usage

### Copy to Clipboard (Default Behavior)

```bash
# Copy secret to clipboard
stackctl get secret MY_PASSWORD

# Copy from custom path (secret/data/ is auto-prepended)
stackctl get secret MY_PASSWORD --path team/credentials
```

## Save to File

### Save Secret to File

```bash
# Save public key to file
stackctl get secret PUB_KEY --to-file ~/.ssh/id_rsa.pub --path resources/vps/elias-oracle

# Save without overwriting existing file (will show warning)
stackctl get secret API_KEY --to-file ./api-key.txt
```

### Replace Existing File

```bash
# Force overwrite if file exists
stackctl get secret PUB_KEY --to-file ~/.ssh/id_rsa.pub --replace
```

## Base64 Decoding

### Decode Base64 Secret Before Saving

```bash
# Decode base64 encoded secret and save to file
stackctl get secret ENCODED_KEY --to-file ./decoded-key.txt --decode-from-b64

# Decode and copy to clipboard
stackctl get secret ENCODED_SECRET --decode-from-b64
```

## Combined Options

### Decode and Save with Replace

```bash
# Decode base64, save to file, and replace if exists
stackctl get secret PUB_KEY \
  --path resources/vps/elias-oracle \
  --to-file ~/.ssh/id_rsa.pub \
  --decode-from-b64 \
  --replace
```

## Environment Variable

### Set Default Path

```bash
# Set default path via environment variable (without secret/data/ prefix)
export STACK_CTL_DEFAULT_SECRET_PATH=team/credentials

# Now you can omit --path flag
stackctl get secret MY_PASSWORD
```

## Path Handling

The command automatically prepends `secret/data/` to all paths for KV v2 compatibility:

```bash
# You specify:
stackctl get secret KEY --path resources/vps/elias-oracle

# Actual Vault path used:
secret/data/resources/vps/elias-oracle
```

If you need to use a full path (already including `secret/data/`), the command detects it and won't double-prepend:

```bash
# This works too (not recommended, but supported):
stackctl get secret KEY --path secret/data/resources/vps/elias-oracle
```

## Behavior Notes

- When `--to-file` is used, the secret is **NOT** copied to clipboard
- Without `--replace`, the command will fail if the file already exists
- File permissions are set to `0600` (read/write for owner only) for security
- `--decode-from-b64` works with both clipboard and file output
- The secret value is never printed to the terminal for security
- Paths are automatically prepended with `secret/data/` for KV v2 compatibility
