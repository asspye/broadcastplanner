# План порта TV Assembly → BroadcastPlanner (кросс-платформенный)

Пошаговый план переноса **всей** логики нативного macOS-приложения (`../BroadcastPlanner`,
детальный разбор — `ANALYSIS.md`) в архитектуру **Go backend + Postgres + S3(MinIO) +
браузерный фронт**. Документ — «источник правды» по объёму работ и порядку.

Принципы:
1. **Логика переносится 1:1** и покрывается golden-тестами, портированными из
   `Tests/TVAssemblyTests/TimecodeTests.swift` (2096 строк эталонных значений).
2. **Платформенное — заменяется** (AVFoundation→ffprobe, NSPanel→HTTP, /usr/bin/zip→
   archive/zip, .tvassembly→Postgres, NAS→S3, AVPlayer→`<video>`).
3. **Экспорт — мультиформатный слой адаптеров** (TELE, Forward, BramTech, …), см. §3.
4. Каждый шаг = отдельный PR/коммит с тестами; ничего не считается готовым без зелёных тестов.

Статусы: ✅ готово · 🟡 в работе · ⬜ не начато.

---

## 1. Карта логики нативного приложения (что вообще портируем)

Полный инвентарь функций — в `ANALYSIS.md §4`. Ниже — сгруппировано по слоям, с целевым
пакетом Go и статусом.

### 1.1 Доменное ядро → `internal/domain`
| Логика | Источник (Swift) | Go | Статус |
|--------|------------------|-----|--------|
| FPS (real vs nominal) | `ProjectFrameRate` | `framerate.go` | ✅ |
| Таймкоды (clock/timecode/broadcastClock/decimal) | `BroadcastFormatters` | `timecode.go` | ✅ |
| Теги графики (LOGO/age/SCTE/REKLAMA/…) | `GraphicTag` | `graphictag.go` | ✅ |
| Медиа-ассет + флаги/labels/storageName | `MediaAsset` | `asset.go` | ✅ |
| Строка плейлиста + RowKind + флаги | `PlaylistItem` | `playlist.go` | ✅ |
| Маркеры | `AdMarker`/`MarkerKind` | `playlist.go` | ✅ |
| Сегментация по AD BREAK | `PlaylistSegmentation` | `segmentation.go` | ✅ |
| ID (UUID) | `UUID` | `id.go` | ✅ |
| **Пересчёт эфирного времени + суточные границы** | `recalculatePlaylist` | `recalculate.go` | ⬜ |
| **Проверка качества (8 кодов)** | `buildPlaylistQualityReport` | `quality.go` | ⬜ |
| Converter-проверка качества | `buildConverterPlaylistQualityReport` | `quality.go` | ⬜ |
| segmentLabel «N из M» | `segmentLabel` | `segmentlabel.go` | ⬜ |
| duplicateRangeKey / shouldRequireAdMarkers | там же | `quality.go` | ⬜ |
| Пакеты по датам эфира | `playlistExportBatchesByAirDate` | `batches.go` | ⬜ |
| Даты эфира (валидация/±дни/формат) | `BroadcastAirDate` + helpers | `airdate.go` | ⬜ |

### 1.2 Импорт → `internal/importer`
| Логика | Источник | Go | Статус |
|--------|----------|-----|--------|
| Импорт точек CSV/XML (авто-кодировка, сводки, playlist-XML) | `MarkerImporter` | `marker.go` | ⬜ |
| Нормализация имён (`normalizedName`, `parseTime`) | `MarkerImporter` | `normalize.go` | ⬜ |
| Импорт каталога CSV/XML | `MediaCatalogExporter.importRows` | `catalog.go` | ⬜ |
| Импорт конвертера (CSV/TSV/XML/XLSX, словари синонимов, сегменты, даты) | `PlaylistConverterImporter` | `converter.go` | ⬜ |
| Онлайн-импорт плейлиста + матчинг к базе | `importOnlinePlaylistItems` + matching | `online.go` | ⬜ |
| Обучение AD-маркеров из повторов | `learnAdBreakMarkersFromImportedPlaylist` | `online.go` | ⬜ |
| Разбор XLSX (unzip + sharedStrings + sheet) | `PlaylistConverterImporter` | `xlsx_read.go` | ⬜ |

