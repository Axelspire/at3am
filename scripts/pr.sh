#!/bin/bash
# Usage: bash scripts/pr.sh <branch-name> "<PR title>" [tag]
#
# Examples:
#   bash scripts/pr.sh fix/tilde-expansion "fix: tilde expansion in at3am-hook"
#   bash scripts/pr.sh feat/static-linking "build: static linking for Linux" v0.1.5
#
# Workflow:
#   1. Creates feature branch from commits ahead of origin/main
#   2. Pushes feature branch
#   3. Resets local main to origin/main
#   4. Creates PR
#   5. Merges with --admin (bypasses review requirement)
#   6. Pulls latest main
#   7. Cleans up feature branch locally and remotely
#   8. Optionally creates and pushes a tag

set -e

BRANCH="$1"
TITLE="$2"
TAG="$3"
REPO_ROOT="$(git rev-parse --show-toplevel)"

cd "$REPO_ROOT"

# ── Validate inputs ────────────────────────────────────────────────────────────
if [ -z "$BRANCH" ] || [ -z "$TITLE" ]; then
  echo "Usage: bash scripts/pr.sh <branch-name> \"<PR title>\" [tag]"
  echo ""
  echo "Examples:"
  echo "  bash scripts/pr.sh fix/tilde-expansion \"fix: tilde expansion in at3am-hook\""
  echo "  bash scripts/pr.sh feat/static-linking \"build: static linking for Linux\" v0.1.5"
  exit 1
fi

# ── Check there are commits to PR ─────────────────────────────────────────────
COMMITS_AHEAD=$(git log origin/main..HEAD --oneline 2>/dev/null)
if [ -z "$COMMITS_AHEAD" ]; then
  echo "❌ No commits ahead of origin/main — nothing to PR."
  exit 1
fi

echo "📋 Commits to include in PR:"
echo "$COMMITS_AHEAD"
echo ""

# ── Create feature branch from current HEAD ───────────────────────────────────
echo "🌿 Creating branch: $BRANCH"
git checkout -b "$BRANCH"
git push origin "$BRANCH"
echo "✅ Feature branch pushed"
echo ""

# ── Reset local main back to origin/main ──────────────────────────────────────
git checkout main
git reset --hard origin/main

# ── Create PR ─────────────────────────────────────────────────────────────────
echo "📬 Creating PR..."
PR_BODY=$(git log origin/main.."origin/$BRANCH" --pretty=format:"- %s" 2>/dev/null || echo "- $TITLE")

PR_URL=$(gh pr create \
  --base main \
  --head "$BRANCH" \
  --title "$TITLE" \
  --body "## Changes

$PR_BODY

---
*Created via \`scripts/pr.sh\`*")

PR_NUM=$(echo "$PR_URL" | grep -o '[0-9]*$')
echo "✅ PR #$PR_NUM created: $PR_URL"
echo ""

# ── Merge with --admin (bypasses review requirement) ──────────────────────────
echo "🔀 Merging PR #$PR_NUM..."
gh pr merge "$PR_NUM" --merge --admin
echo "✅ PR #$PR_NUM merged"
echo ""

# ── Pull latest main ──────────────────────────────────────────────────────────
git pull origin main

# ── Clean up feature branch ───────────────────────────────────────────────────
echo "🧹 Cleaning up branch: $BRANCH"
git branch -D "$BRANCH" 2>/dev/null && echo "   Deleted local branch" || true
git push origin --delete "$BRANCH" 2>/dev/null && echo "   Deleted remote branch" || true

# ── Optionally create and push tag ────────────────────────────────────────────
if [ -n "$TAG" ]; then
  echo ""
  echo "🏷️  Creating tag: $TAG"
  git tag -a "$TAG" -m "$TITLE"
  git push origin "$TAG"
  echo "✅ Tag $TAG pushed"
fi

echo ""
echo "🎉 Done! Latest commits:"
git log --oneline -5

