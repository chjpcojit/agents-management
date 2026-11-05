# Source: ../requirements.md v1.0
Feature: Giao việc và theo dõi

  Background:
    Given project "shop" sẵn sàng và agent "backend-dev" hoạt động

  # R-1
  Scenario: Giao việc
    When owner giao task "Viết test cho approval" cho "backend-dev" trên "shop"
    Then task được tạo ở trạng thái xếp hàng với worktree riêng
    And hàng đợi của "backend-dev" có tin của task

  # R-2
  Scenario: Gửi lại cùng khoá
    Given owner đã giao task với khoá "t-1"
    When owner gửi lại yêu cầu với khoá "t-1"
    Then owner nhận lại task cũ và không có worktree mới

  # R-3
  Scenario: Project chưa sẵn sàng
    Given project "shop" đang clone
    When owner giao task cho "backend-dev" trên "shop"
    Then owner nhận lỗi project chưa sẵn sàng

  # R-4
  Scenario: Hàng đợi đầy sau khi đã tạo worktree
    Given hàng đợi của "backend-dev" đã đầy
    When owner giao task "Sửa lỗi" cho "backend-dev"
    Then owner nhận lỗi hàng đợi đầy
    And worktree vừa tạo vẫn còn trên đĩa

  # R-5
  Scenario: Task chạy khi tin được giao
    When tin của task được giao cho "backend-dev"
    Then task ở trạng thái đang chạy

  # R-6
  Scenario: Task rảnh khi turn xong
    Given task đang chạy turn đầu tiên
    When turn kết thúc với chi phí 0.12 USD
    Then task ở trạng thái rảnh với chi phí cộng dồn 0.12 USD

  # R-7
  Scenario: Task lỗi khi tin thất bại
    When tin của task thất bại sau 3 lần
    Then task ở trạng thái lỗi

  # R-8
  Scenario: Gửi thêm tin trong task
    Given task ở trạng thái rảnh
    When owner nhắn "chỉ test use case thôi"
    Then tin được xếp vào hàng đợi của "backend-dev" cho task đó

  # R-8
  Scenario: Task đã đóng
    Given task đã hoàn tất
    When owner nhắn thêm vào task
    Then owner nhận lỗi task đã đóng

  # R-9
  Scenario: Ngắt rồi làm tiếp
    Given task đang chạy
    When owner ngắt task
    Then trong vòng 5 giây task ở trạng thái rảnh
    And khi owner nhắn tin tiếp, agent làm tiếp trong cùng phiên hội thoại

  # R-10
  Scenario: Huỷ task
    Given task đang chạy và có 1 tin xếp hàng
    When owner huỷ task
    Then task ở trạng thái đã huỷ, tin xếp hàng bị huỷ, worktree vẫn còn

  # R-11
  Scenario: Đóng task
    Given task ở trạng thái rảnh
    When owner đánh dấu task hoàn tất
    Then task ở trạng thái hoàn tất

  # R-12
  Scenario: Danh sách không lộ prompt
    When owner xem danh sách task của "backend-dev"
    Then danh sách có tiêu đề và trạng thái nhưng không có prompt

  # R-13
  Scenario: Xem task chạy realtime
    Given owner đang xem task trên web
    When agent gọi tool trong turn của task
    Then web nhận event gọi tool của đúng task đó
