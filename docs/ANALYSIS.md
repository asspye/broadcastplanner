# TV Assembly — полный разбор кода и спецификация для порта

Документ описывает существующее macOS-приложение **TV Assembly** (папка `BroadcastPlanner`)
целиком: доменную модель, все основные функции, алгоритмы, форматы файлов и
платформенные зависимости. Цель — служить основой для переписывания на архитектуру
**Linux-backend + REST/WS API + браузерный фронтенд**.

---

## 1. Что это за ПО

Инструмент подготовки **эфирного плейлиста телеканала** (FAST/линейное ТВ, детский канал
Останкино Телеком). Пользователь:

1. Импортирует медиатеку (файлы/папки или каталог CSV/XML).
2. Раскладывает материалы в **последовательный плейлист** с эфирными таймкодами.
3. Размечает **рекламные точки (AD BREAK)**, обрезает исходники (IN/OUT).
4. Навешивает **эфирную графику** (LOGO, возрастные метки, SCTE, REKLAMA).
5. Прогоняет **проверку качества** плейлиста.
6. Экспортирует в форматы эфирного сервера, главный — **«ТЕЛЕ» CSV в CP1251**.
7. Ведёт **архив эфира** (JSONL) и служебные логи операций с файлами.

Есть отдельный офлайн-режим **Playlist Converter** — конвертация чужих CSV/XML/XLSX
плейлистов без медиабазы.

---

## 2. Технологический профиль оригинала

| Слой | Технология | Замена в порте |
|------|-----------|----------------|
| Язык | Swift 6, SwiftPM | Go / TS / Python (backend) |
| UI | SwiftUI + AppKit | React/Vue в браузере |
| Модель | `BroadcastStore: ObservableObject` (@MainActor) | сервис + state store на сервере / клиенте |
| Медиа-анализ | AVFoundation (`AVURLAsset`) | `ffprobe` |
| Файловые диалоги | `NSOpenPanel`/`NSSavePanel` | HTTP upload / выбор путей NAS |
| XLSX read/write | `Process` → `/usr/bin/zip` и `/usr/bin/unzip` | zip-библиотека языка |
| Персистентность проекта | JSON-файл `.tvassembly` | БД (Postgres) + JSON export |
| «Последний проект» | `UserDefaults` | таблица настроек / сессия |
| Автосохранение | `Task.sleep(2s)` дебаунс | серверный дебаунс / автосейв |
| Превью-плеер | AVPlayer + аудиометр | `<video>` + HLS/прямой файл, WebAudio |

**Ключевой вывод:** вся бизнес-логика чистая (таймкоды, сегментация, проверки, форматы),
почти не завязана на macOS. Платформенные привязки локализованы в: анализе медиа,
файловых диалогах, XLSX через внешний zip, `UserDefaults`, AVPlayer-превью.

---

## 3. Доменная модель (`Models.swift`)

### Перечисления

- **`MediaKind`**: `video` («Видео»), `audio` («Аудио»), `image` («Графика»), `unknown` («Файл»).
  *Внимание:* `rawValue` — русские строки, они пишутся в JSON проекта и каталог. Порт
  должен сохранить эти значения для совместимости или сделать миграцию.
- **`GraphicTag`**: `logo`, `smoke`, `plus0…plus18` (возрастные, `displayName` = «0+»…«18+»),
  `scte`, `reklama`. Свойства: `isAgeTag`, `ageTags` (набор), `sortOrder` (строковый ключ
  сортировки), `matching(label)` — распознавание из строки. Возрастные метки
  **взаимоисключающие** (при установке одной остальные снимаются).
- **`ProjectFrameRate`**: 23.98/24/25/29.97/30/59.94/60. `framesPerSecond` (реальный fps,
  23.976 и т.п.) и `nominalFrameCount` (24/25/30/60 — число кадров для отображения TC).
