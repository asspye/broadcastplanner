# BroadcastPlanner — Cross-Platform

Кросс-платформенная реализация эфирного планировщика **TV Assembly** (оригинал —
macOS/SwiftUI, см. `../BroadcastPlanner`). Архитектура: **Go backend + REST/WS API +
браузерный фронтенд**, всё в Docker.

Полный разбор оригинала и спецификация форматов — в `../BroadcastPlanner/ANALYSIS.md`.

## Стек и решения

- **Backend:** Go (статичный бинарник, alpine-образ с ffmpeg для анализа медиа).
- **Хранилище:** Postgres (данные проектов) + JSON-совместимость с `.tvassembly`.
- **Медиа:** S3-совместимое хранилище (**MinIO** в compose) вместо NAS.
  `rename/relink/sync` из оригинала → операции над объектами бакета; `ffprobe`
  читает из S3.
- **Кодировки/форматы:** собственный кодировщик **CP1251** без внешних зависимостей;
  TELE CSV пишется как `;`-разделитель, кавычки, **CRLF**, CP1251/UTF-8.

## Дорожная карта

- **M1 — ядро логики + тесты (в работе):** доменные типы, таймкоды, сегментация,
  экспорт TELE/universal — портированы с golden-тестами из оригинала. ✅ базовый набор
- **M2 — API + БД + CRUD:** схема Postgres, эндпоинты проектов/базы/плейлиста, ffprobe.
- **M3 — Frontend:** три-панельная сборка (медиатека → превью → плейлист), режимы.
- **M4:** PROMO/FAST, split по датам, архив эфира, rename/relink/sync по S3.

## Структура

```
backend/
  cmd/server/          — HTTP entrypoint (health + demo TELE export)
  internal/domain/     — типы, framerate, timecode, segmentation (+ tests)
  internal/exporter/   — CSV/TELE экспорт, CP1251 (+ tests)
  internal/importer/   — (M2) парсеры marker/catalog/converter
  Dockerfile, Makefile
docker-compose.yml     — backend + postgres + minio
```

## Запуск

```sh
# тесты ядра
cd backend && go test ./...

# локально
make -C backend run          # http://localhost:8080/healthz

# весь стек в Docker
docker compose up --build
```

Демо-эндпоинт: `POST /api/export/tele` с телом `{frameRate, preset, playlist, markers}`
возвращает TELE CSV.
