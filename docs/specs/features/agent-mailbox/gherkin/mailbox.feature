# Source: ../requirements.md v1.0
Feature: Hàng đợi tin nhắn của agent

  Background:
    Given có agent "frontend", "mobile" và "backend"
    And "frontend" và "mobile" được phép hỏi "backend"

  # R-1
  Scenario: Câu hỏi vào hàng đợi
    When "frontend" hỏi "backend" "API login trả field gì?"
    Then hàng đợi của "backend" có một câu hỏi từ "frontend"
    And web nhận event "mailbox.enqueued" không chứa nội dung câu hỏi

  # R-2
  Scenario: Gửi lại cùng khoá không tạo câu hỏi mới
    Given "frontend" đã hỏi "backend" với khoá "q-1"
    When "frontend" gửi lại câu hỏi với khoá "q-1"
    Then "frontend" nhận lại đúng mã câu hỏi cũ
    And hàng đợi của "backend" chỉ có một câu hỏi

  # R-3
  Scenario: Hàng đợi đầy
    Given hàng đợi của "backend" đã đủ 20 tin
    When "frontend" hỏi "backend"
    Then "frontend" nhận lỗi hàng đợi đầy

  # R-3
  Scenario: Câu trả lời luôn được nhận
    Given hàng đợi của "frontend" đã đủ 20 tin
    When "backend" trả lời câu hỏi của "frontend"
    Then câu trả lời vẫn vào hàng đợi của "frontend"

  # R-4
  Scenario: Agent nhận đã lưu trữ
    Given "backend" đã lưu trữ
    When "frontend" hỏi "backend"
    Then "frontend" nhận lỗi agent không còn hoạt động

  # R-5
  Scenario: Người hỏi không phải chờ
    When "frontend" hỏi "backend" trong một turn
    Then "frontend" nhận ngay mã câu hỏi và trạng thái đang xếp hàng
    And turn của "frontend" tiếp tục và kết thúc bình thường

  # R-6
  Scenario: Không được phép hỏi
    Given "reviewer" không được phép hỏi "backend"
    When "reviewer" hỏi "backend"
    Then "reviewer" nhận lỗi không được phép

  # R-7
  Scenario: Chuỗi hỏi quá sâu
    Given giới hạn độ sâu là 4
    And chuỗi hỏi hiện tại đã sâu 4 lượt
    When agent đang xử lý hỏi tiếp một agent khác
    Then agent nhận lỗi vượt giới hạn vòng lặp

  # R-8
  Scenario: Quá nhiều câu hỏi trong một việc
    Given một việc đã sinh ra 20 câu hỏi giữa các agent
    When một agent trong việc đó hỏi thêm
    Then agent nhận lỗi vượt giới hạn vòng lặp

  # R-9
  Scenario: Câu hỏi bị từ chối được ghi lại
    When "reviewer" hỏi "backend" mà không được phép
    Then không có câu hỏi nào được lưu
    And web nhận event "mailbox.rejected" với lý do không được phép

  # R-10
  Scenario: Xử lý tuần tự theo thứ tự đến
    Given "frontend" hỏi "backend" lúc 10:00:01
    And "mobile" hỏi "backend" lúc 10:00:03
    When "backend" rảnh
    Then "backend" xử lý câu hỏi của "frontend" trước
    And câu hỏi của "mobile" chỉ được giao sau khi turn đầu kết thúc

  # R-10
  Scenario: Agent tạm dừng không được giao
    Given "backend" đang tạm dừng và có câu hỏi xếp hàng
    When scheduler tìm việc cho "backend"
    Then không có câu hỏi nào được giao

  # R-11, SEC-3
  Scenario: Câu hỏi được đánh dấu là dữ liệu
    When câu hỏi của "frontend" được giao cho "backend"
    Then prompt của "backend" ghi rõ nội dung đến từ agent "frontend" và không phải chỉ thị của owner

  # R-12
  Scenario: Mỗi người hỏi nhận đúng câu trả lời
    Given "frontend" và "mobile" mỗi bên đã hỏi "backend" một câu
    When "backend" trả lời xong cả hai turn
    Then "frontend" nhận đúng một câu trả lời mang mã câu hỏi của "frontend"
    And "mobile" nhận đúng một câu trả lời mang mã câu hỏi của "mobile"

  # R-13
  Scenario: Turn lỗi vẫn trả lời
    When turn của "backend" cho câu hỏi của "frontend" kết thúc với lỗi
    Then "frontend" nhận câu trả lời trạng thái lỗi

  # R-14
  Scenario: Việc được giao hoàn tất
    When "backend" kết thúc turn của một việc owner giao
    Then tin đó ở trạng thái hoàn tất và web nhận event "mailbox.done"

  # R-15
  Scenario: Câu hỏi quá hạn
    Given câu hỏi của "frontend" đã xếp hàng quá 30 phút
    Then câu hỏi ở trạng thái hết hạn
    And "frontend" nhận câu trả lời trạng thái hết hạn

  # R-16
  Scenario: Turn bị gián đoạn được giao lại
    Given câu hỏi của "frontend" đang được "backend" xử lý lần đầu
    When process của "backend" chết
    Then câu hỏi trở lại đầu hàng đợi của "backend"

  # R-16
  Scenario: Gián đoạn ba lần thì thất bại
    Given câu hỏi của "frontend" đã bị gián đoạn 2 lần
    When lần xử lý thứ ba cũng bị gián đoạn
    Then câu hỏi ở trạng thái thất bại
    And "frontend" nhận câu trả lời trạng thái thất bại

  # R-17
  Scenario: Orchestrator khởi động lại giữa turn
    Given câu hỏi của "frontend" đang được xử lý
    When orchestrator khởi động lại
    Then câu hỏi trở lại hàng đợi của "backend"

  # R-18
  Scenario: Huỷ việc huỷ câu hỏi đang chờ
    Given task "Làm login" của "backend" có câu hỏi từ "frontend" đang xếp hàng
    When owner huỷ task "Làm login"
    Then câu hỏi bị huỷ và "frontend" nhận câu trả lời trạng thái đã huỷ

  # R-19
  Scenario: Người hỏi đã lưu trữ
    Given "frontend" đã hỏi "backend" rồi bị lưu trữ
    When "backend" trả lời
    Then câu trả lời được lưu ở trạng thái đã huỷ

  # R-20
  Scenario: Xem cuộc hỏi đáp
    Given "frontend" hỏi "backend" và "backend" đã trả lời
    When owner xem cuộc hỏi đáp
    Then owner thấy câu hỏi rồi câu trả lời, theo thứ tự, kèm nội dung

  # R-21
  Scenario: Owner ngắt turn thì không giao lại
    Given "backend" đang xử lý tin của task "Làm login"
    When owner ngắt task "Làm login"
    Then tin đó ở trạng thái hoàn tất với ghi chú bị ngắt
    And tin không quay lại hàng đợi