- **`PlaylistExportProfile`**: `universal` / `segments` / `items`.
- **`PlaylistCSVExportPreset`**: universal/segments/items/**teleCP1251**/**teleUTF8**;
  `isTele`, `profile` (все Tele-пресеты используют профиль universal).
- **`PlaylistExportMode`**: `single` / `batch` (пакетный — по датам эфира).
- **`MarkerKind`**: `inPoint` («IN»), `outPoint` («OUT»), `adBreak` («AD BREAK»).
- **`PlaylistCheckSeverity`**: error/warning/info. **`PlaylistIssueCode`**:
  ad/fps/media/duration/range/short/duplicate/readiness.

### Структуры

- **`MediaAsset`** (Identifiable/Codable/Hashable) — единица медиатеки:
  `id: UUID`, `name`, `url`, `kind`, `duration?`, `frameRate?`, `dimensions?` (`"1920x1080"`),
  `fileExtension`, `status` (строка состояния анализа), `graphicTags: Set<GraphicTag>`,
  `externalID` (Storage/ID), `comment`, `productionYear`, `director`, `production`, `genre`,
  `synopsis`, `categories: Set<String>`, `customGraphicTags: Set<String>`.
  Есть ручной `Decodable` с `decodeIfPresent` (обратная совместимость).
  Вычисляемые: `isVirtual` (путь начинается с `/TVAssembly/`), `isMissingPlaceholder`
  (`/TVAssembly/MissingMedia/`), `displayStatus` (если файла нет → «Нет файла»),
  `graphicLabels` (built-in по sortOrder + custom, объединённые), `storageName`
  (externalID или name).
  «Виртуальные» пути-заглушки: `/TVAssembly/CommentRow`, `/BreakHeader`, `/LiveBreak/…`,
  `/MissingMedia/…`, `/Converter/…`.

- **`PlaylistItem`** — строка плейлиста:
  `id`, `asset`, `startOffset` (позиция в эфире, сек), `duration`, `sourceIn`, `sourceOut`,
  `note`, `rowKind`, `commentText`. `RowKind`: `media`/`comment`/`breakHeader`/`liveBreak`.
  `duration = max(0, sourceOut - sourceIn)` (перевычисляется в init).
  Спец-конструкторы для комментария и заголовка LIVE-блока (создают виртуальный asset).
  Флаги: `isCommentRow`, `isBreakHeaderRow`, `isLiveBreakPlaceholder`, `isNonTimingRow`
  (comment ∪ breakHeader — **не участвуют в тайминге**), `isPreviewableRow`.

- **`PlaylistSegment`** — часть строки между AD BREAK-точками (id `"{itemUUID}-{index}"`),
  с `startOffset`/`endOffset`/`sourceIn`/`sourceOut`, `duration` вычисляется.

- **`PlaylistSegmentation.segments(for:markers:)`** — **чистая функция**: режет `PlaylistItem`
  на сегменты по маркерам `adBreak`, попадающим строго внутрь `(sourceIn, sourceOut)`.
  Точки дедуплицируются (толеранс 0.01с), сортируются; сегменты нарезаются последовательно.
  Порт должен воспроизвести 1:1.

- **`BroadcastAirDate`**: `day/month/year`, `displayText` = `"DD MM YYYY"` (формат «01 06 2026»).
- **`AdMarker`**: `id`, `kind`, `time` (сек), `note`.
- **`BroadcastProject`** (Codable) — то, что сериализуется в `.tvassembly`:
  `assets`, `playlist`, `markersByAssetPath: [String:[AdMarker]]`, `projectFrameRate`,
  `playlistClockStart`, `playlistAirDate?`, `savedAt`.
- **`BroadcastArchiveEntry`** — запись архива эфира (см. §6.6).
- **`FrameStepCommand`**, **`PlaylistCheckIssue`**, **`PlaylistQualityReport`** — служебные.

### Форматтеры таймкода (`BroadcastFormatters`) — критично воспроизвести точно

