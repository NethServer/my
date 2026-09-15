# Third-party Application Icons

This directory holds the icons Logto shows for each third-party application on
the consent page (the "X wants to access your account" screen) and in the
account center. Keeping them here means the icons live in git, next to the
application definition, instead of only in the Logto console.

## Files structure

```
configs/
├── config.yml          # Application definitions
├── sign-in/            # Sign-in experience assets (tenant-wide branding)
└── apps/               # This directory: one icon pair per application
    ├── helpdesk.svg        # Light theme
    ├── helpdesk-dark.svg   # Dark theme
    ├── nethshop.svg
    ├── nethshop-dark.svg
    ├── nethspot.svg
    ├── nethspot-dark.svg
    ├── training.svg
    ├── training-dark.svg
    ├── warehouse.svg       # stock.nethesis.it (NethStock)
    └── warehouse-dark.svg
```

SVG is preferred; PNG, JPEG, GIF and ICO work too. The file is uploaded inline
as a data URL, so keep it small — the icons here are 1-2 KB each.

## Configuration

Reference the pair from the application entry in `config.yml`. Paths are
relative to the config file directory, like the sign-in experience assets:

```yaml
third_party_apps:
  - name: "helpdesk.nethesis.it"
    description: "Helpdesk Nethesis"
    display_name: "Helpdesk Nethesis"
    branding:
      logo_path: "apps/helpdesk.svg"
      logo_dark_path: "apps/helpdesk-dark.svg"
```

Both paths are optional: configure only `logo_path` and Logto uses it for both
themes.

Apply with the usual command — icons are pushed together with the display name,
on both newly created and existing applications:

```bash
./sync sync --config configs/config.yml
```

## Removing icons

Logto merges the branding it receives, so dropping the `branding` block leaves
the icons already uploaded in place. To clear them, sync an empty block:

```yaml
    branding: {}
```

## Notes

- A configured file that is missing or unreadable fails the sync for that
  application, including in `--dry-run`, instead of silently skipping it.
- Icons uploaded from the Logto console are replaced on the next sync of an
  application that configures `branding`.
