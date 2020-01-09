v1.0.0.qr2 / 2020-01-09
===================
### cfm, cfw, DFS Sequence

- 배포 schedule과 task는 같은 의미로 사용됨
- cfw 와 heartbeat 추가
- 여러 서버에 배포된 파일을 한 서버에만 남기고 나머지에서는 제거하는 기능 추가

```plantuml

box "ADC/LSM"
participant cfm
end box

box "VOD"
participant cfw as "cfw"
end box

box "VOD(source)"
participant cfw2 as "cfw"
participant dfs as "DFS"
end box

group heartbeat-thread
loop endless
autonumber 1 "<b>[0]"
cfm->cfw: heartbeat 확인
    note right
      서버별 heartbeat 성공 여부 정보 생성
    end note
cfm->cfw2: heartbeat 확인
end
end

group file-remover-thread
loop endless
autonumber 1 "<b>[0]"
cfm->cfm: file meta 정보 생성
    note right
        - .hitcount.history 에서 file size 추출
        - .grade.info 에서 file 순위 추출
    end note
cfm->cfw: GET /files
    note right
        서버별 file list 확인
    end note
cfm->cfw2: GET /files
cfm->cfm: file meta 에서 여러 서버에 존재하는(배포된) file list 추출
cfm->cfw: DELETE /files/c.mpg
    note right
        여러 서버에 존재하는 file 삭제
    end note

cfm->cfw: GET /df
    note right
        디스크 사용량 체크
    end note
cfm->cfw2: GET /df
cfm->cfw: DELETE /files/a.mpg
    note right
        디스크 사용량이 정해진 한계 사용량을 넘었을 때
        file 순위가 낮은 순으로 file 삭제
    end note
cfm->cfw2: DELETE /files/b.mpg
end
end

group task-manager-thread
loop endless
autonumber 1 "<b>[0]"
cfm->cfm: file meta 정보 생성
    note right
        - .hitcount.history 에서 file size 추출
        - .grade.info 에서 file 순위 추출
    end note
cfm->cfm: check task queue
    note right
        배포 schedule 조회
    end note
cfm->cfm: clean task queue
    note right
        schedule 삭제
        - 완료되었거나 (status = done)
        - timeout ( mtime + 30분 =< current time )
        - heartbeat 성공 하지 않았거나
    end note
cfm->cfw: GET /files
    note right
        서버별 file list 확인
    end note
cfm->cfw2: GET /files
cfm->cfm: create tasks
    note right
        배포 schedule 생성
        - 순위가 높은 순서부터 조건 확인
        - task queue 에 없고
        - VOD 서버에 없고
        - SAN 에 존재하는 파일인 경우 선택
        - 모든 src 가 선택될 때까지 생성
    end note
end
end

group cfw
loop endless
autonumber 1 "<b>[0]"
cfw->cfw: 디스크 사용량 체크
cfw->cfm: GET /tasks
    note left
        디스크가 충분한 경우
        배포 schedule 조회
        자신의 배포 schedule 선택
    end note
cfw->cfm: PATCH /tasks/${task_id}
    note left
        file 다운로드 시작 전 schedule 상태 변경
        - status:ready -> status:working
    end note
cfw->dfs: file 다운로드 요청
cfw->cfm: PATCH /tasks/${task_id}
    note left
        file 다운로드 완료 후 schedule 상태 변경
        - status:working -> status:done
    end note
end
end
```

### cfw downloader State

```plantuml
hide empty description

[*]-> S0
S0 --> S1 : 무한 루프
S1 : task 구함
note right
  cmf와 통신해서
  자신의 task를 구할 때까지 시도
end note
S1 --> S2 : task
S2 : file download 수행
  note right
    DFS downloader 등
    외부 download 명령어 이용하여
    task의 source로부터 file download 수행
  end note
S2 --> S3 : error
S3: error logging
  note right
    download 중 error 가 발생해도
    status는 DONE으로 바꾼다.
  end note
S2 --> S4 : ok\n(logging)
S3 --> S4
S4 : task status DONE report
S4 --> S5 : error
S5: warning logging
note right
  task status를 DONE으로 바꾸지 못하면,
  cfm에서 TIMEOUT 처리할 때까지
  해당 task는 지워지지 않음
end note

S5 --> S0
S4 --> S0 : ok\n(logging)

```

```plantuml
hide empty description

state S1 {

S1 : task 구함
[*] -> S10
S10 --> S11 : 무한 루프
S11: disk usage percent 구함
S11 -> S12 : error
S12 : sleep N초
S12 -> S10

S11 --> S13 : ok
S13 : disk usage limit percent 와 검사

S13 --> S12 : limit 넘는 경우
S13 --> S14 : limit 넘지 않은 경우

S14 : cfm에 task 목록 요청
S14 --> S12: error\n(logging)
S14 --> S15: ok
S15 : 응답 json 에서 task 목록 생성
S15 --> S12: error \n(logging)

S15 --> S16: ok
S16 : dest가 나의 ip와 같은 READY task가 있는 지 검사
S16 --> S17 : task가 있으면
S17 : task status WORKING report
note right
  WORKING report 실패하면
  task 상태에 대한 synch 가 깨져서
  문제 발생하지 않을 지?
    정상적인 경우에는 READY -> WORKING -> DONE 으로 바뀌는데 비해서
    이 경우에는 READY -> DONE 으로 바뀌게 됨
end note

S17 --> S2 : error\n(warning logging)
S17 --> S2 : ok\n(logging)
note right
  loop를 빠져나와서
  S2 으로 return 됨
end note

S16 --> S12 : task가 없으면
}
```

```plantuml
hide empty description
state S2 {
S2 : file download 수행

[*]--> S20
S20 : target direcotry에 임시 directory 생성
S20 --> S3 : error
S20 --> S21 : ok
S21 : 임시 directory에 file download 수행
S21 --> S3 : error
S21 --> S23 : ok
S23 : rename : 임시 directory의 file을\ntarget directory로 이동
S23 --> S3 : error
  note left
  return error
  S3로 이동
  end note
S23 --> S4 : ok
  note right
  return error
  S4로 이동
  end note
}
```
