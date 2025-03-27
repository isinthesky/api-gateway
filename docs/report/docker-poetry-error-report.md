
# 도커 컨테이너 빌드 및 실행 이슈 해결 과정

## 1. 문제 상황

Docker 컨테이너 빌드 및 실행 시 두 가지 주요 오류가 발생했습니다:

1. Dockerfile에서 Poetry 관련 오류
2. 데이터베이스 연결 문제 - 잘못된 호스트 이름 사용

## 2. Poetry 관련 오류 해결

### 문제점
Dockerfile에서 `poetry lock`과 `poetry install` 명령이 분리되어 있어 문제가 발생했습니다.

```dockerfile
# 문제가 있는 코드
RUN poetry lock
RUN poetry install --no-root --no-interaction
```

### 해결 방법
두 명령을 하나로 결합하여 해결했습니다:

```dockerfile
# 수정된 코드
RUN poetry lock --no-update && poetry install --no-root --no-interaction
```

다음 파일들을 수정했습니다:
- auth-fastapi-be/Dockerfile
- receipts-fastapi-be/Dockerfile

## 3. 데이터베이스 연결 문제 해결

### 문제점
서비스가 잘못된 데이터베이스 호스트 이름을 사용하여 연결을 시도했습니다:
- auth-service: "auth-postgres" (잘못됨) -> "auth-db" (올바름)
- receipt-service: "receiptally-db" (잘못됨) -> "receipt-db" (올바름)

이로 인해 다음과 같은 오류가 발생했습니다:
```
psycopg2.OperationalError: could not translate host name "auth-postgres" to address: Name or service not known
```

### 해결 방법
다음 파일들에서 호스트 이름을 수정했습니다:

1. 시작 스크립트 수정:
   - auth-fastapi-be/start.sh: "receiptally-db" -> "auth-db"
   - receipts-fastapi-be/start.sh: "receiptally-db" -> "receipt-db"

2. 환경 설정 파일 수정:
   - auth-fastapi-be/src/settings/environment.py:
     ```python
     DATABASE_URL: str = os.getenv("DATABASE_URL", "postgresql+asyncpg://receiptally:receiptally123@auth-db:5432/auth-db")
     DB_HOST: str = os.getenv("DB_HOST", "auth-db")
     ```

3. Alembic 설정 파일 수정:
   - auth-fastapi-be/alembic.ini:
     ```
     sqlalchemy.url = postgresql://receiptally:receiptally123@auth-db:5432/auth-db
     ```
   - receipts-fastapi-be/alembic.ini:
     ```
     sqlalchemy.url = postgresql://receiptally:receiptally123@receipt-db:5432/receipt-db
     ```

## 4. 결과

위 수정 사항을 적용하여 다음 명령으로 컨테이너를 다시 빌드하고 실행했습니다:

```bash
docker-compose up --build
```

이제 웹 서버가 정상적으로 구동됩니다.

## 5. 교훈

1. Docker 컨테이너 간의 네트워크 통신 시 올바른 서비스 이름(호스트 이름)을 사용해야 합니다.
2. Poetry와 같은 패키지 관리자 사용 시 락 파일과 설치 과정의 순서가 중요합니다.
3. 여러 마이크로서비스로 구성된
4. 시스템에서는 일관된 설정 관리가 필요합니다.