```
clock(t)               -> "HH:MM:SS"        (по целым секундам)
decimal(v)             -> "%.2f"
timecode(t, fps)       -> "HH:MM:SS:FF"
    totalFrames = round(t * realFps)
    frames      = totalFrames % nominalFrames
    seconds     = (totalFrames / nominalFrames) % 60 ... и т.д.
broadcastClockTimecode(t, fps) -> "HH:MM:SS:FF" с обёрткой на 24 часа (суточный эфирный TC)
    framesPerDay = 24*60*60*nominalFrames; totalFrames приводится в [0, framesPerDay)
```

`timecode` использует **nominalFrameCount** для деления, а `realFps` только для перевода
секунд в кадры. Это важная деталь для 23.98/29.97/59.94.

---

## 4. Ядро — `BroadcastStore` (полный инвентарь функций)

`@MainActor final class BroadcastStore: ObservableObject`. Единственный источник истины.
Ниже — все публичные и ключевые приватные методы, сгруппированные по назначению. Это и есть
**«основные функции»**, которые нужно перенести в сервисный слой backend + эндпоинты API.

### 4.1 Состояние (`@Published`)
`assets`, `playlist`, `markersByAssetPath`, выделения (media/playlist/segment/marker),
`activePanel`, `keyboardFocusArea`, флаги диалогов, `mediaSearchText`,
`selectedMediaCategoryFilter`, `playlistSearchText`, `previewPosition`, `frameStepCommand`,
`projectFrameRate`, `playlistClockStart`, `playlistAirDate`, `exportProfile`,
`playlistExportMode`, `isOfflinePlaylistConverterMode`, `statusLine`, `currentProjectURL`,
`hasUnsavedChanges`, `lastAutosaveURL`, `mediaAvailabilityRevision`.
Приватный кэш: `playlistQualityReportCache`, `playlistSegmentsCache`, буфер обмена плейлиста,
два «рабочих состояния» (assembly vs converter) для переключения режимов.

### 4.2 Вычисляемые представления (→ эндпоинты чтения/фильтры)
- `filteredAssets` — фильтр медиатеки по категории + полнотекстовый поиск (токены AND по
  склеенному haystack всех полей).
- `filteredPlaylistEntries` — то же для плейлиста (+ таймкоды сегментов и текст проблем).
- `mediaCategoryOptions`, `availablePlaylistGraphicLabels`.
- `playlistQualityReport` (кэш; assembly- или converter-вариант).
- `selectedPreviewAsset`, `selectedPreviewMarkers`, `selectedSegment`, флаги `can*`.

### 4.3 Импорт
- `importFiles()` / `importNewMediaFiles(from:)` — добавление файлов/папок в базу
  (рекурсивно, дедуп по стандартизованному пути), затем async-анализ; попутно
  `resolveMissingPlaylistItems`.
- `refreshMediaLibraryFromKnownFolders()` — до-сканирование известных папок.
- `importMarkerFiles()` — импорт точек CSV/XML, матч к ассету по `bestAssetMatch` (см. 4.11).
- `importMediaCatalog()` — импорт каталога (обновляет/добавляет ассеты, вливает маркеры).
- `importConverterPlaylist()` — офлайн-конвертер (заменяет плейлист целиком).
- `importOnlinePlaylist()` / `importOnlinePlaylistItems(...)` — импорт плейлиста в сборку с
  матчингом к базе (`onlinePlaylistAssetMatchIndex`), непойманные строки → «Missing media»
  заглушки; обучение AD-маркеров из повторов (`learnAdBreakMarkersFromImportedPlaylist`).

### 4.4 Построение плейлиста
- `addToPlaylist(_/assetID:/assetIDs:)`, `addMediaSelectionToPlaylist`, `addMediaAssetsToPlaylist`.
- `insertPlaylistItemAfterSelection` / `insertPlaylistBlockAfterSelection` — вставка
  после выделения, в т.ч. **между сегментами** (`insertAfterSegment` расщепляет строку на
  leading/insert/trailing).