### 1.3 Экспорт → `internal/exporter` (мультиформат, см. §3)
| Логика | Источник | Go | Статус |
|--------|----------|-----|--------|
| Единая модель строки экспорта (`AirEvent`) | `ExportRow` | `airevent.go` | 🟡 (есть `exportRow`) |
| Адаптер TELE (CP1251/UTF-8) | `TeleRow` | `target_tele.go` | ✅ (как `TeleCSV`) |
| Universal/Segments/Items CSV | `ExportRow.csvLine` | `writers.go` | ✅ |
| XML-экспорт | `writeXML` | `target_xml.go` | ⬜ |
| XLSX-экспорт (archive/zip) | `writeXLSX` | `xlsx_write.go` | ⬜ |
| **Адаптер Forward (ForwardT / FDOnAir)** | — нет в оригинале | `target_forward.go` | ⬜ (нужны образцы) |
| **Адаптер BramTech** | — нет в оригинале | `target_bramtech.go` | ⬜ (нужны образцы) |
| Каталог медиа CSV/XML/XLSX | `MediaCatalogExporter` | `catalog_export.go` | ⬜ |

### 1.4 Оркестрация плейлиста (сервисный слой) → `internal/service`
Это то, что в Swift было в `BroadcastStore` как императивные методы над состоянием. В web
становится сервисом над данными БД, дергается из API. Логика — та же.
| Группа | Методы-источники | Статус |
|--------|------------------|--------|
| Добавление/вставка (в т.ч. между сегментами) | `insertPlaylistItemAfterSelection`, `insertAfterSegment`, блоки | ⬜ |
| Комментарии / LIVE-заглушки / дата-строки | `insertCommentRow`, `insertLiveBreak`, `upsertAirDateComment` | ⬜ |
| Перестановка / drag&drop | `movePlaylistItems(ids:targetItemID:placement:)` | ⬜ |
| Копирование/вставка/дублирование | `copySelection`/`pasteClipboard`/`duplicate` | ⬜ |
| Удаление (выбор/фильтр/всё) | `remove*` | ⬜ |
| Подрезка IN/OUT | `trimSelectedPlaylistItem` | ⬜ |
| Маркеры add/remove | `addMarker`/`removeSelectedMarker` | ⬜ |
| **PROMO/FAST (вставка промо по интервалу)** | `insertPromoFast` | ⬜ |
| **PROMO/FAST конвертера (SCTE/REKLAMA теги)** | `applyConverterPromoFast` | ⬜ |
| Старт эфира / дата эфира / split | `setPlaylistClockStart`, `setPlaylistAirDate`, `splitPlaylistByAirDates` | ⬜ |
| Фильтры/поиск базы и плейлиста | `filteredAssets`, `filteredPlaylistEntries` | ⬜ |

### 1.5 Медиатека и файлы → `internal/media` + `internal/storage`
| Группа | Источник | Замена | Статус |
|--------|----------|--------|--------|
| Анализ медиа (duration/fps/size/kind) | `MediaAnalyzer` (AVFoundation) | **ffprobe** из S3 (presigned) | ⬜ |
| Импорт файлов/папок + дедуп | `importNewMediaFiles`/`expandedFileURLs` | загрузка/скан бакета S3 | ⬜ |
| Метаданные ассета (name/id/genre/…/категории/графика) | `setSelected*`, `updateSelectedAssets` | PATCH ассета | ⬜ |
| Присвоение последовательных ID | `assignSequentialExternalIDs` | сервис | ⬜ |
| **Rename в Storage-имя** | `renameFilesToStorage` (FileManager) | **copy/rename объекта S3** + план/валидация | ⬜ |
| **Relink (одиночный/рекурсивный)** | `relinkMedia*` | поиск объекта в бакете по имени/ID | ⬜ |
| **Sync базы→файлы** | `synchronizeAllFilesToStorage` | сервис над S3 | ⬜ |
| Служебные логи операций | `writeServiceLog` | таблица `service_logs` (+ опц. файл) | ⬜ |

### 1.6 Проекты/персистентность/архив → `internal/store` (Postgres)
| Группа | Источник | Замена | Статус |
|--------|----------|--------|--------|
| Модель проекта | `BroadcastProject` (JSON) | таблицы + импорт/экспорт `.tvassembly` | ⬜ |
| Save/Open/New/Autosave | `saveProject`/`openProject`/`autosave` | CRUD + серверный автосейв/версии | ⬜ |
| «Последний проект» | `UserDefaults` | сессия/таблица настроек | ⬜ |
| **Архив эфира JSONL** | `archiveCurrentPlaylist` | таблица `broadcast_archive` (+ экспорт JSONL) | ⬜ |
| Отчёты по архиву (месяц/квартал/год) | — (задел) | запросы по архиву | ⬜ |

