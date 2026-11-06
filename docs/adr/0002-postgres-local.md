# ADR-0002 — Postgres local qua docker compose, `pgx/v5` + `goose`

Status: Accepted · Date: 2026-11-05 · Decider: Đức

## Context

Approval-inbox đang lưu in-memory (approval-inbox#TD-1): restart là mất approval, mailbox và task. MVP chạy trên máy owner (1 máy, 1 user). Hướng ban đầu là SQLite, Đức chọn đổi sang Postgres để gần môi trường deploy sau này.

## Decision

1. Orchestrator lưu mọi dữ liệu điều phối trong **Postgres 17**, chạy bằng **docker compose** trên máy owner, port chỉ bind `127.0.0.1`.
2. Driver: **`github.com/jackc/pgx/v5`** (`pgxpool`). Không dùng `database/sql` + `lib/pq` (maintenance mode).
3. Migration: **`github.com/pressly/goose/v3`**, file SQL thuần trong `migrations/`, embed vào binary, chạy lúc khởi động. Forward-only.
4. Mật khẩu DB nằm trong `.env` (gitignored); repo chỉ có `.env.example` với giá trị giả.
5. Unit test vẫn dùng fake của port. Test repository chạy với Postgres của compose, tách bằng build tag `integration`.

## Consequences

- Owner phải chạy Docker trước orchestrator (`make db-up`).
- Thêm 2 dependency Go và một service ngoài process; readiness phải kiểm DB.
- Mã hoá at rest dựa vào mã hoá ổ đĩa của máy owner (FileVault).
- Spec chi tiết: [storage](../specs/features/storage/requirements.md).
