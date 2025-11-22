# Heimdal Ansible Playbook Migration Guide

## Overview

This guide explains how the Heimdal Ansible playbooks work with the new monorepo architecture and automatic configuration migration.

## Backward Compatibility

The Heimdal sensor binary now includes **automatic configuration migration** from the legacy format to the new format. This means:

1. **Existing playbooks continue to work** - No changes required to your Ansible playbooks
2. **Legacy configurations are automatically migrated** - When the sensor starts, it detects and migrates old configuration files
3. **Backups are created** - Original configuration files are backed up before migration

## Configuration Format Changes

### Legacy Format (Pre-Monorepo)

The legacy configuration format used by the original hardware-focused Heimdal sensor:

```json
{
  "database": { ... },
  "network": { ... },
  "discovery": {
    "arp_scan_interval_seconds": 60,
    "mdns_enabled": true,
    "inactive_timeout_minutes": 5
  },
  "interceptor": {
    "enabled": true,
    "spoof_interval_seconds": 2,
    "target_macs": []
  },
  "profiler": { ... },
  "api": { ... },
  "cloud": { ... },
  "logging": { ... }
}
```

### New Format (Monorepo)

The new format is identical to the legacy format for hardware deployments. The migration system is designed to handle future format changes transparently.

## Migration Process

When the Heimdal sensor starts:

1. **Detection**: The sensor checks if the configuration file is in legacy format
2. **Backup**: If legacy format is detected, a backup is created at `config.json.backup.YYYYMMDD-HHMMSS`
3. **Migration**: The configuration is converted to the new format
4. **Validation**: The migrated configuration is validated
5. **Logging**: Migration warnings are logged to stderr and the system journal

## Ansible Playbook Compatibility

### Current Playbooks (No Changes Required)

The existing Ansible playbooks in `ansible/roles/heimdal_sensor/` continue to work without modification:

- `tasks/main.yml` - Deployment tasks remain unchanged
- `templates/config.json.j2` - Configuration template remains unchanged
- `templates/heimdal.service.j2` - Systemd service remains unchanged
- `group_vars/all.yml` - Variables remain unchanged

### Binary Name

The hardware sensor binary is now named `heimdal` (previously may have been `heimdal-hardware`). The Ansible playbook already uses the correct name.

### Verification Steps

After deploying with Ansible, verify the migration:

```bash
# Check if migration occurred
sudo journalctl -u heimdal | grep -i migration

# Check for backup files
ls -la /etc/heimdal/config.json.backup.*

# Verify service is running
sudo systemctl status heimdal
```

## Optional: Updating to New Format

If you want to explicitly use the new format in your Ansible templates, the configuration structure remains the same for hardware deployments. No changes are needed.

## Rollback

If you need to rollback to a previous configuration:

```bash
# Stop the service
sudo systemctl stop heimdal

# Restore from backup
sudo cp /etc/heimdal/config.json.backup.YYYYMMDD-HHMMSS /etc/heimdal/config.json

# Start the service
sudo systemctl start heimdal
```

Note: The sensor will migrate the configuration again on next start. To prevent this, ensure your configuration includes the new format markers.

## Testing Migration

To test the migration process:

1. Deploy using existing Ansible playbooks
2. Check logs for migration messages
3. Verify sensor functionality
4. Confirm backup file was created

## Support

For issues with migration:

1. Check `/var/log/heimdal/heimdal.log` for detailed error messages
2. Check `journalctl -u heimdal` for systemd logs
3. Verify configuration file permissions and syntax
4. Review backup files to compare old and new formats

## Future Updates

The automatic migration system will handle future configuration format changes transparently. When new fields are added:

- Existing configurations will be migrated automatically
- Default values will be applied for new fields
- Backups will be created before migration
- Migration warnings will be logged

No Ansible playbook updates will be required for configuration format changes.
