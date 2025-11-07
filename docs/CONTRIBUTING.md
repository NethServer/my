# Contributing to My Documentation

Thank you for your interest in improving the My documentation!

## Documentation Structure

```
docs/
├── index.md                    # Home page
├── 01-authentication.md        # Authentication guide
├── 02-organizations.md         # Organizations guide
├── 03-users.md                 # Users management guide
├── 04-systems.md               # Systems management guide
├── 05-system-registration.md   # Registration workflow
├── 06-inventory-heartbeat.md   # Monitoring guide
├── 07-impersonation.md         # User impersonation guide
├── stylesheets/                # Custom CSS
├── javascripts/                # Custom JS
└── images/                     # Images and diagrams
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

### Example Structure

```markdown
# Page Title

Brief introduction explaining what this page covers.

## Main Section

Detailed explanation with examples.

### Subsection

Specific details or procedures.

**Example:**
\`\`\`bash
command --flag value
\`\`\`

## Troubleshooting

Common problems and solutions.

## Related Documentation

- [Link to related page](other-page.md)
```

## Building Locally

### Prerequisites

```bash
# Install Python 3.x
python3 --version

# Install dependencies
pip install -r requirements.txt
```

### Local Development

```bash
# Start local server with hot reload
mkdocs serve

# Open in browser
open http://localhost:8000
```

The documentation will automatically reload when you save changes.

### Building

```bash
# Build static site
mkdocs build

# Output will be in site/ directory
```

## Making Changes

### 1. Edit Documentation

Edit the relevant `.md` file in the `docs/` directory.

### 2. Preview Locally

```bash
mkdocs serve
```

Check your changes at http://localhost:8000

### 3. Check Links

Ensure all internal links work:
- Relative links to other docs: `[text](other-file.md)`
- Links to sections: `[text](other-file.md#section-name)`
- External links: Full URL

### 4. Add Images

If adding images:

1. Place image in `docs/images/`
2. Use relative path: `![Alt text](images/filename.png)`
3. Optimize image size (max 1MB)

### 5. Test Build

```bash
# Test that build succeeds
mkdocs build --strict

# This will fail if there are warnings
```

## Adding New Pages

### 1. Create File

Create new `.md` file in `docs/` directory:

```bash
touch docs/07-new-feature.md
```

### 2. Add to Navigation

Edit `mkdocs.yml` and add to navigation:

```yaml
nav:
  - User Guide:
      - New Feature: 07-new-feature.md
```

### 3. Link from Other Pages

Add links from relevant pages:

```markdown
See also: [New Feature Guide](07-new-feature.md)
```

## MkDocs Features

### Admonitions

```markdown
!!! note "Title"
    Content here

!!! warning
    Warning content

!!! danger
    Danger content

!!! tip
    Tip content
```

### Code Blocks with Highlighting

````markdown
```python title="example.py" linenums="1"
def hello():
    print("Hello, World!")
```
````

### Tabs

```markdown
=== "Tab 1"
    Content for tab 1

=== "Tab 2"
    Content for tab 2
```

### Task Lists

```markdown
- [x] Completed task
- [ ] Incomplete task
```

## Deployment

Documentation is automatically deployed when you push to `main`:

1. GitHub Actions runs on push
2. MkDocs builds the site
3. Site is deployed to GitHub Pages
4. Available at: https://nethesis.github.io/my/

You can also deploy manually:

```bash
mkdocs gh-deploy
```

## Review Process

1. Make your changes in a feature branch
2. Test locally with `mkdocs serve`
3. Ensure build passes: `mkdocs build --strict`
4. Create Pull Request
5. Documentation will be reviewed
6. Once approved, merge to main
7. Automatic deployment to GitHub Pages

## Common Tasks

### Update Home Page

Edit `docs/index.md`

### Add FAQ Section

Create `docs/faq.md` and add to navigation in `mkdocs.yml`

### Fix Broken Link

1. Search for the link: `grep -r "broken-link" docs/`
2. Update all occurrences
3. Test with `mkdocs serve`

### Add External Resource

Add to `mkdocs.yml` under `extra`:

```yaml
extra:
  social:
    - icon: fontawesome/brands/github
      link: https://github.com/example
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

## Getting Help

- Check existing documentation for examples
- Review [MkDocs documentation](https://www.mkdocs.org/)
- Check [Material theme docs](https://squidfunk.github.io/mkdocs-material/)
- Ask in project discussions

## License

Documentation contributions are covered by the same license as the project (AGPL-3.0-or-later).
