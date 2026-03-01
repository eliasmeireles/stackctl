# stackctl get pass - Usage Examples

## Basic Usage

### Copy to Clipboard (Default Behavior)
```bash
# Copy password to clipboard
stackctl get pass MY_PASSWORD

# Copy from custom path
stackctl get pass MY_PASSWORD --path secret/data/team/credentials
```

## Save to File

### Save Secret to File
```bash
# Save public key to file
stackctl get pass PUB_KEY --to-file ~/.ssh/id_rsa.pub --path secret/data/resources/vps/elias-oracle

# Save without overwriting existing file (will show warning)
stackctl get pass API_KEY --to-file ./api-key.txt
```

### Replace Existing File
```bash
# Force overwrite if file exists
stackctl get pass PUB_KEY --to-file ~/.ssh/id_rsa.pub --replace
```

## Base64 Decoding

### Decode Base64 Secret Before Saving
```bash
# Decode base64 encoded secret and save to file
stackctl get pass ENCODED_KEY --to-file ./decoded-key.txt --decode-from-b64

# Decode and copy to clipboard
stackctl get pass ENCODED_SECRET --decode-from-b64
```

## Combined Options

### Decode and Save with Replace
```bash
# Decode base64, save to file, and replace if exists
stackctl get pass PUB_KEY \
  --path secret/data/resources/vps/elias-oracle \
  --to-file ~/.ssh/id_rsa.pub \
  --decode-from-b64 \
  --replace
```

## Environment Variable

### Set Default Path
```bash
# Set default path via environment variable
export STACK_CTL_DEFAULT_PASS_PATH=secret/data/team/credentials

# Now you can omit --path flag
stackctl get pass MY_PASSWORD
```

## Behavior Notes

- When `--to-file` is used, the secret is **NOT** copied to clipboard
- Without `--replace`, the command will fail if the file already exists
- File permissions are set to `0600` (read/write for owner only) for security
- `--decode-from-b64` works with both clipboard and file output
- The secret value is never printed to the terminal for security
