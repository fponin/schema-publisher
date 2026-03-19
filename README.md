# hpub — GraphQL subgraph schema publisher

CLI-инструмент для публикации схем GraphQL-подграфов в [GraphQL Hive](https://the-guild.dev/graphql/hive).

Автоматизирует полный цикл: интроспекция схемы → проверка совместимости → публикация.

## Как работает

**Полный пайплайн** (схема берётся из кластера):
```
kubectl port-forward → Rover introspect → Hive schema:check → Hive schema:publish
```

**Режим schema-only** (есть готовый файл схемы):
```
Hive schema:check → Hive schema:publish
```

## Установка

### Скачать бинарник

На странице [Releases](https://github.com/fponin/schema-publisher/releases/latest) скачай два файла под свою платформу:

| Файл | Платформа |
|------|-----------|
| `hpub-darwin-arm64` | macOS Apple Silicon |
| `hpub-darwin-amd64` | macOS Intel |
| `hpub-windows-amd64.exe` | Windows |
| `hpub-linux-amd64` | Linux |
| `defaults.yaml` | шаблон конфигурации (нужен всем) |

### Зависимости

- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [rover](https://rover.apollo.dev) — Apollo Rover CLI
- [hive CLI](https://the-guild.dev/graphql/hive/docs/api-reference/cli) v0.42.1

```bash
npm install -g @graphql-hive/cli@0.42.1
```

### Установка бинарника

**macOS / Linux:**

```bash
# переименуй бинарник в hpub
mv hpub-darwin-arm64 hpub   # или hpub-darwin-amd64 / hpub-linux-amd64

sudo cp hpub /usr/local/bin/ && sudo chmod +x /usr/local/bin/hpub
```

**Windows** — переименуй `hpub-windows-amd64.exe` в `hpub.exe` и добавь в `PATH`.

Конфиг создаётся автоматически при первом запуске. Hive access token будет запрошен интерактивно.

> **ВАЖНО** `defaults.yaml` из релиза это шаблон конфига с окружениями и подграфами.
> Заполни его перед установкой бинарника.

## Использование

```bash
hpub                                 # интерактивный визард
hpub run --env dev                   # пропустить выбор окружения
hpub run --schema ./schema.graphql   # использовать готовый файл схемы
hpub check --schema ./schema.graphql # только проверить, без публикации
hpub config show                     # показать текущий конфиг
hpub config edit                     # открыть конфиг в $EDITOR
```

## Конфигурация

Конфиг живёт в `~/.config/hpub/config.yaml`. Структура:

```yaml
defaults:
  schemaFile: "~/new.graphql"   # дефолтный путь к файлу схемы
  hiveEndpoint: "https://..."   # endpoint Hive registry

environments:
  dev:
    authUrl: "..."
    authBearerToken: "..."
    defaultLocalPort: 8080
    jwtHeader: "jwt-token"
    kubectlContext: ""          # заполняется при первом запуске
    hiveEndpoint: ""            # заполняется при первом запуске
    hiveAccessToken: ""         # заполняется при первом запуске
  stage: { ... }
  prod: { ... }

subgraphs:
  - name: my-service
    publishUrl: "http://my-service:8080/graphql"
    k8sResource: "svc/my-service"
    namespace: my-namespace
    remotePort: 8080
    graphqlPath: "/graphql"
```

## Сборка

```bash
make build    # собрать бинарник ./hpub
make install  # собрать + установить в /usr/local/bin/hpub
make test     # запустить тесты
make lint     # go vet
make package  # собрать дистрибутив в dist/ (бинарник + defaults.yaml)
```
