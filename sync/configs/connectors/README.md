# Email Connectors Configuration

This directory contains email connector configurations and templates for Logto SMTP integration.

## Overview

The connectors configuration enables professional email communication for authentication flows, including password reset emails with custom branding and templates.

## Files Structure

```
configs/connectors/
├── README.md                    # This documentation file
└── forgot-password.html        # HTML template for password reset emails
```

## SMTP Connector Configuration

Configure SMTP settings in your main configuration file (`config.yml`):

```yaml
connectors:
  smtp:
    # SMTP server configuration
    host: "smtp.your-provider.com"
    port: 587
    username: "your-smtp-username"
    password: "your-smtp-password"
    from_email: "no-reply@your-company.com"
    from_name: "Your Company Name"
    
    # Security settings
    tls: true
    secure: false
    require_tls: true
    
    # Debugging and logging
    debug: false
    logger: true
    
    # Security restrictions
    disable_file_access: true
    disable_url_access: true
    
    # Template variable settings for dynamic content replacement
    template_settings:
      company_name: "Your Company Name"
      support_email: "support@your-company.com"
    
    # Custom headers (optional)
    custom_headers: {}
```

## Template Variables

Templates support dynamic variable replacement:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{code}}` | Verification code | `123456` |
| `{{.CompanyName}}` | Company name from template_settings | `"Your Company Name"` |
| `{{.SupportEmail}}` | Support email from template_settings | `"support@your-company.com"` |

## Email Templates

### Password Reset Template (forgot-password.html)

Professional HTML template for password reset emails featuring:

- **Branded Header**: Company logo and gradient background
- **Clear Messaging**: Professional password reset instructions
- **Prominent Code Display**: Verification code in highlighted box
- **Security Notice**: Important warnings about unauthorized requests
- **Professional Footer**: Company branding and contact information
- **Mobile Responsive**: Optimized for all device sizes

#### Template Features:
- Custom Nethesis branding with logo
- Red gradient header for urgency/security
- Monospace code display for clarity
- Warning section with security notice
- Professional typography (Poppins font family)
- Mobile-responsive design with breakpoints

### Template Customization

To customize the password reset template:

1. Edit `forgot-password.html` in this directory
2. Use template variables for dynamic content:
   - `{{code}}` - The verification code
   - `{{.CompanyName}}` - Your company name
   - `{{.SupportEmail}}` - Your support email
3. Run `sync sync` to update Logto configuration

#### Logo Customization:
Update the logo URL in the template:
```html
<img src="https://your-domain.com/logo.png" alt="Company logo" width="80" height="auto" />
```

#### Color Customization:
Modify the gradient background in the header section:
```html
style="background: linear-gradient(135deg, #your-color-1 0%, #your-color-2 50%, #your-color-3 100%);"
```

## Configuration Fields

### SMTP Settings

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | ✅ | SMTP server hostname |
| `port` | integer | ❌ | SMTP server port (default: 587) |
| `username` | string | ✅ | SMTP authentication username |
| `password` | string | ✅ | SMTP authentication password |
| `from_email` | string | ✅ | Sender email address |
| `from_name` | string | ❌ | Sender display name |

### Security Settings

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `tls` | boolean | `false` | Enable TLS encryption |
| `secure` | boolean | `false` | Use secure connection |
| `disable_file_access` | boolean | `true` | Disable file system access |
| `disable_url_access` | boolean | `true` | Disable URL access |

### Template Settings

| Field | Type | Description |
|-------|------|-------------|
| `company_name` | string | Company name for template variables |
| `support_email` | string | Support email for template variables |

## Usage Examples

### Basic SMTP Setup
```yaml
connectors:
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    from_email: "no-reply@yourcompany.com"
    from_name: "Your Company"
    tls: true
```

### Production Configuration
```yaml
connectors:
  smtp:
    host: "email-smtp.eu-west-1.amazonaws.com"
    port: 587
    username: "AKIAXXXXXXXXXXXXXXXX"
    password: "your-ses-smtp-password"
    from_email: "no-reply@yourcompany.com"
    from_name: "Your Company | Production"
    tls: true
    secure: false
    logger: true
    disable_file_access: true
    disable_url_access: true
    template_settings:
      company_name: "Your Company | Production"
      support_email: "support@yourcompany.com"
```

## Supported Email Templates

Currently supported template types:

- **ForgotPassword**: Custom HTML template from `forgot-password.html`
- **SignIn**: Default plain text template
- **Register**: Default plain text template  
- **Generic**: Default plain text template

## Testing

After configuration, test the SMTP setup:

```bash
# Preview connector changes
./build/sync sync --dry-run --verbose

# Apply connector configuration
./build/sync sync

# Test password reset flow in Logto
```

## Troubleshooting

### Common Issues

**SMTP Connection Failed**
- Verify host and port settings
- Check username/password credentials
- Ensure TLS settings match your provider

**Template Not Loading**
- Verify `forgot-password.html` exists in this directory
- Check file permissions and path
- Review sync logs for template loading errors

**Template Variables Not Replaced**
- Ensure `template_settings` are configured
- Check variable syntax: `{{.CompanyName}}`, `{{.SupportEmail}}`
- Verify template content uses correct variable names

### Debug Mode

Enable debug logging for detailed SMTP information:

```yaml
connectors:
  smtp:
    debug: true
    logger: true
    # ... other settings
```

## Security Considerations

- **Credentials**: Use environment variables or secure storage for SMTP passwords
- **TLS**: Always enable TLS for production environments
- **Access Controls**: Keep `disable_file_access` and `disable_url_access` enabled
- **Template Security**: Avoid including sensitive information in email templates

## Provider-Specific Examples

### AWS SES
```yaml
host: "email-smtp.us-east-1.amazonaws.com"
port: 587
tls: true
```

### Gmail
```yaml
host: "smtp.gmail.com"
port: 587
tls: true
# Use App Password, not regular password
```

### Outlook/Hotmail
```yaml
host: "smtp-mail.outlook.com"  
port: 587
tls: true
```