### 1.7 UI → `frontend/` (браузер)
| Экран/режим | Источник | Статус |
|-------------|----------|--------|
| 3-панельная сборка (база / превью / плейлист) | `assemblyWorkspace` | ⬜ |
| AD Break Points (превью + аудиометр + маркеры) | `adBreakPointsWorkspace` | ⬜ |
| Media Prep (карточка метаданных) | `MediaPrepWorkspace` | ⬜ |
| Playlist Converter (офлайн) | `playlistConverterWorkspace` | ⬜ |
| Broadcast Archive | `broadcastArchiveWorkspace` | ⬜ |
| Превью-плеер + шаг кадра + аудиометр | `PreviewView`/`AudioMeterView` | ⬜ |
| Таблица плейлиста (сегменты, качество, drag&drop) | `PlaylistItemRows` и др. | ⬜ |
| Панель качества / бейджи проблем | `PlaylistQualityPanel` | ⬜ |

---

## 2. Порядок работ по фазам

### Фаза 0 — каркас ✅
Go-модуль, Docker (alpine+ffmpeg), compose (backend+postgres+minio), health, CI-тесты.

### Фаза 1 — доменное ядро (🟡 идёт)
Шаги (каждый = коммит с тестами):
1. ✅ framerate + timecode
2. ✅ graphictag + asset + playlist + segmentation
3. ⬜ **airdate.go** — валидация даты, `±дни`, формат `"DD MM YYYY"`, `isAirDateComment`.
4. ⬜ **recalculate.go** — пересчёт `startOffset` от `clockStart`, вставка дата-комментариев
   на суточных границах. Тесты: `playlistClockStartAppliesToRowsAndExport`,
   `playlistClockStartWrapsAfterTwentyFourHours`.
5. ⬜ **segmentlabel.go** — «N из M». Тест: `playlistSegmentLabelUsesSourceMarkersAcrossSplitRows`.
6. ⬜ **quality.go** — 8 кодов проблем (MEDIA/DUR/READY/FPS/AD/RANGE/SHORT/DUP) + converter-режим.
   Тесты: `playlistQualityReport*` (порог AD>10мин, SHORT<5с, FPS>0.2, диапазон, дубликаты).
7. ⬜ **batches.go** — резка по дата-комментариям.

### Фаза 2 — экспорт (мультиформат, см. §3)
1. ✅ TELE CSV (CP1251/UTF-8) + universal/segments/items CSV.
2. ⬜ Вынести единую модель `AirEvent` и интерфейс `PlaylistTarget`; TELE переоформить как адаптер.
3. ⬜ XML-адаптер (тест `playlistExporterWritesItemProfileRows`).
4. ⬜ XLSX через `archive/zip` (без внешнего zip).
5. ⬜ Каталог медиа CSV/XML/XLSX + импорт каталога (тесты `mediaCatalog*`).
6. ⬜ **Адаптер Forward** (по образцам — см. §4).
7. ⬜ **Адаптер BramTech** (по образцам — см. §4).

### Фаза 3 — импортёры
1. ⬜ normalize (`normalizedName`, `parseTime`) — тесты `markerImporterParsesTimecodeAndNormalizesNames`.
2. ⬜ marker CSV/XML — тесты `markerImporterReads*`, `...DoesNotTurnPlaylistSegmentColumns...`.
3. ⬜ converter (CSV/TSV/XML/XLSX, словари синонимов, сегменты, даты).
4. ⬜ online-import + матчинг к базе + обучение AD — тесты `onlinePlaylistImport*`.

### Фаза 4 — БД + API (M2)
1. ⬜ Схема Postgres (projects, assets, playlist_items, markers, service_logs, broadcast_archive)
   + миграции; импорт/экспорт `.tvassembly` для совместимости.
2. ⬜ Сервисный слой (§1.4) поверх БД.
3. ⬜ REST-эндпоинты (см. `ANALYSIS.md §8.2`) + WebSocket (прогресс анализа, автосейв, статус).
4. ⬜ ffprobe-анализ из S3 (очередь задач, WS-прогресс).

