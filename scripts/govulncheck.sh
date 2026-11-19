#!/usr/bin/env bash
#
# Run govulncheck and fail only on reachable vulnerabilities that
# .govulncheck-allow does not list.
#
# The upstream golang/govulncheck-action has no allowlist, so a single finding
# we cannot act on leaves this check permanently red — and a permanently red
# check is one nobody reads. That is not hypothetical here: it is how twelve
# security advisories sat untriaged for four months. The allowlist keeps the
# signal, and every entry has to say why it is there.
#
# Same script in CI and locally (`make vulncheck`), so the two cannot disagree.
set -euo pipefail

cd "$(dirname "$0")/.."

ALLOW_FILE=${ALLOW_FILE:-.govulncheck-allow}

# Default to a temp file so a local run leaves nothing behind in the repo. Set
# REPORT to keep the raw JSON for inspection.
REPORT=${REPORT:-}
if [ -z "$REPORT" ]; then
	REPORT=$(mktemp -t govulncheck-XXXXXX)
	trap 'rm -f "$REPORT"' EXIT
fi

if ! command -v jq >/dev/null 2>&1; then
	echo "jq is required (preinstalled on GitHub runners; brew install jq locally)" >&2
	exit 2
fi

if ! command -v govulncheck >/dev/null 2>&1; then
	echo "installing govulncheck"
	go install golang.org/x/vuln/cmd/govulncheck@latest
fi

# govulncheck exits non-zero when it finds anything. Here that is data, so the
# exit status is deliberately ignored and the report is what decides.
govulncheck -format json ./... >"$REPORT" || true

# Keep only findings whose trace names a function: those are reachable from our
# code. A finding without one says the module is in the build graph and nothing
# more, which is not something to fail a build over.
found=$(jq -r 'select(.finding != null)
	| select([.finding.trace[]? | select(.function != null)] | length > 0)
	| .finding.osv' "$REPORT" | sort -u)

allowed=$(sed 's/#.*//' "$ALLOW_FILE" | tr -d '[:blank:]' |
	grep -E '^GO-[0-9]{4}-[0-9]+$' | sort -u || true)

blocking=$(comm -23 <(printf '%s\n' "$found") <(printf '%s\n' "$allowed") | grep -E '^GO-' || true)

printf 'reachable: %s\n' "$(printf '%s\n' "$found" | grep -cE '^GO-' || true)"
printf 'allowlisted: %s\n' "$(printf '%s\n' "$allowed" | grep -cE '^GO-' || true)"

if [ -n "$blocking" ]; then
	echo
	echo "Blocking vulnerabilities:"
	while IFS= read -r id; do
		printf '  %s  %s\n' "$id" "$(jq -r --arg id "$id" \
			'select(.osv != null and .osv.id == $id) | .osv.summary' "$REPORT" | head -1)"
	done <<<"$blocking"
	echo
	echo "Bump the dependency, or add the id to $ALLOW_FILE with a reason, how you"
	echo "checked it, and a review date. Read $ALLOW_FILE before adding anything."
	exit 1
fi

echo "No blocking vulnerabilities."
