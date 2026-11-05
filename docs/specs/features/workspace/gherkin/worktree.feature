# Source: ../requirements.md v1.0
Feature: Project và worktree theo task

  # R-1
  Scenario: Workspace root nằm trong repo khác
    Given thư mục workspace được đặt bên trong một git repo
    When orchestrator khởi động
    Then orchestrator từ chối khởi động

  # R-2
  Scenario: Thêm project
    When owner thêm project "shop" với repo "git@github.com:example/shop.git"
    Then project "shop" ở trạng thái đang clone
    And sau khi clone xong project "shop" sẵn sàng với nhánh mặc định "main"

  # R-2
  Scenario: Clone thất bại
    Given repo không tồn tại
    When owner thêm project "ghost"
    Then project "ghost" ở trạng thái lỗi kèm lý do từ git

  # R-3
  Scenario Outline: URL repo không được phép
    When owner thêm project với repo "<url>"
    Then owner nhận lỗi không hợp lệ

    Examples:
      | url                                      |
      | file:///etc                              |
      | ext::sh -c touch% /tmp/x                 |
      | https://user:0000000000@github.com/a/b   |

  # R-4
  Scenario: Thử clone lại
    Given project "shop" đang lỗi
    When owner thử lại project "shop"
    Then project "shop" được clone lại

  # R-5
  Scenario: Dùng credential git của owner
    Given máy owner có SSH key truy cập repo private
    When owner thêm project từ repo private
    Then project sẵn sàng mà orchestrator không lưu credential nào

  # R-6
  Scenario: Tên worktree có ngày
    Given hôm nay là 2026-11-05 theo giờ máy
    When task "Làm login" được giao cho "agent-a" trên project "shop"
    Then worktree và nhánh tên "20261105-agent-a-lam-login" được tạo từ nhánh mặc định mới nhất

  # R-7
  Scenario: Trùng tên trong ngày
    Given đã có worktree "20261105-agent-a-lam-login"
    When một task "Làm login" khác được giao cho "agent-a" cùng ngày
    Then worktree mới tên "20261105-agent-a-lam-login-2"

  # R-8
  Scenario: Không fetch được vẫn tạo worktree
    Given mạng tới origin bị lỗi
    When task "Làm login" được giao cho "agent-a"
    Then worktree vẫn được tạo từ nhánh mặc định local
    And event tạo worktree ghi chưa fetch được

  # R-9
  Scenario: Hai task cùng lúc trên một project
    When hai task được giao cùng lúc cho "agent-a" và "agent-b" trên project "shop"
    Then cả hai worktree được tạo, không lỗi khoá git

  # R-10
  Scenario: Worktree được ghi lại
    When worktree của task "Làm login" được tạo
    Then web nhận event "worktree.created" có tên, đường dẫn và commit gốc

  # R-11
  Scenario: Không tự dọn
    Given task "Làm login" đã xong
    Then worktree và nhánh của task vẫn còn

  # R-12
  Scenario: Push phải qua duyệt
    When agent chạy "git push origin HEAD" trong worktree
    Then yêu cầu chạy lệnh vào hộp duyệt

  # R-13
  Scenario: Xem worktree cũ
    Given project "shop" có worktree tạo ngày 2026-11-01 và 2026-11-05
    When owner xem worktree của "shop"
    Then owner thấy cả hai cùng ngày tạo và task tương ứng
