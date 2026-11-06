# ADR-0004 — Secret của nguồn dữ liệu ngoài lưu ở GCP Secret Manager

Status: Accepted · Date: 2026-11-05 · Decider: Đức

## Context

Agent được cấp quyền vào nguồn dữ liệu ngoài (DB nghiệp vụ, API qua MCP server). MCP server cần credential (mật khẩu DB, API token). Orchestrator phải giữ chỗ tham chiếu tới credential đó. Lưu plaintext trong DB hoặc file config thì ai đọc được ổ đĩa hay bản backup cũng vào được hệ thống nghiệp vụ.

## Decision

1. Secret của nguồn dữ liệu nằm ở **GCP Secret Manager**. Postgres chỉ lưu **tên tài nguyên** (`projects/<project>/secrets/<name>/versions/<version>`), không bao giờ lưu giá trị.
2. Orchestrator đọc secret bằng Application Default Credentials (`gcloud auth application-default login` trên máy owner). Principal chỉ có `roles/secretmanager.secretAccessor` trên đúng các secret được dùng.
3. Giá trị secret chỉ được đưa vào env của process MCP server lúc khởi động, không vào env, prompt, `--mcp-config`, log hay event của agent.
4. Không thuộc ADR này: login Claude (do `claude` tự quản), mật khẩu Postgres của orchestrator (`.env`, ADR-0002), bearer token MCP của agent (sinh mỗi lần spawn).

## Consequences

- Thêm dependency `cloud.google.com/go/secretmanager` và cần mạng tới GCP khi spawn agent có nguồn dùng secret.
- Secret Manager không truy cập được thì agent có nguồn đó không được spawn (fail closed).
- Khi deploy lên GCP sau này, đổi principal sang service account của VM, không đổi thiết kế.
- Spec chi tiết: [data-access](../specs/features/data-access/requirements.md).
