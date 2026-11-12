#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${PIPELINE_BASE_URL:-http://localhost:8084}"
ENDPOINT="${BASE_URL%/}/api/v1/pipeline/analyze"
FIXTURE_DIR="${1:-$(cd "$(dirname "$0")/.." && pwd)/testdata/pipeline}"
EXPECTATIONS_FILE="${PIPELINE_EXPECTATIONS_FILE:-${FIXTURE_DIR}/expected_results.json}"

auth_args=()
if [[ -n "${PIPELINE_API_TOKEN:-}" ]]; then
  auth_args=(-H "Authorization: Bearer ${PIPELINE_API_TOKEN}")
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to run the pipeline validation matrix" >&2
  exit 1
fi

failures=0

printf 'Pipeline matrix against %s\n' "$ENDPOINT"
printf '%-20s %-12s %-8s %-8s %-16s %-16s %-10s\n' 'fixture' 'final_status' 'accept' 'stage1' 'stage4' 'stage6' 'assertion'

for fixture in "$FIXTURE_DIR"/*.json; do
  name="$(basename "$fixture" .json)"
  if [[ "$name" == "expected_results" ]]; then
    continue
  fi

  response="$(curl -sS -X POST "$ENDPOINT" -H 'Content-Type: application/json' "${auth_args[@]}" --data @"$fixture")"
  final_status="$(printf '%s' "$response" | jq -r '.final_status')"
  accepted="$(printf '%s' "$response" | jq -r '.accepted')"
  stage1="$(printf '%s' "$response" | jq -r '.stages[] | select(.stage==1) | .result.proposed_zone_id')"
  stage4="$(printf '%s' "$response" | jq -r '.stages[] | select(.stage==4) | .result.analyzer_zone_key')"
  stage6="$(printf '%s' "$response" | jq -r '.stages[] | select(.stage==6) | .result.zone_key')"

  assertion="SKIP"
  if [[ -f "$EXPECTATIONS_FILE" ]]; then
    expected="$(jq -r --arg name "$name" '.[$name] // empty' "$EXPECTATIONS_FILE")"
    if [[ -n "$expected" ]]; then
      expected_final_status="$(jq -r --arg name "$name" '.[$name].final_status' "$EXPECTATIONS_FILE")"
      expected_accepted="$(jq -r --arg name "$name" '.[$name].accepted' "$EXPECTATIONS_FILE")"
      expected_stage1="$(jq -r --arg name "$name" '.[$name].stage1' "$EXPECTATIONS_FILE")"
      expected_stage4="$(jq -r --arg name "$name" '.[$name].stage4' "$EXPECTATIONS_FILE")"
      expected_stage6="$(jq -r --arg name "$name" '.[$name].stage6' "$EXPECTATIONS_FILE")"

      if [[ "$final_status" == "$expected_final_status" && "$accepted" == "$expected_accepted" && "$stage1" == "$expected_stage1" && "$stage4" == "$expected_stage4" && "$stage6" == "$expected_stage6" ]]; then
        assertion="PASS"
      else
        assertion="FAIL"
        failures=$((failures + 1))
        printf 'Mismatch for %s\n' "$name" >&2
        printf '  expected: status=%s accept=%s stage1=%s stage4=%s stage6=%s\n' "$expected_final_status" "$expected_accepted" "$expected_stage1" "$expected_stage4" "$expected_stage6" >&2
        printf '  actual:   status=%s accept=%s stage1=%s stage4=%s stage6=%s\n' "$final_status" "$accepted" "$stage1" "$stage4" "$stage6" >&2
      fi
    fi
  fi

  printf '%-20s %-12s %-8s %-8s %-16s %-16s %-10s\n' "$name" "$final_status" "$accepted" "$stage1" "$stage4" "$stage6" "$assertion"
done

if (( failures > 0 )); then
  exit 1
fi