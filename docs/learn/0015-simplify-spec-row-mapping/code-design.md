# Code Design

## 필요한 추상화만 남기기

이번 변경에서는 `queryerContext`와 `getSpecByRunID`를 제거했다. 둘 다 일반적으로는 나쁜 구조가 아니지만, 현재 코드에서는 실제 이득보다 간접성이 더 컸다.

추상화는 미래 가능성만으로 두기보다 현재 반복, 변경 압력, 테스트 필요성 중 하나를 해결할 때 힘이 생긴다. 지금 `GetSpec`의 query는 repository 내부에서 한 번만 사용되고, 트랜잭션이나 mock queryer가 필요한 구조도 아니다. 그래서 helper 함수로 분리된 query를 다시 `GetSpec` 안으로 넣는 편이 읽는 사람에게 더 정직하다.

좋은 기준은 "이 이름이 코드보다 더 많은 의미를 주는가?"이다. `getSpecByRunID`는 `GetSpec`과 거의 같은 의미였고, `queryerContext`는 아직 실제 소비자가 없었다. 이럴 때는 구조를 줄이는 편이 낫다.

## 관심사에 맞는 위치 고르기

JSON 컬럼을 Go struct로 읽는 동작은 domain data 자체의 책임이 아니라 DB row mapping의 책임이다. 그래서 `spec.ModelOptions` 같은 순수 데이터 타입에 `Scan`을 구현하지 않고, `entity` 패키지 내부에 `jsonField[T]` wrapper를 만들었다.

이 구조의 장점은 domain 타입이 persistence 기술을 모른다는 점이다. domain data는 JSON 컬럼, SQLite, `database/sql.Scanner`를 몰라도 된다. 반대로 DB entity는 "이 컬럼은 JSON으로 저장되어 있고 읽을 때 특정 타입으로 복원한다"는 저장소 규칙을 명시적으로 가진다.

이런 배치는 layer 의존성도 깔끔하게 만든다. repository/db/entity는 common data 타입을 가져다 저장소 모양으로 감싸지만, common data는 repository/db를 알지 않는다.
