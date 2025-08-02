# Sign-in Experience Assets

This directory contains the default assets for customizing the Logto sign-in experience.

## Files Structure

```
configs/
├── config.yml              # Main configuration file
└── sign-in/                 # Sign-in experience assets
    ├── README.md           # This file
    ├── default.css         # Default CSS styles for sign-in pages
    ├── logo.png            # Logo for light theme (optional)
    ├── logo-dark.png       # Logo for dark theme (optional)
    ├── favicon.ico         # Favicon for light theme (optional)
    └── favicon-dark.ico    # Favicon for dark theme (optional)
```

## Configuration-Based Workflow

The sign-in experience is now managed through the `config.yml` file and applied using the `sync` command:

### 1. Configure in config.yml

Add the `sign_in_experience` section to your `configs/config.yml`:

```yaml
# Sign-in experience configuration
sign_in_experience:
  # Brand colors (hex format)
  colors:
    primary_color: "#0069A8"
    primary_color_dark: "#0087DB"
    dark_mode_enabled: true

  # Branding assets (relative paths from config file directory)
  branding:
    logo_path: "sign-in/logo.png"
    logo_dark_path: "sign-in/logo-dark.png"
    favicon_path: "sign-in/favicon.ico"
    favicon_dark_path: "sign-in/favicon-dark.ico"

  # Custom CSS (relative path from config file directory)
  custom_css_path: "sign-in/default.css"

  # Language configuration
  language:
    auto_detect: true
    fallback_language: "en"

  # Sign-in methods configuration
  sign_in:
    methods:
      - identifier: "email"
        password: true
        verification_code: false
        is_password_primary: true

  # Sign-up configuration (disabled by default)
  sign_up:
    identifiers: []
    password: false
    verify: false
    secondary_identifiers: []

  # Social sign-in configuration (empty by default)
  social_sign_in: {}
```

### 2. Apply Configuration

Use the `sync` command to apply the configuration:

```bash
# Preview changes (dry-run)
sync sync --config configs/config.yml --dry-run

# Apply configuration
sync sync --config configs/config.yml
```

### 3. Workflow Integration

```bash
# Initial setup (creates basic Logto structure)
sync init --tenant-id your-tenant --backend-app-id your-app --backend-app-secret your-secret

# Configure sign-in experience (and other RBAC settings)
sync sync --config configs/config.yml
```

## File Requirements

### Images (Logo & Favicon)
- **Formats**: PNG, JPG, SVG, ICO, GIF
- **Size**: Recommended dimensions for logos: 200x60px, favicons: 32x32px
- **Encoding**: Files are automatically converted to base64 data URLs

### CSS
- **Format**: Standard CSS file
- **Theme Support**: Use `html[data-theme="light"]` and `html[data-theme="dark"]` selectors
- **Classes**: Target Logto CSS classes (see default.css for examples)

### Colors
- **Format**: Hex color codes (e.g., #0069A8)
- **Light Theme**: Primary brand color for buttons, links, accents
- **Dark Theme**: Primary brand color optimized for dark backgrounds

## Default Configuration Applied

When using these assets, the following sign-in experience configuration is applied:

- **Colors**: Primary #0069A8 (light), #0087DB (dark)
- **Dark Mode**: Enabled
- **Language**: Auto-detect with English fallback
- **Sign-in Method**: Email + Password (password primary)
- **Sign-up**: Disabled
- **Social Sign-in**: Disabled

## Customization

You can customize the sign-in experience by:

1. **Replacing files**: Put your custom logo, favicon, and CSS in this directory
2. **Modifying CSS**: Edit `default.css` to change colors, layout, animations  
3. **Changing colors**: Update `colors.primary_color` and `colors.primary_color_dark` in config.yml
4. **Custom paths**: Update the file paths in the `branding` section of config.yml
5. **Configuration**: Modify sign-in methods, language settings, etc. in config.yml

## API Integration

The sign-in experience configuration uses the Logto Management API:

- **Endpoint**: `PATCH /api/sign-in-exp` 
- **Documentation**: https://openapi.logto.io/dev/operation/operation-updatesigninexp
- **Authentication**: Requires M2M application with Management API access

### Configuration Structure

```json
{
  "color": {
    "primaryColor": "#0069A8",
    "isDarkModeEnabled": true,
    "darkPrimaryColor": "#0087DB"
  },
  "branding": {
    "logoUrl": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
    "darkLogoUrl": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
    "favicon": "data:image/x-icon;base64,AAABAAEAEBAAAAEAIABoBAA...",
    "darkFavicon": "data:image/x-icon;base64,AAABAAEAEBAAAAEAIABoBAA..."
  },
  "customCss": "/* CSS content here */",
  "languageInfo": {
    "autoDetect": true,
    "fallbackLanguage": "en"
  },
  "signIn": {
    "methods": [
      {
        "identifier": "email",
        "password": true,
        "verificationCode": false,
        "isPasswordPrimary": true
      }
    ]
  },
  "signUp": {
    "identifiers": [],
    "password": false,
    "verify": false,
    "secondaryIdentifiers": []
  },
  "socialSignIn": {}
}
```

## File Detection Logic

The `sync sync` command loads files based on the paths specified in `config.yml`:

1. **CSS**: Loads from `custom_css_path` (e.g., `sign-in/default.css`)
2. **Logo**: Loads from `branding.logo_path` (e.g., `sign-in/logo.png`)
3. **Dark Logo**: Loads from `branding.logo_dark_path` (e.g., `sign-in/logo-dark.png`)
4. **Favicon**: Loads from `branding.favicon_path` (e.g., `sign-in/favicon.ico`)
5. **Dark Favicon**: Loads from `branding.favicon_dark_path` (e.g., `sign-in/favicon-dark.ico`)
6. **Brand Colors**: Uses `colors.primary_color` and `colors.primary_color_dark` from config

**Note**: All paths are relative to the config file directory (`configs/`)

## CSS Classes and Selectors

The default CSS targets these Logto-specific classes:

- `.logto_main-content` - Main panel background
- `.logto_page-container` - Full page container (for animated background)
- `input[type="email"]`, `input[type="password"]` - Input fields
- `button.odkpo_large` - Primary buttons
- `html[data-theme="light"]` / `html[data-theme="dark"]` - Theme selectors

## Troubleshooting

### Common Issues

1. **Files not loading**: Ensure file paths are correct and files exist
2. **CSS not applying**: Check CSS selectors match Logto's class names
3. **Images not showing**: Verify image formats are supported (PNG, JPG, SVG, ICO, GIF)
4. **Build fails**: Run `sync init --help` to verify flag names

### Debug Commands

```bash
# Test if files are detected
ls -la ./configs/sign-in/

# Verify CSS syntax
cat ./configs/sign-in/default.css

# Check file sizes (large files may cause issues)
du -h ./configs/sign-in/*

# Test configuration loading
sync sync --config configs/config.yml --dry-run --verbose
```

## Notes

- Files are loaded during sync and embedded as data URLs
- Missing files are logged as debug messages but don't cause sync to fail
- All assets are optional - you can configure only the ones you need
- File size limit: Keep images under 1MB for better performance
- Paths in config.yml are relative to the config file directory (`configs/`)
- Use `sync sync` to apply sign-in experience changes (not `sync init`)