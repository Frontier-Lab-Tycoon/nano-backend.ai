# LocalProcess 런타임 — status 조회

PR: #47
Date: 2026-03-21

## What was done

- `LocalProcess.Status()` 메서드 구현 — stub을 실제 프로세스 상태 감지 로직으로 교체
- `done` channel의 non-blocking select로 프로세스 실행/종료 상태 판별
- 5가지 시나리오에 대한 단위 테스트 추가

## Categories

- [Go Programming](./go.md) — non-blocking channel select, ProcessState, 테스트 유틸리티와 패턴

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|------------------------|
| Non-blocking `select` on `done` channel | 프로세스 상태를 polling 없이 즉시 확인 가능; reaper goroutine이 이미 `done`을 close함 | `cmd.Process.Signal(syscall.Signal(0))` (OS-specific, race condition 가능) |
| `cmd.ProcessState.ExitCode()` 사용 | `Wait()` 완료 후 자동으로 populated; 별도 파싱 불필요 | `ProcessState.Sys().(syscall.WaitStatus)` (더 세밀하지만 불필요한 복잡도) |

## Further study

- [ ] `ProcessState.ExitCode()`가 -1을 반환하는 경우 — signal로 종료된 프로세스 처리
- [ ] Status 호출 빈도가 높을 때 `sync.RWMutex` 전환 검토
