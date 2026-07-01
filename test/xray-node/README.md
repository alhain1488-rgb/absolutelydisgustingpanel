# test/xray-node — локальная нода Xray для интеграционных тестов

Образ содержит Go и реальный бинарник `xray-core`. Он прогоняет интеграционный
тест, который собирает `config.json` с **VLESS Reality + VMess + Trojan** и
проверяет, что `xray -test` его принимает (Фаза 4 ROADMAP; позже — проверка
пуша sync-движком, Фаза 5).

## Запуск

Из корня репозитория:

```bash
docker build -t xray-node -f test/xray-node/Dockerfile .
docker run --rm xray-node
```

Ожидаемый результат — тест `TestGeneratedConfig_AcceptedByXray` проходит
(`xray -test` принял сгенерированный конфиг).

## Без Docker

Если `xray` установлен локально:

```bash
cd backend
XRAY_BIN=$(command -v xray) go test -tags integration ./internal/xrayconfig/...
```

Без тега `integration` этот тест не собирается и не влияет на обычный
`go test ./...`. Если бинарник `xray` не найден — тест помечается как skipped.
