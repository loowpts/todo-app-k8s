# todo-app

Kubernetes-развёртывание todo-приложения (Go REST API + HTML-фронтенд, Postgres,
Redis) с NetworkPolicy (default-deny + точечные allow), RBAC, HPA/PDB,
health-пробами и Kustomize.

```
todo-app/   исходники приложения, Dockerfile, docker-compose.yml
k8s/        манифесты для развёртывания в Kubernetes (Kustomize)
```

## Kubernetes

Манифесты (Deployment, StatefulSet, Service, Ingress, NetworkPolicy, HPA, PDB,
RBAC) лежат в [`k8s/`](k8s) и собираются через Kustomize:

```
k8s/
  namespace.yaml, ingress.yaml, limit-range.yaml, resource-quota.yaml
  kustomization.yaml           корневой Kustomize-файл
  base/
    kustomization.yaml         собирает всё ниже
    app/      Deployment/Service/ConfigMap/Secret/HPA/PDB backend + Job миграций
    postgres/ StatefulSet + headless Service
    redis/    Deployment + Service (кэш, без персистентности)
    netpol/   default-deny-all + точечные allow-правила
    rbac/     ServiceAccount/Role/RoleBinding для backend-пода
```

### Секреты — перед первым apply

Два файла с реальными значениями не хранятся в git (см. `.gitignore`) и
создаются локально из шаблонов:

```bash
cp k8s/base/app/secret.yaml.example k8s/base/app/secret.yaml
# отредактировать пароли внутри

kubectl create secret docker-registry gitlab-registry-secret \
  --namespace todo-app \
  --docker-server=<REGISTRY_HOST>:<PORT> \
  --docker-username=<USERNAME> \
  --docker-password=<TOKEN> \
  --dry-run=client -o yaml > k8s/gitlab-registry-secret.yaml
```

Без них `kubectl apply -k k8s/` упадёт с "no such file" — это ожидаемо.

### Применение

```bash
kubectl apply -k k8s/
```

### Адаптация под свой кластер

- `nodeSelector: kubernetes.io/hostname: k8s-worker1/2` (StatefulSet Postgres,
  Deployment Redis) — привязка к нодам моего домашнего кластера, замени под свои
  или убери.
- `image: 192.168.0.21:5050/...` — мой приватный registry, замени на свой.
- Redis стартует с `--requirepass`, пароль берётся из `app-secret`
  (`REDIS_PASSWORD`) — менять только в Secret, деплой подхватит на rollout.
- RBAC (`rbac/`) даёт поду `backend` право читать список подов в неймспейсе.
  Само приложение сейчас не ходит в Kubernetes API — это заготовка на будущее
  (ServiceAccount + Role + RoleBinding + egress к apiserver в NetworkPolicy),
  а не рабочая функциональность. Не нужно — удали `rbac/` и соответствующее
  правило в `netpol/backend-egress-netpol.yaml`.

## Приложение

Стек: Go (`chi`), Postgres (миграции через `golang-migrate`, встроены в бинарник
через `go:embed`, применяются автоматически при старте), Redis (кэш списка задач
+ rate limiting), `html/template` для фронтенда.

### Ручки

**Health (для k8s-проб)**

| Метод | Путь       | Что делает                                                    |
|-------|------------|----------------------------------------------------------------|
| GET   | `/healthz` | liveness — `200 ok`, не трогает БД/Redis                       |
| GET   | `/readyz`  | readiness — пингует Postgres и Redis, `503` если что-то недоступно |

**JSON API** (`/api/tasks`, rate limit по умолчанию 120 req/min на IP)

| Метод  | Путь              | Тело                                    | Ответ            |
|--------|-------------------|-------------------------------------------|--------------------|
| GET    | `/api/tasks`      | —                                          | `200` список       |
| POST   | `/api/tasks`      | `{"title","description"}`                  | `201` создана      |
| GET    | `/api/tasks/{id}` | —                                          | `200` / `404`      |
| PUT    | `/api/tasks/{id}` | `{"title","description","done"}`           | `200` / `404`      |
| DELETE | `/api/tasks/{id}` | —                                          | `204` / `404`      |

**HTML-фронтенд**

`GET /`, `GET /tasks/new`, `POST /tasks`, `GET /tasks/{id}/edit`,
`POST /tasks/{id}`, `POST /tasks/{id}/delete`, `GET /static/*`.

### Конфигурация (переменные окружения)

| Переменная                | По умолчанию                                                        |
|-----------------------------|-------------------------------------------------------------------|
| `HTTP_PORT`                | `8080`                                                              |
| `POSTGRES_DSN`              | `postgres://postgres:postgres@localhost:5432/todoapp?sslmode=disable` |
| `REDIS_ADDR`                | `localhost:6379`                                                    |
| `REDIS_PASSWORD`            | ``                                                                   |
| `REDIS_DB`                  | `0`                                                                  |
| `CACHE_TTL_SECONDS`         | `10`                                                                 |
| `RATE_LIMIT_RPM`            | `120`                                                                |
| `SHUTDOWN_TIMEOUT_SECONDS`  | `15`                                                                 |

См. также [todo-app/.env.example](todo-app/.env.example).

### Запуск локально

```bash
cd todo-app
docker-compose up --build
```

Приложение — http://localhost:8080, вместе с Postgres и Redis, миграции
применяются автоматически при старте контейнера `app`.
