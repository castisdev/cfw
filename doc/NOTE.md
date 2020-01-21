## 2020-01-21
- 메모리 누수 방지 위한 코드 추가, keep alive 설정 시에 문제 방지 코드 추가
  - // https://stackoverflow.com/questions/33229860/go-http-requests-json-reusing-connections
   - // https: //stackoverflow.com/questions/17959732/why-is-go-https-client-not-reusing-connections


## 2020-01-09
DELETE API:
변경:
directroy 지우는 기능 제거, 406 return 으로 변경
	StatusNotAcceptable                = 406 // RFC 7231, 6.5.6
설정에서 if_name 제거


## 2019-12-13

addr/files API :
하위 directory의 file 은 보내지 않음
변경 :
directory 정보는 제거하고, file만 보내도록 수정

DELETE API:
변경:
없는 파일 지우는 경우, 204 return
	StatusNoContent            = 204 // RFC 7231, 6.3.5
directroy 지우는 기능 제거, 409 return
	StatusConflict                     = 409 // RFC 7231, 6.5.8

## 2019-12-12

task 찾을 때,

기존 조건:
config 의 if_name을 이용해서, myIp로 설정하고
task의 DstIP 와 myIP가 같은 task 를 찾음

IP 만 사용

수정 조건:
config 의 listen_addr 값을 myAddr 로 설정하고
task.DstAddr 와 myAddr이 같은 task를 찾음
IP:Port 조합 사용
