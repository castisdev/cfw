# API

## HEAD /hb
- heartbeat 요청
- Response:
  - 200 OK
- curl 사용 예:
```bash
    $ curl -I 127.0.0.1:9888/hb
```
- httpie 사용 예:
```bash
    $ http HEAD 127.0.0.1:9888/hb
```

## GET /files
- base directory의 파일 목록 조회
    - base directory는 cfm.yml 의 base_dir 설정을 참고함
    - subdirectory 는 목록에 포함되지 않음
- Response
  - 200 OK
  - 500 Internal Server Error

```text
file1.mpg
file2.mpg
file3.mpg
```
- curl 사용 예:
```bash
    $ curl 127.0.0.1:9888/files
```
- httpie 사용 예:
```bash
    $ http 127.0.0.1:9888/files
```

## GET /df
- base directory의 disk free 조회
    - base directory는 cfm.yml 의 base_dir 설정을 참고함
- Response
  - 200 OK
  - 500 Internal Server Error

```json
{"total_size":"314913513472","used_size":"200937054208","avail_size":"97956167680","used_percent":67,"free_size":"113976459264"}

```
- curl 사용 예:
```bash
    $ curl 127.0.0.1:9888/df
```
- httpie 사용 예:
```bash
    $ http 127.0.0.1:9888/df
```

## DELETE /files/{filename}
- base directory의 파일 삭제
    - base directory는 cfm.yml 의 base_dir 설정을 참고함
- Response
  - 200 OK
  - 400 Bad Request
  - 404 Not Found
  - 406 Not Acceptable
  - 500 Internal Server Error

- curl 사용 예:
```bash
    curl -X DELETE 127.0.0.1:9888/files/file1.mpg
```
- httpie 사용 예:
```bash
    $ http DELETE 127.0.0.1:9888/files/file1.mpg
```
