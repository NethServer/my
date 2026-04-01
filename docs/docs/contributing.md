---
sidebar_position: 99
---

# Contributing to My Documentation

Thank you for your interest in improving the My documentation!

## Documentation Structure

```
docs/
├── intro.md                              # Home page
├── getting-started/
│   ├── authentication.md                 # Authentication guide
│   └── account.md                        # Account settings guide
├── platform/
│   ├── organizations.md                  # Organizations guide
│   ├── users.md                          # Users management guide
│   └── impersonation.md                  # User impersonation guide
├── systems/
│   ├── management.md                     # Systems management guide
│   ├── registration.md                   # Registration workflow
│   └── inventory-heartbeat.md            # Monitoring guide
├── features/
│   ├── dashboard.md                      # Dashboard overview
│   ├── applications.md                   # Applications guide
│   ├── avatar.md                         # Avatar management
│   ├── rebranding.md                     # Organization rebranding
│   └── export.md                         # Data export
└── contributing.md                       # This file
```

## Writing Guidelines

### Style Guide

- **Tone**: Clear, professional, helpful
- **Audience**: End users and administrators (non-technical)
- **Language**: Simple, avoiding jargon when possible
- **Examples**: Always include practical examples

### Formatting

- **Headers**: Use `##` for main sections, `###` for subsections
- **Code blocks**: Always specify language (bash, json, python, etc.)
- **Lists**: Use `-` for unordered lists, `1.` for ordered
- **Emphasis**: Use **bold** for important terms, *italic* for emphasis
- **Links**: Use descriptive text, not "click here"

### Admonitions

Use Docusaurus admonition syntax:

```markdown
:::note
Informational content here
:::

:::tip
Helpful tip content here
:::

:::warning
Warning content here
:::

:::danger
Danger content here
:::
```

### Example Structure

````markdown
# Page Title

Brief introduction explaining what this page covers.

## Main Section

Detailed explanation with examples.

### Subsection

Specific details or procedures.

**Example:**
```bash
command --flag value
```

## Troubleshooting

Common problems and solutions.

## Related Documentation

- [Link to related page](./other-page)
````

## Building Locally

### Prerequisites

- Node.js 20+
- npm

```bash
# Install dependencies
npm install
```

### Local Development

```bash
# Start local server with hot reload
npm start

# Open in browser
open http://localhost:3000
```

The documentation will automatically reload when you save changes.

### Building

```bash
# Build static site
npm run build

# Output will be in build/ directory
```

## Making Changes

### 1. Edit Documentation

Edit the relevant `.md` file in the `docs/` directory.

### 2. Preview Locally

```bash
npm start
```

Check your changes at http://localhost:3000

### 3. Check Links

Ensure all internal links work:
- Relative links to other docs: `[text](relative-path)` (without `.md` extension)
- Links to sections: `[text](relative-path#section-name)`
- External links: Full URL

### 4. Add Images

If adding images:

1. Place image in `static/img/`
2. Reference using: `![Alt text](/img/filename.png)`
3. Optimize image size (max 1MB)

### 5. Test Build

```bash
# Test that build succeeds
npm run build

# This will fail if there are broken links or errors
```

## Adding New Pages

### 1. Create File

Create new `.md` file in the appropriate `docs/` subdirectory:

```bash
touch docs/features/new-feature.md
```

### 2. Add Frontmatter

Every page needs frontmatter with at minimum a `sidebar_position`:

```markdown
---
sidebar_position: 6
---

# New Feature

Content here...
```

### 3. Link from Other Pages

Add links from relevant pages using relative paths without the `.md` extension:

```markdown
See also: [New Feature Guide](new-feature)
```

## Style and Conventions

### Command Examples

Always show complete commands:

```bash
# Good
curl -X POST https://api.example.com/endpoint \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'

# Bad
curl endpoint
```

### File Paths

Use absolute paths from project root:

```bash
# Good
/Users/edospadoni/Workspace/my/backend/main.go

# Bad
../backend/main.go
```

### API Examples

Show both request and response:

```bash
# Request
curl -X GET https://api.example.com/resource

# Response (HTTP 200)
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

## Deployment

Documentation is automatically deployed when changes are pushed to the `main` branch:

1. GitHub Actions runs on push
2. Docusaurus builds the site
3. Site is deployed to GitHub Pages

## Review Process

1. Make your changes in a feature branch
2. Test locally with `npm start`
3. Ensure build passes: `npm run build`
4. Create Pull Request
5. Documentation will be reviewed
6. Once approved, merge to main
7. Automatic deployment

## Getting Help

- Check existing documentation for examples
- Review [Docusaurus documentation](https://docusaurus.io/docs)
- Ask in project discussions

## License

Documentation contributions are covered by the same license as the project (AGPL-3.0-or-later).
