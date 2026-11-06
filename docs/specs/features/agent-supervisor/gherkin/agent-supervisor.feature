# Source: ../requirements.md v1.0
Feature: Chạy và giám sát process của agent

  # R-1, R-6
  Scenario: Chạy một turn từ trạng thái offline
    Given agent "backend-dev" đang offline
    When scheduler giao cho "backend-dev" một tin của task "viết test"
    Then process của "backend-dev" được khởi động trong worktree của task "viết test"
    And scheduler nhận câu trả lời cuối turn

  # R-2
  Scenario: Không gửi hai tin cùng lúc
    Given "backend-dev" đang chạy một turn
    When scheduler cố giao thêm một tin cho "backend-dev"
    Then tin thứ hai chỉ được gửi sau khi turn đầu kết thúc

  # R-3
  Scenario: Lệnh khởi động theo cấu hình
    Given "backend-dev" dùng model "opus" và có quyền đọc "shop-db"
    When process của "backend-dev" được khởi động
    Then process chạy model "opus" và có MCP server của "shop-db"
    And prompt riêng không nằm trong tham số dòng lệnh

  # R-4
  Scenario: Nhớ hội thoại sau khi process dừng
    Given "backend-dev" đã trả lời một turn trong task "viết test"
    And process của "backend-dev" đã dừng
    When scheduler giao tin tiếp theo của task "viết test"
    Then process mới tiếp tục cùng phiên hội thoại

  # R-5
  Scenario: Event của turn có định danh
    When "backend-dev" gọi tool "Read" trong task "viết test"
    Then web nhận event gọi tool có agent, task, phiên và resource

  # R-7
  Scenario: Dùng lại process rảnh
    Given process của "backend-dev" đang rảnh trong task "viết test"
    When scheduler giao tin tiếp theo của task "viết test"
    Then không có process mới được khởi động

  # R-8
  Scenario: Đổi task thì đổi process
    Given process của "backend-dev" đang rảnh trong task "viết test"
    When scheduler giao tin của task "sửa lỗi login"
    Then process cũ được đóng và process mới chạy trong worktree của "sửa lỗi login"

  # R-8
  Scenario: Đổi cấu hình thì khởi động lại
    Given process của "backend-dev" đang rảnh
    And owner vừa đổi model của "backend-dev"
    When scheduler giao tin tiếp theo
    Then process mới được khởi động với model mới và cùng phiên hội thoại

  # R-9
  Scenario: Process rảnh lâu bị đóng
    Given process của "backend-dev" rảnh hơn 10 phút
    Then process được đóng và "backend-dev" ở trạng thái offline

  # R-10
  Scenario: Process chết giữa turn
    Given "backend-dev" đang chạy một turn
    When process của "backend-dev" thoát bất thường
    Then web nhận event "agent.exited"
    And tin đang xử lý được giao lại

  # R-11
  Scenario: Login Claude hết hạn
    When process của "backend-dev" báo lỗi xác thực
    Then "backend-dev" ở trạng thái lỗi với lý do xác thực
    And "backend-dev" không nhận turn mới tới khi owner đặt lại

  # R-12
  Scenario: Turn chạy quá lâu
    Given giới hạn một turn là 60 phút
    When một turn của "backend-dev" chạy quá 60 phút
    Then process bị dừng và tin được đánh dấu thất bại một lần

  # R-13
  Scenario: Dừng theo yêu cầu
    Given "backend-dev" đang chạy một turn
    When owner huỷ turn của "backend-dev"
    Then process của "backend-dev" dừng trong vòng 5 giây

  # R-14
  Scenario: Tạm dừng để turn chạy nốt
    Given "backend-dev" đang chạy một turn
    When owner tạm dừng "backend-dev"
    Then turn đang chạy kết thúc bình thường
    And "backend-dev" không nhận turn mới

  # R-15
  Scenario: Khởi động lại orchestrator
    Given "backend-dev" có phiên hội thoại đã lưu
    When orchestrator khởi động lại
    Then "backend-dev" ở trạng thái offline và phiên vẫn còn

  # R-16, R-17
  Scenario: Xem trạng thái runtime
    Given "backend-dev" đang chạy một turn
    When owner xem runtime của "backend-dev"
    Then owner thấy trạng thái đang chạy, worktree và phiên hiện tại

  # R-18
  Scenario: Đặt lại agent lỗi
    Given "backend-dev" ở trạng thái lỗi
    When owner đặt lại runtime của "backend-dev"
    Then "backend-dev" ở trạng thái offline và nhận lại turn
