# Source: ../requirements.md v1.1
@wip
Feature: Chạy agent qua claude -p

  # R-1
  Scenario: Khởi động agent với login của owner
    Given owner đã đăng nhập Claude bằng gói Max trên máy
    When orchestrator khởi động agent "agent-a"
    Then agent "agent-a" chuyển sang trạng thái rảnh

  # R-2
  Scenario: API key trong môi trường không được dùng
    Given môi trường của orchestrator có một API key giả
    When orchestrator khởi động agent "agent-a"
    Then agent "agent-a" chạy bằng login gói Max của owner

  # R-3
  Scenario: Auth lỗi làm agent thất bại
    Given owner chưa đăng nhập Claude
    And agent "agent-a" đang rảnh
    When người dùng giao việc đầu tiên cho agent "agent-a"
    Then agent "agent-a" ở trạng thái thất bại với lý do xác thực

  # R-3
  Scenario: Agent im lặng sau khi nhận việc
    Given agent "agent-a" đang rảnh
    When người dùng giao một việc và agent không trả về gì trong 90 giây
    Then agent "agent-a" ở trạng thái thất bại với lý do không có phản hồi

  # R-4
  Scenario: Giao việc cho agent rảnh
    Given agent "agent-a" đang rảnh
    When người dùng giao "liệt kê file trong thư mục"
    Then agent "agent-a" chuyển sang đang chạy

  # R-5
  Scenario: Thấy tiến độ realtime
    Given agent "agent-a" đang chạy một việc
    When agent gọi một tool
    Then người dùng thấy lời gọi tool trước khi việc kết thúc

  # R-6
  Scenario: Dòng output lạ không làm dừng stream
    Given agent "agent-a" đang chạy một việc
    When agent xuất một dòng không đọc được
    Then người dùng vẫn tiếp tục nhận tiến độ

  # R-7
  Scenario: Kết thúc lượt
    Given agent "agent-a" đang chạy một việc
    When việc kết thúc
    Then agent "agent-a" chuyển về rảnh
    And chi phí của lượt được ghi nhận

  # R-8
  Scenario: Giao việc khi agent đang bận
    Given agent "agent-a" đang chạy một việc
    When người dùng giao thêm một việc
    Then việc mới được xếp hàng cho "agent-a"

  # R-9
  Scenario: Agent khởi động với kênh duyệt của orchestrator
    When orchestrator khởi động agent "agent-a"
    Then mọi yêu cầu quyền của "agent-a" đi vào inbox duyệt

  # R-10
  Scenario Outline: Kết quả duyệt trả về cho agent
    Given agent "agent-a" xin chạy "touch x"
    When yêu cầu được <quyết định>
    Then agent <kết quả>

    Examples:
      | quyết định  | kết quả                     |
      | cho phép    | chạy lệnh và tạo file x     |
      | từ chối     | không chạy lệnh, nhận lý do |
      | để hết hạn  | không chạy lệnh             |

  # R-11
  Scenario: Kênh duyệt chưa sẵn sàng
    Given kênh duyệt của orchestrator chưa sẵn sàng
    When orchestrator khởi động agent "agent-a"
    Then agent "agent-a" không được khởi động

  # R-12
  Scenario: Hủy việc đang chạy
    Given agent "agent-a" đang chạy lệnh "sleep 60"
    When người dùng hủy việc
    Then agent "agent-a" chuyển về rảnh
    And agent vẫn nhận việc mới được

  # R-13
  Scenario: Agent chết và khôi phục
    Given agent "agent-a" đã làm một việc trước đó
    When process của agent bị dừng đột ngột
    And người dùng khởi động lại agent
    Then agent nhớ việc đã làm trước đó

  # R-14
  Scenario: Orchestrator tắt êm
    Given hai agent đang rảnh
    When orchestrator tắt
    Then cả hai agent dừng trong thời hạn tắt

  # R-15
  Scenario: Nén ngữ cảnh
    Given agent "agent-a" đã làm vài việc
    When người dùng gửi lệnh nén ngữ cảnh
    Then người dùng thấy số token trước khi nén

  # SEC-4
  Scenario: Agent không giả danh được agent khác
    Given agent "agent-a" và agent "agent-b" cùng chạy
    When agent "agent-a" gọi công cụ duyệt bằng địa chỉ của agent "agent-b"
    Then yêu cầu bị từ chối và không có yêu cầu duyệt nào được mở cho agent "agent-b"