### Фаза 5 — файлы/S3 (M2/M3)
1. ⬜ storage-интерфейс (S3/MinIO): list/stat/copy/rename/presign.
2. ⬜ rename-to-storage + relink (одиночный/рекурсивный) + sync — с планами/валидацией/логами.

### Фаза 6 — Frontend (M3)
1. ⬜ Каркас SPA + клиент API + WS.
2. ⬜ Медиатека → плейлист → превью (`<video>`) → маркеры.
3. ⬜ Панель качества, сегменты, drag&drop, режимы, экспорт (выбор целевой системы).

### Фаза 7 — фичи-надстройки
1. ⬜ PROMO/FAST (оба варианта).
2. ⬜ Split по датам, архив эфира + отчёты.
3. ⬜ Пакетный экспорт по датам во все форматы.

---

## 3. Архитектура мультиформатного экспорта (ключевое)

Экспорт разделяется на **нормализацию** (плейлист → список эфирных событий) и **рендер**
(события → байты конкретного формата). Один нормализатор, много адаптеров.

### 3.1 Единая модель `AirEvent`
Нормализатор (порт `exportRows`) раскрывает плейлист с учётом профиля (universal/segments/
items) в плоский список событий:

```go
type AirEvent struct {
    ItemIndex, SegmentIndex int
    StartTC, DurationTC     string   // broadcastClock / timecode
    SourceInTC, SourceOutTC string
    StartSec, DurationSec   float64  // сырьё для форматов, считающих в секундах/кадрах
    Kind                    string   // MediaKind rawValue
    Title, Storage, File, Path string
    Graphics    []string             // GraphicLabels (сорт.)
    GraphicTags map[string]bool      // быстрый доступ: LOGO/SCTE/REKLAMA/age/…
    Markers     []domain.AdMarker
    Note        string
}
```

### 3.2 Интерфейс адаптера и реестр
```go
type PlaylistTarget interface {
    ID() string             // "tele-cp1251","tele-utf8","forward","bramtech","xml","xlsx"
    Title() string          // человекочитаемо для UI
    FileExtension() string  // "csv","xml","air","xlsx",...
    Render(events []AirEvent, fr domain.ProjectFrameRate, opts Options) ([]byte, error)
}

var registry = map[string]PlaylistTarget{}  // регистрация адаптеров
func Targets() []PlaylistTarget             // список для UI (выпадающий выбор системы)
func Render(id string, playlist, markers, fr, opts) ([]byte, string /*ext*/, error)
```

- Адаптеры: `tele`, `forward`, `bramtech`, `xml`, `xlsx`, `universal-csv`.
- Каждый адаптер — маппинг `AirEvent` → структура своего формата (у Forward/BramTech свои
  колонки/теги/кодировки/расширения). Кодировки (CP1251 и др.) — уже есть в `cp1251.go`.
- **Пакетный экспорт** (по датам) и **архивация** работают одинаково для любого адаптера.

### 3.3 Что даёт такой дизайн
Добавить новую плейаут-систему = написать один файл-адаптер + тест на образце. Ядро,
пересчёт, сегментация, PROMO/FAST, качество — не трогаются.

---

## 4. Что мне нужно от тебя (блокеры для Forward / BramTech)

TELE вышел точным только потому, что были **боевые образцы** (`test batch/*TELE CP1251.csv`).
Для остальных плейаут-систем нужно то же самое:

Для **каждой** целевой системы (Forward, BramTech, и любых других):
1. **1–2 реальных файла экспорта** (как их сейчас формируют/ждут).
2. **Спецификация формата**, если есть: список колонок/тегов, разделитель, кодировка,
   расширение файла, как кодируются: старт-таймкод, длительность, IN/OUT, Storage/ID,
   графика (LOGO/возраст/SMOKE), рекламные метки (SCTE/аналог), комментарии/LIVE.
3. Особенности: дискретность кадров/fps, поведение на границе суток, требуется ли отдельная
   строка на сегмент или на элемент.

По Forward (ForwardT/FDOnAir) и BramTech у меня нет гарантированно точной спецификации —
делать «на глаз» рискованно. С образцами адаптеры сделаю так же точно, как TELE.

---

## 5. Definition of Done по каждому шагу
- Логика портирована из соответствующего Swift-источника.
- Есть Go-тест с golden-значениями (по возможности из `TimecodeTests.swift` или из образца формата).
- `go vet` и `go test ./...` зелёные; gofmt чистый.
- Обновлён статус в этом документе.
