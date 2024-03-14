# simple-backup
This is a simple git repository backup tools with Gitlab WebHook

* environ variables

```
LISTEN_PORT=8000
SG_HOOK_BACKUP_DIR=backup_dir
```

* start

```bash
$ go build -o gitlab-webhook-backup simple_backup.go
$ SG_HOOK_BACKUP_DIR=/tmp/repo_backup LISTEN_PORT=8000 ./gitlab-webhook-backup
```
