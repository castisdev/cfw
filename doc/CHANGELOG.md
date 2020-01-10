v1.0.0.qr2 / 2020-01-10
===================
[형상변경]
  * file 조회 API 응답 내용 변경
    - base directory 에 하위 directory 정보는 포함시키지 않고, file만 보내도록 수정
  * file delete API 응답 추가
    - base direcotry 내의 하위 directroy 지우는 요청이 올 경우 409
    - 없는 파일 지우는 요청이 올 경우: 404
  * df api 결과 값 수정
	  - df 명령 결과 값과 같은 계산식을 사용하게 수정
  * cfm tasks API 변경 반영
    - reposne가 map 형태에서 list 형태로 바뀐 것 반영
  * 일부 로그 추가, 변경
  * 최신 로그 파일에 대한 symbol link 파일 생성
  * 설정 제거
    * [if_name] 제거되고, listen_addr 의 ip,port 값을 사용함
  * 설정 추가
    * [downloader_sleep_sec] 추가
    * [downloader_download_success_match_string] 추가

[개선]
  * heartbeat api 추가
  * port 충돌로 실행 실패했을 때 화면에 error 메시지 출력하고 종료하게 변경
  * 일부 내부 에러 처리 방식 개선 및 추가

[버그]
  * download 시에 사용하는 임시 directory의 권한 mode 설정 잘못된 것 수정

[참고]
  * API 응답 시에 header를 중복해서 write 하는 문제 원인 파악
    - https://github.com/golang/go/issues/18761
    - https://github.com/caddyserver/caddy/issues/2537
    - json.NewEncoder(w).Encode(&du) 호출이 성공하면 w.WriteHeader(http.StatusOK)를 호출할 필요없음
    - 관련된 comment 제거

v1.0.0.qr1 / 2019-11-13
===================
* 최초 릴리즈
