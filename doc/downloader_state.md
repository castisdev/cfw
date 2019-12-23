```plantuml
hide empty description

[*]--down-> S0
S0 --> S1
S1: disk usage percent 구함
S1 --down-> S2 : error
S2 : sleep 3초
S2 --up-> S0

S1 --> S3 : ok
S3 : disk usage limit percent 와 검사

S3 --> S2 : limit 넘는 경우
S3 --> S5 : limit 넘지 않은 경우
S5 : task 구함
note right
  task 가 있을 때까지 계속 시도
end note
S5 --> S6
S6 : task download 수행
S6 --> S7 : error 발생
S7: error log 남김

S6 --> S8 : ok
S7 --> S8
S8 : task state DONE report
S8 --> S9 : error 발생
S9: warning log 남김
note right
  task state를 DONE으로 바꾸지 못하면,
  cfm 에서 TIMEOUT 처리할 때까지 지워지지 않음
end note

S9 --> S0
S8 --> S0 : ok

```

```plantuml
hide empty description

state S5 {
S5 : task 구함

[*]--> S50
S50 --> S51
S51 : task 목록 요청
S51 --> S52: error (logging)
S52 : sleep 5초
S52 --> S50
S51 --> S53: ok
S53 : 응답 json 에서 task 목록 생성
S53 --> S52: error (logging)

S53 --> S54: ok
S54 : dest가 나의 ip와 같은 READY task가 있는 지 검사
S54 --> S55 : task가 있으면
S55 : task state WORKING report
note right
  WORKING report 실패하면
  task 상태에 대한 synch 가 깨져서
  문제 발생하지 않을 지?
    정상적인 경우에는 WORKING -> DONE 으로 바뀌는데 비홰서
    이 경우에는 READY -> DONE 으로 바뀔 듯
end note

S55 --> [*] : error (logging)
S55 --> [*] : ok
note right
  task가 있으면 loop를 빠져나와서
  S6 으로 return 됨
end note

S54 --> S52 : task가 없으면 (logging)
}
```

```plantuml
hide empty description
state S6 {
S6 : download 수행

[*]--> S60
S60 : make target dir
S60 --> [*] : error (return error)
S60 --> S61
S61 : download 수행
S61 --> [*] : error (return error)
S61 --> S64 : success
S64 : rename
S64 --> [*] : error (return error)



}
```
