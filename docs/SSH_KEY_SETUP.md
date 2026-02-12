# SSH Key Upload Usage

## 1. Generate key pair (if needed)
```bash
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N ""
```

## 2. Upload local public key to remote server
```bash
scripts/upload_ssh_key.sh --host 192.168.1.100 --user root
```

Optional parameters:
- `--port`: SSH port, default `22`
- `--pubkey`: local public key path, default `~/.ssh/id_rsa.pub`
- `--doc-only`: only generate this doc file, skip upload step

## 3. Verify login with key auth
```bash
ssh -i ~/.ssh/id_rsa root@192.168.1.100
```

If it enters shell without password prompt, key auth is ready.

## 4. Common failures
- Public key file missing:
  - Run `ssh-keygen` first, or pass correct `--pubkey`
- Permission denied (publickey):
  - Confirm remote user is correct
  - Check `~/.ssh` and `~/.ssh/authorized_keys` permissions on remote host
- Network timeout:
  - Check IP/port and firewall
