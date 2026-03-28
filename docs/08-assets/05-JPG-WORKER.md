# Assets — JPEG Conversion Worker

The JPG worker is a background service that converts non-JPEG image assets to JPEG format.
Its primary purpose is to reduce storage and bandwidth costs by normalising uploaded images
(PNG, WebP, JPEG with wrong extension) to a single compressed format.

---

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ASSETS_JPG_WORKER_ENABLED` | `false` | Enable the scheduled worker |
| `ASSETS_JPG_WORKER_CRON` | `0 * * * *` | Cron schedule (default: every hour) |
| `ASSETS_JPG_WORKER_TAGS` | `""` | Comma-separated tag names; only assets matching one of these tags are eligible |
| `ASSETS_JPG_WORKER_BATCH_SIZE` | `100` | Maximum assets processed per tick |
| `ASSETS_JPG_WORKER_JPEG_QUALITY` | `90` | JPEG encoder quality (1–100) |
| `ASSETS_JPG_WORKER_TIMEOUT_MS` | `300000` | Per-tick maximum duration (5 minutes) |

The worker silently disables itself even when `ENABLED=true` if `TAGS` or `CRON` are not
configured.

---

## Trigger Paths

| Trigger | How |
|---------|-----|
| **Scheduled** | Cron job registered during `module.Start(ctx)`. Deregistered on `Stop(ctx)`. |
| **Manual HTTP** | `POST /assets/workers/jpg/run` (permission: `product:manage`). Query params override config values. |
| **External module** | The Falabella module calls `module.Service().RunJPGWorker(ctx, cmd)` directly through the exposed service accessor. |

---

## Eligibility Criteria

An asset is eligible for conversion in the current tick if:

1. It has **at least one tag** matching the configured tag list.
2. It is **not already JPEG** — the asset is considered JPEG only if ALL three conditions
   hold simultaneously:
   - `mime_type = "image/jpeg"` (or `"image/jpg"`)
   - `key` suffix is `.jpg`
   - `original_name` suffix is `.jpg`

The SQL query targeting eligible assets:

```sql
SELECT assets.*
FROM assets
JOIN asset_tags ON asset_tags.asset_id = assets.id
WHERE asset_tags.name IN ('web', 'promo')
  AND NOT (
      (assets.mime_type = 'image/jpeg' OR assets.mime_type = 'image/jpg')
      AND assets.key LIKE '%.jpg'
      AND assets.original_name LIKE '%.jpg'
  )
  AND assets.deleted_at IS NULL
GROUP BY assets.id
ORDER BY assets.updated_at ASC, assets.id ASC
LIMIT <batchSize>
```

Ordering `ASC` by `updated_at` ensures least-recently-touched assets are processed first.

---

## Per-Asset Conversion Pipeline

For each eligible asset in the batch:

```
Step 1: Acquire keyed lock "asset:<id>"
            Prevents concurrent modifications during conversion.

Step 2: Reload from DB
            Verifies the asset has not been converted since the batch query ran.
            If already JPEG: skip (counted as "skipped").

Step 3: Download binary
            storage.Download(asset.Key) → []byte

Step 4: Decode image
            image.Decode(rawBytes)
            Supported formats:
              - JPEG  (stdlib image/jpeg)
              - PNG   (stdlib image/png)
              - WebP  (golang.org/x/image/webp)
            Unrecognised format → error → failure counter +1, continue to next asset.

Step 5: Encode to JPEG
            jpeg.Encode(buffer, decoded, &Options{Quality: q})
            Quality is clamped to [1, 100], default 90.

Step 6: Build new key and name
            newKey = replaceExtension(asset.Key, ".jpg")
            newOriginalName = replaceExtension(asset.OriginalName, ".jpg")

Step 7: Upload new binary
            storage.Upload(newKey, "image/jpeg", jpgBytes)
            On failure → skip rollback (nothing changed in DB yet), count as failed.

Step 8: Update database record
            repository.UpdateBinary(id, {newKey, newOriginalName, "image/jpeg", len(jpgBytes)})
            On failure:
              - storage.Delete(newKey)     ← remove the orphan new object
              - count as failed

Step 9: Delete old binary (if key changed)
            storage.Delete(oldKey)
            On failure:
              - repository.UpdateBinary back to old values (rollback DB)
              - storage.Delete(newKey)     ← clean up new object
              - count as failed

Step 10: Publish "assets.v1.updated" event

Step 11: Release lock
         Count as "converted".
```

**Context cancellation** is checked before each asset starts processing. If the context is
cancelled (deadline exceeded), the worker returns partial results immediately.

---

## Result Counters

| Field | Meaning |
|-------|---------|
| `Scanned` | Total assets returned by the eligibility query |
| `Converted` | Successfully converted to JPEG |
| `Skipped` | Already JPEG (no-op) |
| `Failed` | Any error during steps 3–9 |

---

## Example: Tick Run for "web" Tag

```
[Cron tick: 2026-03-28 11:00:00]

Eligibility query: tag IN ('web'), batchSize=100
  → 47 assets returned (PNG + WebP)

Asset "a1b2c3d4" (logo.png, 320 KB PNG):
  ✅ Downloaded → decoded (PNG) → encoded (JPEG 90) = 28 KB
  ✅ Uploaded: assets/a1b2c3d4-logo.jpg
  ✅ DB updated: mimeType=image/jpeg, size=28672
  ✅ Old object deleted: assets/a1b2c3d4-logo.png
  ✅ Event published: assets.v1.updated

Asset "b5c6d7e8" (banner.webp, 1.2 MB WebP):
  ✅ Converted → 88 KB JPEG

Asset "ff001234" (photo.jpg, 240 KB):
  → Already JPEG (mime+key+name all match) → Skipped

Asset "cc002233" (diagram.pdf):
  → Decode failed (unsupported format) → Failed

Result: Scanned=47, Converted=44, Skipped=2, Failed=1
```

---

## Sync Recording

If a `SyncRecorder` is injected into the module, the worker calls:
1. `StartRun(ctx, "assets.jpg_conversion", trigger)` — returns a `runID`.
2. On success: `CompleteRun(ctx, runID, scanned, converted, failed, skipped)`.
3. On any fatal error: `FailRun(ctx, runID, ...)` with a `SyncError` slice.

The default recorder is a no-op. The Falabella module injects a real recorder so that
conversion runs appear in the global sync record history.
