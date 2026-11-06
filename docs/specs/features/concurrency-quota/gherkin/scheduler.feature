# Source: ../requirements.md v1.0
Feature: Giới hạn chạy đồng thời và quota Max

  Background:
    Given giới hạn chạy đồng thời là 2

  # R-1
  Scenario: Không vượt giới hạn
    Given 4 agent mỗi agent có một việc xếp hàng
    When scheduler giao việc
    Then đúng 2 agent chạy turn và 2 agent chờ

  # R-2
  Scenario: Việc đến trước được chạy trước
    Given đang có 2 turn chạy
    And "reviewer" có việc từ 10:00 và "mobile" có việc từ 10:05
    When một turn kết thúc
    Then "reviewer" được chạy tiếp theo

  # R-3
  Scenario: Việc mới được giao ngay khi có chỗ
    Given chưa có turn nào chạy
    When owner giao một việc cho "backend-dev"
    Then "backend-dev" bắt đầu turn trong vòng 5 giây

  # R-4
  Scenario: Agent lỗi bị bỏ qua
    Given "backend-dev" đang ở trạng thái lỗi và có việc xếp hàng
    When scheduler giao việc
    Then "backend-dev" không được giao và việc vẫn xếp hàng

  # R-5
  Scenario: Đọc quota định kỳ
    When tới kỳ đọc quota
    Then web nhận event "quota.updated" với phần trăm phiên 5 giờ và tuần

  # R-6
  Scenario: Gần hết quota thì ngừng giao việc
    Given đang có 1 turn chạy
    When quota phiên 5 giờ báo 92%
    Then scheduler ngừng giao việc mới với lý do quota
    And turn đang chạy vẫn chạy tới hết

  # R-6
  Scenario: Quota hồi lại thì giao tiếp
    Given scheduler đang dừng vì quota
    When quota báo 10% sau khi reset
    Then scheduler giao việc trở lại

  # R-7
  Scenario: Không đọc được quota
    When output quota có định dạng lạ
    Then web nhận event quota trạng thái không rõ
    And scheduler vẫn giao việc

  # R-8
  @wip
  Scenario: Bị giới hạn tốc độ
    When một turn báo bị giới hạn tốc độ của gói
    Then scheduler ngừng giao việc mới trong 5 phút

  # R-9
  Scenario: Owner dừng tay
    Given owner đã dừng scheduler
    When quota báo dưới ngưỡng
    Then scheduler vẫn dừng tới khi owner cho chạy lại

  # R-10
  Scenario: Xem scheduler
    When owner xem trạng thái scheduler
    Then owner thấy giới hạn, turn đang chạy, số việc chờ theo agent và quota mới nhất

  # R-11
  Scenario: Shutdown không giao thêm
    Given có việc xếp hàng
    When owner dừng orchestrator
    Then không có turn mới nào được bắt đầu