- `insertCommentRowAfterSelection`, `insertLiveBreak(durationText:)` (создаёт header + liveBreak).
- `duplicateSelectedPlaylistEntry`, `copySelection`/`pasteClipboard` (буфер строк).
- Перестановка: `movePlaylistItems(from:to:)`, drag&drop
  (`playlistDragPayload`/`handlePlaylistDrop`/`movePlaylistItems(ids:targetItemID:placement:)`).
- Удаление: `requestDeletePlaylistSelection`/`confirmDeletePlaylistSelection`,
  `removeSelectedPlaylistItem`, `removeFilteredPlaylistItems`, `clearPlaylist`.
- **`recalculatePlaylist()`** — пересчёт `startOffset` всех строк от `playlistClockStart`,
  вставка авто-комментариев с датами при переходе через суточную границу (см. §5.3).

### 4.5 PROMO/FAST (ключевая бизнес-фича)
- `insertPromoFast(source:minimumIntervalMinutes:)` — рассыпает промо-ролики (из выделения
  или категории `promo`) по плейлисту с минимальным интервалом; сегментированные строки
  разворачиваются в физические; сброс счётчика на промо/live. Интервалы UI: 0/5/10/15 мин.
- `applyConverterPromoFast(...)` — в режиме конвертера **не вставляет**, а ставит теги `SCTE`
  на строки со storage `Promo_Fast*`/`reklama*` и `REKLAMA` на предыдущую реальную строку.

### 4.6 Редактирование метаданных
- `setGraphicTag(tag:isEnabled:)` (база), `setPlaylistGraphicLabel`/`addCustomGraphicToSelectedPlaylistItems`
  (плейлист), возрастные — взаимоисключающие.
- `setSelectedMedia{Name,ExternalID,Comment,ProductionYear,Director,Production,Genre,Synopsis}`.
- `addCategoryToSelected`/`removeCategoryFromSelected`, `addCustomGraphicToSelected`,
  `assignSequentialExternalIDs(startingAt:)` (напр. `CL100045` → +1 по выделению,
  `parseSequentialID` разбирает префикс+число+ширину).
- Все правки в базе **синхронно прокидываются в строки плейлиста** (единый `updateSelectedAssets`).

### 4.7 Файлы на диске (Storage/rename/relink)
- `requestRenameSelectedFilesToStorage` / `confirmRenameSelectedFilesToStorage` /
  `quietlyRenameSelectedFilesToStorage` — переименование файлов в имя = `externalID`.
- `synchronizeAllFilesToStorage` — массовая синхронизация имён + подтяжка ID из плейлиста.
- `makeDiskRenamePlan` — валидация (файл существует, нет коллизий/дублей целевых имён).
- `renameFilesToStorage` — `FileManager.moveItem`, обновление ассетов/плейлиста/маркеров,
  запись служебного лога, принудительное сохранение проекта.
- `relinkSelectedMedia`, `relinkMedia(assetID:to:)` (с тихим переименованием в Storage-имя),
  `relinkMediaLibraryRecursively(in:)` — рекурсивный релинк по индексу имён
  (точное имя → нормализованное; `canRelink` сверяет externalID/storage/name/filename).
- Всё логируется в `TV Assembly Service/Logs/{id,rename,relink,archive}/`.

### 4.8 Маркеры и сегменты
- `performMarkerAction(kind)`: в плейлисте IN/OUT → `trimSelectedPlaylistItem` (подрезка),
  иначе `addMarker`. `addMarker`/`removeSelectedMarker` в `markersByAssetPath[url.path]`.
- `markers(for:)`, `segments(for:)` (с кэшем), `segmentLabel(...)` («N из M» по границам),
  `qualityIssues(for:segment:)`.

