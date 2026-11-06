# Source: ../requirements.md v1.0
Feature: Lưu trữ Postgres local

  # R-1, R-2, R-3
  Scenario: Dựng Postgres trên máy owner
    Given file ".env" có mật khẩu Postgres
    When owner chạy "make db-up"
    Then Postgres sẵn sàng và chỉ nghe trên 127.0.0.1 cổng 5432

  # R-4
  Scenario: Thiếu cấu hình DB
    Given chưa đặt địa chỉ kết nối DB
    When orchestrator khởi động
    Then orchestrator thoát và báo thiếu "ORCH_DATABASE_URL" cùng mọi lỗi cấu hình khác

  # R-5, SEC-1
  Scenario: DB không chạy
    Given Postgres đang tắt
    When orchestrator khởi động
    Then sau 30 giây orchestrator thoát với lỗi "database unreachable"
    And log không chứa mật khẩu DB

  # R-6
  Scenario: Migration chạy khi khởi động
    Given DB mới, chưa có bảng nào
    When orchestrator khởi động
    Then mọi bảng được tạo và orchestrator bắt đầu nhận request

  # R-7
  Scenario: Binary cũ gặp schema mới
    Given DB đã ở schema mới hơn bản orchestrator đang chạy
    When orchestrator khởi động
    Then orchestrator từ chối khởi động và báo hai phiên bản schema

  # R-8
  Scenario: Readiness khi DB rớt
    Given orchestrator đang chạy
    When Postgres bị tắt
    Then kiểm tra readiness trả 503 và chỉ ra phụ thuộc "database"

  # R-9
  Scenario: Lỗi giữa chừng không để lại dữ liệu dở
    When một thao tác ghi hai bảng và lỗi sau khi ghi bảng thứ nhất
    Then không bảng nào bị thay đổi

  # R-10, R-13
  Scenario: Hai quyết định đồng thời trên Postgres
    Given có yêu cầu đang chờ duyệt
    When hai người duyệt cùng lúc gửi cho phép và từ chối
    Then đúng một quyết định được lưu
    And quyết định còn lại nhận lỗi đã quyết định

  # R-11
  Scenario: Xung đột transaction được thử lại
    Given một transaction gặp lỗi serialization lần đầu
    When thao tác được chạy
    Then thao tác thành công ở lần thử thứ hai

  # R-12
  Scenario: Shutdown đóng DB cuối cùng
    When owner dừng orchestrator
    Then kết nối DB chỉ đóng sau khi HTTP và agent đã dừng

  # R-14
  Scenario: Approval treo sau restart
    Given có yêu cầu đang chờ duyệt
    When orchestrator khởi động lại
    Then yêu cầu ở trạng thái hết hạn với lý do "orchestrator khởi động lại"
