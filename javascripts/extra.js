// Extra JavaScript for My documentation

// Add copy button functionality enhancement
document.addEventListener('DOMContentLoaded', function() {
  // Add click handler for code blocks
  document.querySelectorAll('pre > code').forEach(function(codeBlock) {
    codeBlock.addEventListener('click', function() {
      const button = this.parentElement.querySelector('.md-clipboard');
      if (button) {
        const originalTitle = button.title;
        button.title = 'Copied!';
        setTimeout(function() {
          button.title = originalTitle;
        }, 2000);
      }
    });
  });

  // Auto-scroll to active nav item
  const activeLink = document.querySelector('.md-nav__link--active');
  if (activeLink) {
    activeLink.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
  }

  // Enhance tables with sorting (optional)
  document.querySelectorAll('table').forEach(function(table) {
    table.classList.add('sortable-table');
  });
});

// Add analytics tracking for external links (if analytics configured)
if (typeof gtag !== 'undefined') {
  document.addEventListener('click', function(e) {
    if (e.target.tagName === 'A' && e.target.href.startsWith('http')) {
      gtag('event', 'click', {
        event_category: 'external_link',
        event_label: e.target.href,
        transport_type: 'beacon'
      });
    }
  });
}