### 4.9 Экспорт
- `exportCSV(preset:)`/`exportXML()`/`exportXLSX()` — одиночный или пакетный
  (`exportBatch*` по `playlistExportBatchesByAirDate`).
- `exportMediaCatalog{CSV,XML,XLSX}`.
- `splitPlaylistByAirDates` — только пересчёт (даты-комментарии уже расставляет recalc).
- `exportFile(...)` — обёртка с диалогом сохранения; после успеха может архивировать плейлист.

### 4.10 Проект / персистентность
- `saveProject`/`saveProjectAs`/`openProject`/`newProject`/`openLastProjectIfAvailable`.
- `writeProject` (JSONEncoder prettyPrinted+sortedKeys), `autosaveProject` (дебаунс 2с в
  `*.autosave.tvassembly`), `resetToNewProject`.
- `recordCurrentPlaylistToArchive`/`archiveCurrentPlaylist` — JSONL + лог.

### 4.11 Матчинг (важно для API импорта)
- `bestAssetMatch(for:hint)` + `matchScore` (точное=1000, вхождение=700, пересечение слов).
- `normalizedName` (см. `MarkerImporter`): lower, срез `ad_break_points`, `_`/`-`→пробел,
  выкидывание не-алфанумерики, слова через пробел.
- `onlinePlaylistAssetMatchIndex` — путь → id → storage-хинты → name/file-хинты.
- `applyPlaylistStorageMapping`, `applyMissingPlaylistTitle`, `resolveMissingPlaylistItems`,
  `learnAdBreakMarkersFromImportedPlaylist`, `mergeMarkers`, `moveMarkersIfNeeded`.

---

## 5. Ключевые алгоритмы (перенести дословно)

### 5.1 Проверка качества `buildPlaylistQualityReport`
Для каждой тайминговой строки формируются проблемы:
- **MEDIA (error):** файл не найден (не виртуальный и нет на диске) или Missing-заглушка.
- **DUR (error):** `asset.duration == nil`; **DUR (error):** `item.duration <= 0`.
- **READY (warn):** `asset.status != "Готово"` (кроме LIVE).
- **FPS (warn):** `|asset.frameRate - projectFps| > 0.2`.
- **AD (warn):** видео целиком (весь файл) длиннее **10 минут** без AD-точек
  (`shouldRequireAdMarkers`: `isWholeFile && duration > 600`).
- **RANGE (error):** сегмент `sourceIn < 0` или `sourceOut > assetDuration` (если файл доступен).
- **SHORT (warn):** сегмент короче **5 секунд**.
- **DUP (info):** совпадение `duplicateRangeKey` = `path|inFrame|outFrame` (по кадрам).

`statusTitle`: ERROR > WARN > (EMPTY при 0 строк) > OK. Converter-вариант проверяет только
нулевую длительность и некорректный диапазон.

### 5.2 Сегментация
`PlaylistSegmentation.segments` — см. §3. Кэш `playlistSegmentsCache` инвалидируется при
любом изменении (`invalidateDerivedPlaylistCaches`).

### 5.3 Пересчёт эфирного времени + суточные границы `recalculatePlaylist`
- `cursor = playlistClockStart`; если задана `playlistAirDate` — исходный список чистится от
  уже-существующих дата-комментариев, в начало ставится дата-строка.
- По каждой тайминговой строке `startOffset = cursor`, `cursor += duration`.
- При пересечении `nextDateBoundary` (кратно 24ч) вставляется новый дата-комментарий
  (`airDate + N дней`). Формат даты-комментария: `"DD MM YYYY"` (`isAirDateComment` =
  regex `^\d{2}\s\d{2}\s\d{4}$`).
- Пакетный экспорт (`playlistExportBatchesByAirDate`) режет плейлист по этим дата-строкам.

### 5.4 PROMO/FAST — см. §4.5 (счётчик `elapsedSinceLastPromo`, промо по кругу).

