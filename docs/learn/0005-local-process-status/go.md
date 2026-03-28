# Go Programming

## Non-blocking Channel Select

Go의 `select`문에 `default` case를 추가하면 non-blocking 동작이 된다.
이 패턴으로 channel의 상태를 즉시 확인할 수 있다.

```go
select {
case <-entry.done:
    // channel이 닫혀있음 → 프로세스 종료
    return common.Exited(entry.cmd.ProcessState.ExitCode()), nil
default:
    // channel이 아직 열려있음 → 프로세스 실행 중
    return common.Running(), nil
}
```

### Blocking vs Non-blocking

| 패턴 | 동작 |
|------|------|
| `<-ch` | channel에서 값을 받을 때까지 블로킹 |
| `select { case <-ch: ... default: ... }` | 즉시 확인 후 반환 (non-blocking) |

`done chan struct{}`가 `close()`되면 `<-done`은 즉시 zero value를 반환한다.
이 특성 덕분에 여러 goroutine이 동시에 종료를 감지할 수 있다.

## cmd.ProcessState와 ExitCode

`exec.Cmd`의 `Wait()` 호출이 완료되면 `cmd.ProcessState` 필드가 채워진다.
이 필드는 `Wait()` 전에는 `nil`이다.

```go
// Wait() 완료 후에만 접근 가능
exitCode := entry.cmd.ProcessState.ExitCode()
```

### ExitCode() 반환값

| 값 | 의미 |
|----|------|
| `0` | 정상 종료 |
| `1-255` | 비정상 종료 (프로그램이 반환한 코드) |
| `-1` | signal에 의해 종료되었거나 아직 종료되지 않음 |

현재 구현에서는 `done` channel이 닫힌 후에만 `ExitCode()`를 호출하므로
`ProcessState`가 `nil`인 경우는 발생하지 않는다.
reaper goroutine이 `Wait()`를 호출한 후 `close(done)`을 실행하기 때문이다.

## Go 테스트 유틸리티와 패턴

### t.Cleanup — 리소스 정리 보장

`t.Cleanup()`은 테스트 함수가 완료된 후(패닉 포함) 실행되는 정리 함수를 등록한다.
`defer`와 비슷하지만 테스트 프레임워크에 의해 관리되어 서브테스트에서도 올바르게 동작한다.

```go
func TestCreateSuccess(t *testing.T) {
    id, err := lp.Create(ctx, spec)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    t.Cleanup(func() { _ = lp.Destroy(ctx, id) })

    // 이후 assertion — 실패해도 Cleanup이 반드시 실행됨
    if id.IsZero() {
        t.Error("expected non-zero KernelID")
    }
}
```

#### t.Cleanup vs defer vs 수동 정리

| 방식 | 테스트 실패 시 정리 | 서브테스트 scope | 패닉 시 정리 |
|------|:---:|:---:|:---:|
| 함수 끝에 수동 호출 | X (Fatal로 중단되면 실행 안 됨) | 수동 관리 | X |
| `defer` | O | 현재 함수에만 | O |
| `t.Cleanup` | O | 서브테스트별 독립 scope | O |

`t.Fatalf()`는 `runtime.Goexit()`를 호출하여 테스트를 즉시 중단시킨다.
이 경우 `defer`는 실행되지만, 함수 끝에 있는 수동 정리 코드는 도달하지 못한다.
`t.Cleanup`은 `defer`와 동일하게 보장되면서 서브테스트에서 scope가 더 명확하다.

#### 여러 리소스 정리

```go
t.Cleanup(func() {
    _ = lp.Destroy(ctx, id1)
    _ = lp.Destroy(ctx, id2)
})
```

`t.Cleanup`은 여러 번 호출 가능하며, LIFO(후입선출) 순서로 실행된다.
리소스별로 분리 등록해도 되고, 하나의 클로저에 묶어도 된다.

### t.Helper — 헬퍼 함수의 에러 위치 보정

테스트 헬퍼 함수에서 `t.Helper()`를 호출하면, 에러 발생 시 보고 위치가
헬퍼 내부가 아닌 **헬퍼를 호출한 테스트 코드**를 가리킨다.

```go
func waitForExited(t *testing.T, lp *LocalProcess, ctx context.Context, id common.KernelID) common.KernelStatus {
    t.Helper() // 이 함수 내 t.Fatal이 호출자 라인을 보고함
    deadline := time.Now().Add(2 * time.Second)
    for time.Now().Before(deadline) {
        status, err := lp.Status(ctx, id)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if status.Type == common.StatusExited {
            return status
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatal("timeout waiting for process to exit")
    return common.KernelStatus{}
}
```

`t.Helper()` 없이 `t.Fatal`이 호출되면 "helper.go:15"처럼 헬퍼 내부 라인이 보고되어
실제 실패한 테스트를 찾기 어렵다. `t.Helper()`를 추가하면 "my_test.go:42"처럼
호출한 테스트의 라인이 보고된다.

### 폴링 기반 대기 — time.Sleep 대신

프로세스 종료 같은 비동기 이벤트를 테스트할 때 `time.Sleep`은 위험하다:
- CI 환경(부하 높음)에서 시간이 부족하면 flaky test 발생
- 시간을 넉넉히 잡으면 테스트 실행 시간이 불필요하게 길어짐

폴링 루프로 대체하면 두 문제를 모두 해결한다:

```go
// Anti-pattern ❌
time.Sleep(100 * time.Millisecond)
status, _ := lp.Status(ctx, id)

// Polling pattern ✅
status := waitForExited(t, lp, ctx, id)  // 조건 충족 시 즉시 반환, 타임아웃으로 hang 방지
```

#### 폴링 헬퍼 작성 시 핵심 요소

| 요소 | 설명 |
|------|------|
| `t.Helper()` | 에러 위치 보정 |
| deadline/timeout | 무한 대기 방지 |
| 짧은 polling interval (10~50ms) | 반응 속도와 CPU 사용 균형 |
| `t.Fatal` on timeout | 실패 원인 명확히 보고 |

### 불필요한 time.Sleep 식별

동기화 메커니즘이 이미 존재하면 `time.Sleep`은 불필요하다:

```go
// Destroy 내부에서 이미 <-entry.done으로 goroutine 완료를 대기
func (lp *LocalProcess) Destroy(ctx context.Context, id common.KernelID) error {
    // ...
    <-entry.done  // reaper goroutine 완료 대기
    // ...
}

// 따라서 테스트에서 Destroy 후 추가 sleep은 불필요 ❌
lp.Destroy(ctx, id)
time.Sleep(50 * time.Millisecond)  // 제거해야 함
```

코드가 channel, WaitGroup, Mutex 등으로 동기화를 보장하면
테스트에서 별도의 sleep을 추가하지 않는다.
