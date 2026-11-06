# Source: ../requirements.md v1.0
Feature: Nguồn dữ liệu ngoài và quyền của agent

  # R-1, R-3
  Scenario: Đăng ký nguồn MCP Postgres
    When owner đăng ký nguồn "shop-db" loại MCP với tool "query" và mật khẩu lấy từ secret "projects/demo/secrets/shop-db-ro/versions/latest"
    Then nguồn "shop-db" được lưu chỉ với tên secret

  # R-2
  Scenario Outline: Thư mục nhạy cảm bị từ chối
    When owner đăng ký nguồn thư mục "<đường dẫn>"
    Then owner nhận lỗi đường dẫn bị cấm

    Examples:
      | đường dẫn               |
      | ~/.ssh                  |
      | ~                       |
      | link trỏ vào ~/.aws     |

  # R-4
  Scenario: Domain web sai định dạng
    When owner đăng ký nguồn web với domain "https://example.com/path"
    Then owner nhận lỗi không hợp lệ ở field "domains"

  # R-5
  Scenario: Xoá nguồn thu hồi quyền của agent
    Given agent "backend-dev" có quyền đọc "shop-db"
    When owner xoá nguồn "shop-db"
    Then "backend-dev" mất quyền "shop-db" và process của nó bị dừng

  # R-6
  Scenario: Cấp quyền áp ở lần chạy sau
    Given agent "backend-dev" đang chạy một turn
    When owner cấp quyền đọc "design-docs" cho "backend-dev"
    Then turn đang chạy không thấy "design-docs"
    And turn kế tiếp đọc được "design-docs"

  # R-7
  Scenario: Xem quyền của agent
    Given agent "backend-dev" có quyền đọc "shop-db"
    When owner xem quyền của "backend-dev"
    Then owner thấy "shop-db" với tool "query" và không thấy giá trị secret nào

  # R-8, R-10
  Scenario: Không có quyền thì không thấy nguồn
    Given agent "frontend" không có quyền trên "shop-db"
    When "frontend" chạy
    Then "frontend" không có tool nào của "shop-db"

  # R-9
  Scenario: Ghi vào nguồn vẫn phải duyệt
    Given agent "backend-dev" có quyền ghi thư mục "design-docs"
    When "backend-dev" sửa một file trong "design-docs"
    Then yêu cầu sửa file vào hộp duyệt

  # R-11
  Scenario: Thu hồi có hiệu lực ngay
    Given agent "backend-dev" đang chạy turn dùng "shop-db"
    When owner thu hồi quyền "shop-db" của "backend-dev"
    Then process của "backend-dev" dừng trong vòng 5 giây
    And câu hỏi đang xử lý trở lại hàng đợi

  # R-12, R-13, SEC-1
  Scenario: Secret chỉ tới MCP server
    Given agent "backend-dev" có quyền đọc "shop-db" dùng secret
    When "backend-dev" khởi động
    Then MCP server của "shop-db" nhận mật khẩu trong env
    And cấu hình, tham số và env của process agent không chứa mật khẩu

  # R-13
  Scenario: Token khởi động dùng lại bị từ chối
    Given một token khởi động đã được dùng
    When có tiến trình dùng lại token đó
    Then tiến trình bị từ chối và không nhận secret nào

  # R-13, SEC-3
  Scenario: Không lấy được secret
    Given Secret Manager không truy cập được
    When "backend-dev" khởi động
    Then "backend-dev" chạy không có tool của "shop-db"
    And web nhận event nguồn "shop-db" không lấy được secret

  # R-14
  Scenario: Secret không nằm trên đĩa
    When "backend-dev" khởi động với nguồn "shop-db"
    Then không file nào của orchestrator chứa giá trị secret

  # R-15
  Scenario: Truy cập nguồn hiện trên web
    When "backend-dev" gọi tool "query" của "shop-db"
    Then web nhận event gọi tool với kho "source:shop-db"