---

## 6. Форматы файлов (полная спецификация I/O)

### 6.1 Проект `.tvassembly` — JSON
Кодирование: pretty-printed, `sortedKeys`. Структура = `BroadcastProject`. Ассеты хранят
русские `kind` и enum-`rawValue` тегов. `markersByAssetPath` — словарь `путь → [AdMarker]`.

### 6.2 Экспорт плейлиста — универсальный CSV / XML / XLSX
Внутренняя строка `ExportRow`, колонки:
`#, Segment, Mode, Start, Duration, TC_IN, TC_OUT, Type, Storage, Graphics, File, Path, Markers, Note`.
- **Профиль universal/segments:** строка на каждый **сегмент**; в universal маркеры все,
  в segments — только попавшие в сегмент.
- **Профиль items:** одна строка на элемент.
- **CSV:** UTF-8, разделитель `,`, поля в кавычках (`""` эскейп), заголовок = склейка headers.
- **XML:** `<playlist app profile frameRate><item …><start/><duration/><sourceIn/><sourceOut/>
  <type/><storage/><graphics/><file/><path/><markers><marker type time>note</marker></markers>
  <markersText/><note/></item></playlist>`.
- **XLSX:** ручной OpenXML (Content_Types, _rels, workbook, sheet1 с `inlineStr` ячейками),
  упаковка через `/usr/bin/zip`.
- `Start` = **broadcastClockTimecode**, `Duration/TC_IN/TC_OUT` = timecode. `Graphics`/`Markers`
  — join через `"; "`.

### 6.3 Экспорт «ТЕЛЕ» (главный целевой формат эфирного автомата)
`TeleRow`, разделитель **`;`**, все поля в кавычках, перевод строки **CRLF**, кодировка
**CP1251** (лоссовая) либо UTF-8. Колонки:
`START TIME; NAME; TC_IN; DURATION; TC_OUT; STORAGE; LOGO; SCTE; REKLAMA`.
- `LOGO` = склейка через пробел из: `LOGO` (если есть), возрастной displayName («12+»), `SMOKE`.
- `SCTE` = `"SCTE_3"` если есть тег SCTE, иначе `"SCTE_0"`.
- `REKLAMA` = `"REKLAMA"` если тег есть, иначе пусто.
Реальный пример (`test batch/…TELE CP1251.csv`):
```
"START TIME";"NAME";"TC_IN";"DURATION";"TC_OUT";"STORAGE";"LOGO";"SCTE";"REKLAMA"
"06:00:00:00";"Ларва фаст/1/Копия Мороженое";"00:00:00:00";"00:01:43:00";"00:01:43:00";"LA1_EP001_….mp4";"LOGO 12+";"SCTE_0";""
```

### 6.4 Каталог медиа — CSV/XML/XLSX (`MediaCatalogExporter`)
Колонки `CatalogRow`:
`Media ID, Storage, Name, File, Path, Type, Duration, Duration Seconds, FPS, Size, Graphics,
Markers, Categories, Year, Director, Production, Genre, Synopsis, Comment, Status`.
Импорт каталога читает те же колонки (нормализация ключей = lower без пробелов), маркеры —
из строки `"AD BREAK HH:MM:SS:FF; …"`.

### 6.5 Импорт точек (`MarkerImporter`)
- **CSV:** авто-кодировка (utf8/utf16/CP1251/latin1), заголовки нормализуются. Либо
  колонка-сводка маркеров (`markers/ad_breaks/cue_points/…`) с парсингом `(LABEL) HH:MM:SS[:FF]`
  (IN/OUT отбрасываются как метки, всё → `adBreak`), либо предпочтительные тайм-колонки
  (`tc_in/timecode/time/…`). Хинт ассета — из колонок пути/имени или имени файла.
- **XML:** сначала пытается playlist-XML (`<item|clip|asset|media>` с `<marker|adBreak|break|
  point|cue time=…>`), иначе — regex по всему тексту.
