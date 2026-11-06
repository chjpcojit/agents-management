# Source: ../requirements.md v1.0
Feature: Stream event lên web

  # R-1
  Scenario: Event có envelope đầy đủ
    Given agent "backend-dev" đang chạy task "viết test"
    When agent bắt đầu một turn
    Then web nhận event "turn.started" có id, phiên bản 0, thời điểm, agent và task

  # R-2
  Scenario: Type ngoài danh mục không được phát
    When một thành phần cố phát event loại "foo.bar"
    Then event bị từ chối và web không nhận gì

  # R-3
  Scenario: Thứ tự event của một agent được giữ
    Given agent "backend-dev" in ra lần lượt text, gọi tool, rồi kết thúc turn
    When web đang nghe kênh event
    Then web nhận "turn.text", "tool.requested", "turn.completed" theo đúng thứ tự đó

  # R-4
  Scenario: Text của agent không được lưu
    Given agent "backend-dev" viết một đoạn text
    When orchestrator khởi động lại
    Then lịch sử event đã lưu không chứa đoạn text đó

  # R-5
  Scenario: Event được lưu cùng thay đổi trạng thái
    When người duyệt cho phép một yêu cầu đang chờ
    Then trạng thái yêu cầu và event "approval.decided" cùng được lưu

  # R-6
  Scenario: Không lưu được event thì không đổi trạng thái
    Given kho event đang lỗi
    When người duyệt cho phép một yêu cầu đang chờ
    Then người duyệt nhận lỗi
    And yêu cầu vẫn đang chờ

  # R-7, R-8
  Scenario: Tool đọc file trong worktree
    When agent đọc file "internal/domain/approval.go"
    Then web nhận event "tool.requested" với resource kho "worktree", thao tác "read", đích "internal/domain/approval.go", không phải suy đoán

  # R-9
  Scenario Outline: Tool Bash được suy đoán resource
    When agent chạy lệnh Bash "<lệnh>"
    Then resource là kho "<kho>", thao tác "<thao tác>", có đánh dấu suy đoán

    Examples:
      | lệnh                       | kho     | thao tác |
      | git push origin HEAD       | github  | network  |
      | rm -rf build               | worktree| delete   |
      | curl https://example.com/x | web     | network  |
      | ./scripts/do-magic.sh      | unknown | exec     |

  # R-10
  Scenario: Đích không chứa giá trị nhạy cảm
    When agent chạy lệnh Bash "curl -H 'Authorization: Bearer 0000000000' https://api.example.com/v1?q=1"
    Then đích của resource là "api.example.com"
    And event không chứa "0000000000"

  # R-11, R-12
  Scenario: Lọc event theo task
    Given hai agent cùng chạy hai task khác nhau
    When web nghe kênh event lọc theo task "viết test"
    Then web chỉ nhận event của task "viết test"

  # R-13
  Scenario: Kết nối lại không hụt event
    Given web đã nhận tới event 120 rồi mất mạng
    And trong lúc đó có 5 event mới được phát
    When web kết nối lại từ event 120
    Then web nhận đủ 5 event theo đúng thứ tự rồi tiếp tục nhận event mới

  # R-13
  Scenario: Text bị hụt được báo
    Given web mất mạng đủ lâu để text của agent rơi khỏi bộ đệm
    When web kết nối lại
    Then web nhận event "stream.gap"

  # R-14
  Scenario: Heartbeat giữ kết nối
    Given không có event nào trong 30 giây
    When web đang nghe kênh event
    Then web nhận ít nhất 2 heartbeat

  # R-15
  Scenario: Client đọc chậm bị ngắt
    Given một client không đọc event
    When có hơn 256 event được phát
    Then kết nối của client đó bị đóng

  # R-16
  Scenario: Client bỏ qua field mới
    Given event "turn.completed" có thêm một field mới trong cùng phiên bản 0
    When web nhận event
    Then event vẫn hợp lệ theo schema phiên bản 0

  # R-17
  Scenario: Dòng stream lạ
    When process agent in ra một dòng có loại không biết
    Then web nhận event "stream.unknown" chỉ có độ dài dòng
