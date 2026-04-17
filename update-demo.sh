#!/bin/bash
#
# Update GitHub Pages demo site
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Updating GitHub Pages Demo ==="
echo ""

# Check if demo file exists
if [ ! -f demo/index.html ]; then
    echo "Error: demo/index.html not found!"
    exit 1
fi

# Create docs directory if it doesn't exist
mkdir -p docs

# Copy demo to docs
echo "📄 Copying demo/index.html to docs/index.html..."
cp demo/index.html docs/index.html

# Check file size
FILESIZE=$(wc -c < docs/index.html)
echo "✓ Demo file size: $(numfmt --to=iec --suffix=B $FILESIZE 2>/dev/null || echo $FILESIZE bytes)"

# Git operations
echo ""
echo "📝 Committing changes..."

if git diff --quiet docs/index.html 2>/dev/null; then
    echo "ℹ️  No changes to commit"
else
    git add docs/index.html demo/index.html
    git commit -m "Update demo page: $(date +%Y-%m-%d\ %H:%M:%S)"

    echo ""
    echo "📤 Pushing to GitHub..."
    git push origin main

    echo ""
    echo "✅ Demo updated successfully!"
    echo ""
    echo "📋 Next steps:"
    echo "  1. Visit GitHub repository settings"
    echo "  2. Enable GitHub Pages (Settings → Pages)"
    echo "  3. Select branch: main, folder: /docs"
    echo "  4. Wait 1-3 minutes for deployment"
    echo ""
    echo "🌐 Live URL: https://liqcui.github.io/reignx/"
    echo ""
    echo "⏳ Deployment in progress..."
    echo "   Check status: https://github.com/liqcui/reignx/actions"
fi

echo ""
echo "Done!"
