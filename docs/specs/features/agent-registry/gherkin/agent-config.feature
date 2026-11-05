# Source: ../requirements.md v1.0
Feature: Tạo và cấu hình agent

  # R-1
  Scenario: Tạo agent hợp lệ
    When owner tạo agent "backend-dev" model "opus" với prompt "Bạn là Go dev" và tool "Read", "Edit", "Bash"
    Then agent "backend-dev" được tạo ở trạng thái hoạt động với cấu hình phiên bản 1
    And web nhận event "agent.created"

  # R-2
  Scenario: Nhiều field sai được báo cùng lúc
    When owner tạo agent với slug "Backend Dev" và số lượt tối đa 0
    Then owner nhận lỗi không hợp lệ chỉ ra cả "slug" và "max_turns_per_task"
    And không agent nào được tạo

  # R-3
  Scenario: Slug đã có
    Given đã có agent "backend-dev" đã lưu trữ
    When owner tạo agent khác với slug "backend-dev"
    Then owner nhận lỗi slug đã được dùng

  # R-4
  Scenario: Model không được phép
    When owner tạo agent với model "gpt-9"
    Then owner nhận lỗi không hợp lệ ở field "model"

  # R-5
  Scenario: Danh sách được hỏi chứa agent lạ
    When owner tạo agent "reviewer" cho phép "ghost" hỏi nó
    Then owner nhận lỗi không hợp lệ ở field "can_message"

  # R-6
  Scenario: Danh sách không lộ prompt
    Given có agent "backend-dev" với prompt riêng
    When owner xem danh sách agent
    Then danh sách có "backend-dev" nhưng không có prompt

  # R-7
  Scenario: Xem chi tiết kèm quyền thực tế
    Given agent "backend-dev" được phép tool "Bash"
    When owner xem chi tiết "backend-dev"
    Then quyền thực tế xếp "Bash" vào nhóm phải hỏi duyệt

  # R-8
  Scenario: Sửa cấu hình với phiên bản cũ
    Given agent "backend-dev" đang ở cấu hình phiên bản 3
    When owner sửa model của "backend-dev" kèm phiên bản 2
    Then owner nhận lỗi xung đột phiên bản
    And model không đổi

  # R-8
  Scenario: Slug không đổi được
    When owner sửa slug của "backend-dev" thành "be"
    Then owner nhận lỗi không hợp lệ ở field "slug"

  # R-9
  Scenario: Cấu hình mới áp ở lần chạy sau
    Given agent "backend-dev" đang chạy một turn với model "sonnet"
    When owner đổi model thành "opus"
    Then turn đang chạy vẫn dùng "sonnet"
    And turn kế tiếp chạy bằng "opus"

  # R-10, R-11
  Scenario: Không gỡ được luật hỏi duyệt
    When owner cho agent "backend-dev" được phép "Bash(git push:*)"
    Then lệnh "git push" của "backend-dev" vẫn phải qua hộp duyệt

  # R-11
  Scenario: Luật tool sai cú pháp
    When owner cấm tool "Bash(("
    Then owner nhận lỗi không hợp lệ ở field "disallowed_tools"

  # R-12
  Scenario: Tạm dừng giữ hàng đợi
    Given agent "reviewer" có 2 tin đang xếp hàng
    When owner tạm dừng "reviewer"
    Then "reviewer" không nhận tin mới để xử lý
    And 2 tin vẫn còn trong hàng đợi

  # R-13
  Scenario: Tiếp tục agent tạm dừng
    Given agent "reviewer" đang tạm dừng với 2 tin xếp hàng
    When owner cho "reviewer" hoạt động lại
    Then "reviewer" xử lý tin đầu tiên trong hàng đợi

  # R-14
  Scenario: Không lưu trữ agent đang chạy
    Given agent "backend-dev" đang chạy một turn
    When owner xoá "backend-dev"
    Then owner nhận lỗi agent đang bận

  # R-14
  Scenario: Lưu trữ agent rảnh
    Given agent "reviewer" rảnh, có 1 câu hỏi xếp hàng từ "frontend", và "frontend" được phép hỏi "reviewer"
    When owner xoá "reviewer"
    Then "reviewer" ở trạng thái lưu trữ
    And câu hỏi xếp hàng bị huỷ và "frontend" được báo
    And worktree của "reviewer" vẫn còn

  # R-15
  Scenario: Thao tác sai trạng thái
    Given agent "reviewer" đã lưu trữ
    When owner tạm dừng "reviewer"
    Then owner nhận lỗi trạng thái không hợp lệ
