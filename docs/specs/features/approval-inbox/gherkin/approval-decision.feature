# Source: ../requirements.md v1.1
Feature: Quyết định yêu cầu duyệt

  Background:
    Given thời gian duyệt tối đa là 1 phút

  # R-1
  Scenario: Agent mở yêu cầu duyệt hợp lệ
    Given agent "agent-a" đang chạy
    When agent "agent-a" xin chạy tool "Bash" với lệnh "touch x" mức rủi ro "high"
    Then inbox có một yêu cầu đang chờ duyệt của "agent-a"
    And hạn duyệt là 1 phút sau thời điểm tạo

  # R-2
  Scenario Outline: Yêu cầu không hợp lệ bị từ chối
    When một yêu cầu duyệt được gửi với <vấn đề>
    Then yêu cầu bị từ chối vì không hợp lệ
    And inbox không có yêu cầu mới

    Examples:
      | vấn đề                       |
      | thiếu agent                  |
      | loại không phải tool hay ask |
      | loại tool nhưng thiếu tên    |
      | mức rủi ro không xác định    |
      | input không phải JSON        |

  # R-4
  Scenario: Duyệt trước hạn
    Given có yêu cầu đang chờ duyệt tạo 10 giây trước
    When người duyệt "duc" cho phép yêu cầu
    Then yêu cầu ở trạng thái đã cho phép
    And người duyệt được ghi là "duc"

  # R-5
  Scenario: Cho phép sau hạn vẫn bị từ chối
    Given có yêu cầu đang chờ duyệt tạo 2 phút trước
    When người duyệt "duc" cho phép yêu cầu
    Then người duyệt nhận lỗi quyết định quá hạn
    And yêu cầu ở trạng thái hết hạn
    And agent không được chạy tool

  # R-6
  Scenario: Không đổi được quyết định đã có
    Given yêu cầu đã bị từ chối
    When người duyệt "duc" cho phép yêu cầu
    Then người duyệt nhận lỗi yêu cầu đã được quyết định
    And yêu cầu vẫn ở trạng thái đã từ chối

  # R-7
  Scenario: Hai quyết định cùng lúc chỉ một được nhận
    Given có yêu cầu đang chờ duyệt
    When 50 quyết định cho phép và từ chối đến cùng lúc
    Then đúng một quyết định được ghi nhận

  # R-8
  Scenario: Quyết định thiếu người duyệt
    Given có yêu cầu đang chờ duyệt
    When một quyết định cho phép được gửi mà không có người duyệt
    Then quyết định bị từ chối vì không hợp lệ
    And yêu cầu vẫn đang chờ duyệt

  # R-9
  Scenario: Yêu cầu không tồn tại
    When người duyệt "duc" cho phép yêu cầu "nope"
    Then người duyệt nhận lỗi không tìm thấy

  # R-10
  Scenario: Hết hạn tự động thành từ chối
    Given có yêu cầu đang chờ duyệt tạo 61 giây trước
    And có yêu cầu đang chờ duyệt tạo 31 giây trước
    When lượt quét hết hạn chạy
    Then yêu cầu tạo 61 giây trước ở trạng thái hết hạn
    And yêu cầu tạo 31 giây trước vẫn đang chờ duyệt

  # R-11
  @wip
  Scenario: Một bản ghi lỗi không làm dừng lượt quét
    Given hai yêu cầu đã quá hạn
    And kho lưu trữ lỗi khi cập nhật yêu cầu thứ nhất
    When lượt quét hết hạn chạy
    Then yêu cầu thứ hai ở trạng thái hết hạn
    And lỗi của yêu cầu thứ nhất được ghi log

  # R-12
  @wip
  Scenario: Inbox liệt kê yêu cầu đang chờ, hạn gần nhất trước
    Given "agent-a" có yêu cầu hết hạn sau 10 giây
    And "agent-b" có yêu cầu hết hạn sau 50 giây
    When người dùng mở inbox
    Then inbox hiển thị yêu cầu của "agent-a" trước "agent-b"

  # R-13
  @wip
  Scenario: Yêu cầu rủi ro cao gửi thông báo
    When agent "agent-a" xin chạy tool "Bash" mức rủi ro "high"
    Then thiết bị của người dùng nhận thông báo duyệt

  # R-14
  Scenario: Xem chi tiết trước khi duyệt
    Given agent "agent-a" xin chạy tool "Bash" với lệnh "touch x"
    When người dùng mở chi tiết yêu cầu
    Then người dùng thấy lệnh "touch x"

  # R-15
  Scenario Outline: Kết quả gửi quyết định
    Given <tình huống>
    When người dùng gửi quyết định cho phép
    Then người dùng nhận <kết quả>

    Examples:
      | tình huống                        | kết quả                     |
      | yêu cầu đang chờ duyệt            | yêu cầu đã cho phép         |
      | yêu cầu không tồn tại             | lỗi không tìm thấy          |
      | yêu cầu đã bị từ chối             | lỗi đã được quyết định      |
      | yêu cầu đã quá hạn                | lỗi quá hạn                 |

  # R-16
  @wip
  Scenario: Yêu cầu mới hiện ngay không cần tải lại
    Given người dùng đang mở inbox
    When agent "agent-a" xin chạy tool "Bash"
    Then trong vòng 1 giây inbox hiện yêu cầu của "agent-a"

  # R-17
  @wip
  Scenario: Kết nối lại không mất event
    Given người dùng đã nhận event đến số 10
    And mất kết nối trong lúc có 3 yêu cầu mới
    When người dùng kết nối lại từ event số 10
    Then người dùng nhận đủ 3 yêu cầu mới

  # R-17
  @wip
  Scenario: Kết nối lại quá trễ
    Given event số 10 đã rơi khỏi bộ đệm
    When người dùng kết nối lại từ event số 10
    Then người dùng được yêu cầu tải lại toàn bộ inbox

  # R-18
  @wip
  Scenario: Kết nối rảnh vẫn được giữ
    Given người dùng đang mở inbox và không có yêu cầu mới trong 60 giây
    Then kết nối vẫn mở