- `parseTime`: число секунд, либо `HH:MM:SS` / `HH:MM:SS:FF` (кадры делятся на realFps),
  запятая→точка.

### 6.6 Архив эфира — JSONL (`broadcast-archive.jsonl`)
Дописывается построчно (`FileHandle.seekToEnd`). Одна строка = `BroadcastArchiveEntry`
(sortedKeys, даты ISO8601). Поля: `id, recordedAt, projectFile, airDate, playlistClockStart,
itemID, mediaID, rowKind, title, storage, fileName, path, startOffset, duration, sourceIn,
sourceOut, startTimecode, durationTimecode, sourceInTimecode, sourceOutTimecode, fps,
dimensions, graphics[], categories[], productionYear, director, production, genre, synopsis,
comment`. **Только тайминговые строки** (комментарии не пишутся). Плюс текстовый `.log`.

### 6.7 Импорт конвертера (`PlaylistConverterImporter`) — самый «умный» парсер
Вход: CSV/TXT/TSV (авто-разделитель `;`/`,`/`\t`, авто-кодировка), XML (regex по узлам
`event/item/playlistitem/program/clip/row`), XLSX (свой распаковщик через `/usr/bin/unzip` +
разбор sharedStrings и sheet1 побайтово). Колонки распознаются по **словарям синонимов**
(рус/англ): title/file/storage/start/duration/in/out/date/graphics/segments (см. константы
в конце файла). Умеет: сегменты из списка таймкодов, вывод длительности из соседних стартов,
парсинг дат в двух форматах, распознавание графики по значениям колонок (age/logo/smoke/scte/
reklama, truthy = on/yes/1/да/+/вкл).

### 6.8 Служебные логи
`TV Assembly Service/Logs/{id,rename,relink,archive}/{type}-YYYYMMDD-HHmmss.log`, TSV-таблицы
операций. Директория проекта определяется поиском вверх до `Package.swift` или `TV Assembly Service`.

---

## 7. UI / модель взаимодействия (`ContentView`)

Одно окно (min 1180×760). Верхний тулбар: меню Импорт/Проект/База/Экспорт/FPS/Режим.
5 рабочих режимов (`WorkspaceMode`): **assembly** (превью+плейлист), **adBreakPoints**
(крупное превью + аудиометр + маркеры), **mediaPrep** (`MediaPrepWorkspace` — карточка
метаданных), **playlistConverter** (офлайн), **broadcastArchive** (инфо об архиве).
Левая колонка — «База файлов» (поиск + фильтр категории + список с drag&drop).
Плейлист — таблица строк/сегментов с панелью качества, поиском, drag&drop, контекст-меню.
Клавиши: ⌘A/⌘C/⌘V/Delete, ←/→ (шаг кадра превью), ⌘N/O/S, ⌘I, ⌘⇧M.
Компоненты: `PreviewView` (AVPlayer), `AudioMeterView`, `MarkerPanel`, `SelectionInspectorPanel`,
`PlaylistQualityPanel`, `PlaylistPropertiesWindow`, `AirDatePickerPopover` и др.

---

## 8. Предлагаемая целевая архитектура (BE Linux + API + браузер)

### 8.1 Backend (рекомендую Go или TypeScript/Node)
Портировать как **чистые модули** (без изменений в логике):
`timecode`, `segmentation`, `quality`, `recalculate`, `promoFast`, `matching/normalizeName`,
`exporters` (universal CSV/XML/XLSX + TELE CP1251/UTF8), `catalog`, `archive`,
все `importers` (marker/catalog/converter/online).
Платформенные замены:
- Медиа-анализ → **`ffprobe`** (duration/size/fps/kind), очередь задач.
- XLSX → zip-библиотека языка (не внешний процесс).
- Хранилище → **Postgres** (assets, playlists, playlist_items, markers, projects, archive) +
  экспорт/импорт `.tvassembly` JSON для совместимости. `UserDefaults` → таблица настроек.
