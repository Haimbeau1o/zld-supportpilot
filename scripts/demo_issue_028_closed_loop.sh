#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${BASE_URL:-http://localhost:8080}
EMAIL=${EMAIL:-issue28-$(date +%s)@example.com}
PASSWORD=${PASSWORD:-secret123}
TENANT_NAME=${TENANT_NAME:-中联数据智能服务台演示租户}
USER_NAME=${USER_NAME:-Issue28 Demo User}
QUESTION=${QUESTION:-VPN 691 错误怎么处理？}
GENERIC_CONTENT=${GENERIC_CONTENT:-VPN 691 错误通常需要由 IT 服务台人工处理，请先提交工单，稍后由客服继续排查。}
CANDIDATE_TITLE=${CANDIDATE_TITLE:-VPN 691 错误排查}
CANDIDATE_SUMMARY=${CANDIDATE_SUMMARY:-通过重置账号拨号权限恢复}
CANDIDATE_CONTENT=${CANDIDATE_CONTENT:-处理步骤：检查账号状态，重置账号拨号权限，重新连接验证。}
EXPECTED_SNIPPET=${EXPECTED_SNIPPET:-重置账号拨号权限}

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

log() {
  printf '[issue-028 demo] %s\n' "$*"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

request_json() {
  local method=$1
  local path=$2
  local body=${3-}
  local token=${4-}
  local response http_code response_body
  local -a cmd=(curl -sS -w '\n%{http_code}' -X "$method" "$BASE_URL$path")
  if [[ -n "$token" ]]; then
    cmd+=(-H "Authorization: Bearer $token")
  fi
  if [[ -n "$body" ]]; then
    cmd+=(-H "Content-Type: application/json" -d "$body")
  fi
  # 用数组执行 curl，避免 JSON body 中的空格或中文被 shell 二次拆词。
  response="$("${cmd[@]}")"
  http_code=$(printf '%s' "$response" | tail -n1)
  response_body=$(printf '%s' "$response" | sed '$d')
  if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
    echo "request failed: $method $path -> HTTP $http_code" >&2
    echo "$response_body" >&2
    exit 1
  fi
  printf '%s' "$response_body"
}

upload_file() {
  local path=$1
  local file_path=$2
  local token=$3
  local response http_code response_body
  response=$(curl -sS -w '\n%{http_code}' -X POST "$BASE_URL$path" \
    -H "Authorization: Bearer $token" \
    -F "file=@$file_path")
  http_code=$(printf '%s' "$response" | tail -n1)
  response_body=$(printf '%s' "$response" | sed '$d')
  if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
    echo "upload failed: POST $path -> HTTP $http_code" >&2
    echo "$response_body" >&2
    exit 1
  fi
  printf '%s' "$response_body"
}

wait_document_ready() {
  local document_id=$1
  local token=$2
  local attempts=${3:-80}
  for ((attempt=1; attempt<=attempts; attempt++)); do
    local body status
    body=$(request_json GET "/api/v1/knowledge/documents/$document_id" "" "$token")
    status=$(printf '%s' "$body" | jq -er '.status')
    if [[ "$status" == "ready" ]]; then
      return 0
    fi
    if [[ "$status" == "failed" ]]; then
      echo "document $document_id processing failed" >&2
      echo "$body" >&2
      exit 1
    fi
    sleep 0.1
  done
  echo "document $document_id did not become ready in time" >&2
  exit 1
}

require_cmd curl
require_cmd jq
require_cmd mktemp

log "检查服务健康状态"
request_json GET "/healthz" >/dev/null

GENERIC_FILE="$TMP_DIR/vpn-general-guide.txt"
printf '%s\n' "$GENERIC_CONTENT" > "$GENERIC_FILE"

log "注册演示用户：$EMAIL"
request_json POST "/api/v1/auth/register" "$(jq -nc --arg name "$USER_NAME" --arg email "$EMAIL" --arg password "$PASSWORD" --arg tenant "$TENANT_NAME" '{name:$name,email:$email,password:$password,tenant_name:$tenant}')" >/dev/null

log "登录获取 access token"
LOGIN_BODY=$(request_json POST "/api/v1/auth/login" "$(jq -nc --arg email "$EMAIL" --arg password "$PASSWORD" '{email:$email,password:$password}')")
ACCESS_TOKEN=$(printf '%s' "$LOGIN_BODY" | jq -er '.access_token')

log "创建知识库"
KB_BODY=$(request_json POST "/api/v1/knowledge/bases" "$(jq -nc '{name:"IT 支持知识库",description:"用于沉淀 IT 文档"}')" "$ACCESS_TOKEN")
KNOWLEDGE_BASE_ID=$(printf '%s' "$KB_BODY" | jq -er '.id')

log "上传一份泛化旧知识，模拟第一次问答只能给出泛化建议"
UPLOAD_BODY=$(upload_file "/api/v1/knowledge/bases/$KNOWLEDGE_BASE_ID/documents" "$GENERIC_FILE" "$ACCESS_TOKEN")
GENERIC_DOCUMENT_ID=$(printf '%s' "$UPLOAD_BODY" | jq -er '.id')
wait_document_ready "$GENERIC_DOCUMENT_ID" "$ACCESS_TOKEN"

log "第一次统一 AI 受理"
FIRST_INTAKE_BODY=$(request_json POST "/api/v1/ai/intakes" "$(jq -nc --arg knowledge_base_id "$KNOWLEDGE_BASE_ID" --arg question "$QUESTION" '{knowledge_base_id:$knowledge_base_id,question:$question,top_k:3}')" "$ACCESS_TOKEN")
FIRST_RESULT_TYPE=$(printf '%s' "$FIRST_INTAKE_BODY" | jq -er '.result_type')
FIRST_INTAKE_ID=$(printf '%s' "$FIRST_INTAKE_BODY" | jq -er '.intake_id')
if [[ "$FIRST_RESULT_TYPE" != "answered" ]]; then
  echo "expected first intake result_type answered, got $FIRST_RESULT_TYPE" >&2
  echo "$FIRST_INTAKE_BODY" >&2
  exit 1
fi

log "提交 unresolved 反馈，触发人工跟进工单"
FEEDBACK_BODY=$(request_json POST "/api/v1/ai/intakes/$FIRST_INTAKE_ID/feedback" "$(jq -nc '{status:"unresolved",comment:"按建议操作后仍失败，需要人工继续处理。"}')" "$ACCESS_TOKEN")
TICKET_ACTION=$(printf '%s' "$FEEDBACK_BODY" | jq -er '.ticket_action')
TICKET_ID=$(printf '%s' "$FEEDBACK_BODY" | jq -er '.ticket_id')
if [[ "$TICKET_ACTION" != "ticket_created" ]]; then
  echo "expected feedback ticket_action ticket_created, got $TICKET_ACTION" >&2
  echo "$FEEDBACK_BODY" >&2
  exit 1
fi

log "触发 AI 工单辅助，把 AI 建议沉淀到时间线"
ASSIST_BODY=$(request_json POST "/api/v1/ai/tickets/$TICKET_ID/assist" "$(jq -nc --arg knowledge_base_id "$KNOWLEDGE_BASE_ID" '{knowledge_base_id:$knowledge_base_id}')" "$ACCESS_TOKEN")
RECORDED_COMMENT_ID=$(printf '%s' "$ASSIST_BODY" | jq -er '.recorded_comment_id')
[[ -n "$RECORDED_COMMENT_ID" ]] || {
  echo "missing recorded_comment_id" >&2
  echo "$ASSIST_BODY" >&2
  exit 1
}

log "将工单推进到 resolved"
request_json POST "/api/v1/tickets/$TICKET_ID/status" '{"status":"in_progress"}' "$ACCESS_TOKEN" >/dev/null
request_json POST "/api/v1/tickets/$TICKET_ID/status" '{"status":"resolved"}' "$ACCESS_TOKEN" >/dev/null

log "从已解决工单创建知识候选"
CANDIDATE_BODY=$(request_json POST "/api/v1/knowledge/candidates" "$(jq -nc --arg ticket_id "$TICKET_ID" --arg knowledge_base_id "$KNOWLEDGE_BASE_ID" --arg title "$CANDIDATE_TITLE" --arg summary "$CANDIDATE_SUMMARY" --arg content "$CANDIDATE_CONTENT" '{ticket_id:$ticket_id,knowledge_base_id:$knowledge_base_id,title:$title,summary:$summary,content:$content}')" "$ACCESS_TOKEN")
CANDIDATE_ID=$(printf '%s' "$CANDIDATE_BODY" | jq -er '.id')

log "审核通过并自动入库"
REVIEW_BODY=$(request_json POST "/api/v1/knowledge/candidates/$CANDIDATE_ID/review" '{"action":"approve","comment":"内容可入库"}' "$ACCESS_TOKEN")
APPROVED_DOCUMENT_ID=$(printf '%s' "$REVIEW_BODY" | jq -er '.approved_document_id')
wait_document_ready "$APPROVED_DOCUMENT_ID" "$ACCESS_TOKEN"

log "第二次统一 AI 受理，验证新知识已命中"
SECOND_INTAKE_BODY=$(request_json POST "/api/v1/ai/intakes" "$(jq -nc --arg knowledge_base_id "$KNOWLEDGE_BASE_ID" --arg question "$QUESTION" '{knowledge_base_id:$knowledge_base_id,question:$question,top_k:3}')" "$ACCESS_TOKEN")
SECOND_RESULT_TYPE=$(printf '%s' "$SECOND_INTAKE_BODY" | jq -er '.result_type')
if [[ "$SECOND_RESULT_TYPE" != "answered" ]]; then
  echo "expected second intake result_type answered, got $SECOND_RESULT_TYPE" >&2
  echo "$SECOND_INTAKE_BODY" >&2
  exit 1
fi

if ! printf '%s' "$SECOND_INTAKE_BODY" | jq -er --arg expected "$EXPECTED_SNIPPET" '
  (.answer | contains($expected)) or ([.citations[].content] | map(contains($expected)) | any)
' >/dev/null; then
  echo "expected second intake to hit approved knowledge content" >&2
  echo "$SECOND_INTAKE_BODY" >&2
  exit 1
fi

log "主闭环演示完成"
log "knowledge_base_id=$KNOWLEDGE_BASE_ID"
log "ticket_id=$TICKET_ID"
log "candidate_id=$CANDIDATE_ID"
log "approved_document_id=$APPROVED_DOCUMENT_ID"
