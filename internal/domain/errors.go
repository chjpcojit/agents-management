// Package domain holds the core entities and business rules of the orchestrator.
// It has no dependency on any other internal package, framework, storage or transport.
//
// Package domain chứa các entity cốt lõi và business rule của orchestrator.
// Package này không phụ thuộc vào package internal nào khác, framework, storage hay transport.
package domain

import "errors"

// Sentinel errors returned by domain rules. Callers match them with errors.Is.
// Các sentinel error do domain rule trả về. Caller so khớp bằng errors.Is.
var (
	// ErrInvalidArgument means an input violates a domain invariant.
	// ErrInvalidArgument: input vi phạm một invariant của domain.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrNotFound means the requested entity does not exist.
	// ErrNotFound: entity được yêu cầu không tồn tại.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists means an entity with the same identity already exists.
	// ErrAlreadyExists: đã tồn tại entity có cùng định danh.
	ErrAlreadyExists = errors.New("already exists")

	// ErrApprovalAlreadyDecided means the approval left the pending state before this call.
	// ErrApprovalAlreadyDecided: approval đã rời trạng thái pending trước lần gọi này.
	ErrApprovalAlreadyDecided = errors.New("approval already decided")

	// ErrApprovalExpired means the decision arrived after the approval deadline.
	// ErrApprovalExpired: quyết định đến sau deadline của approval.
	ErrApprovalExpired = errors.New("approval expired")
)