- CP1251 → `iconv`/`windows-1251` кодек. **Сохранить CRLF и `;`** для TELE.
- Файлы на NAS: операции rename/relink/sync выполняет backend по путям NAS, с теми же
  проверками коллизий и записью логов; служебные логи/архив — в БД и/или на диск.
- Автосейв → серверный дебаунс или явный save; WebSocket для статуса анализа.

### 8.2 API (эскиз REST + WS)
```
Проекты:   GET/POST/PUT /projects, POST /projects/{id}/save, /open, /new
Медиа:     GET /projects/{id}/assets, POST /assets/import (upload/paths),
           POST /assets/{id}/analyze, PATCH /assets/{id} (метаданные/теги/категории),
           POST /assets/reindex, POST /assets/relink, /rename-to-storage, /sync
Плейлист:  GET /playlist, POST /playlist/items (add/insert), PATCH /items/{id} (trim/graphics),
           POST /playlist/move, DELETE /items, POST /playlist/promo-fast,
           POST /playlist/comment, POST /playlist/live-break, POST /playlist/air-date
Маркеры:   GET/POST/DELETE /assets/{id}/markers
Проверка:  GET /playlist/quality  (кэшируемый отчёт)
Импорт:    POST /import/markers, /import/catalog, /import/converter, /import/online-playlist
Экспорт:   POST /export/playlist?format=csv|xml|xlsx&preset=…&mode=single|batch
           POST /export/catalog?format=…
Архив:     POST /archive/record, GET /archive?filter=…
WS:        /ws  (прогресс анализа, автосейв, статус-строка, инвалидация отчёта качества)
```

### 8.3 Frontend (React/Vue)
Три-панельная раскладка = порт assembly-режима: слева медиатека, сверху превью
(`<video>` + WebAudio-метр вместо AVPlayer), снизу таблица плейлиста с сегментами, панелью
качества и drag&drop. Режимы = вкладки. Состояние — серверное (стор клиента — проекция).
Таймкоды/сегменты приходят посчитанными с backend (единый источник правды по TC).

### 8.4 Порядок реализации
1. Доменные типы + `timecode` + `segmentation` + `quality` + `recalculate` (+ юнит-тесты,
   у оригинала есть `TimecodeTests.swift`).
2. Экспортёры (в первую очередь **TELE CP1251** — это то, ради чего всё) + импорт конвертера.
3. Хранилище (Postgres) + CRUD плейлиста/базы + API.
4. Медиа-анализ через ffprobe + очередь + WS-прогресс.
5. Frontend: медиатека → плейлист → превью → маркеры → экспорт.
6. PROMO/FAST, air-date split, архив эфира, rename/relink/sync по NAS.

---

## 9. Тонкие места (не потерять при порте)

- **Nominal vs real fps** в таймкодах (§3) — иначе разъедутся drop-frame форматы.
- **`isNonTimingRow`** строки не двигают `cursor`; дата-комментарии авто-генерятся и не
  должны попадать в архив/экспорт как контент.
- **Русские `rawValue`** у `MediaKind`/`GraphicTag` пишутся в файлы — сохранить или мигрировать.
- **TELE:** `;`-разделитель, кавычки, **CRLF**, **CP1251 (lossy)**, маппинг SCTE_3/SCTE_0.
- **Дедупликация точек** с толерансом 0.01–0.04с в разных местах — воспроизвести пороги.
- **Матчинг по нормализованным именам** (срез `ad_break_points`, схлопывание разделителей).
- **AD-warning** только для целых файлов > 10 мин; **SHORT** < 5с; **FPS** дельта > 0.2.
- Storage-имя файла = `externalID`; rename/relink синхронизируют пути в ассетах, плейлисте и
  ключах `markersByAssetPath` одновременно.
