# Falabella — Image Transcoding

Falabella's Seller Center API requires images to be served as JPEG. When source images are stored
in other formats (PNG, WebP, etc.) the transcode proxy converts them on the fly before Falabella
fetches them.

---

## How It Works

The transcode proxy is exposed as a public HTTP endpoint:

```
GET /falabella/images/transcoded?src=<url>
```

### Processing Steps

1. Validate `src` against `FALABELLA_PRODUCT_IMAGE_TRANSCODE_ALLOWED_PREFIXES`. Requests with
   URLs not matching any allowed prefix are rejected with `400 Bad Request`.
2. Fetch the source image from the `src` URL with a configurable timeout.
3. Decode the image. Supported input formats: JPEG, PNG, WebP.
4. Re-encode as JPEG at quality 90.
5. Respond with `Content-Type: image/jpeg` and the transcoded byte stream.

---

## Integration with the Sync Pipeline

When `FALABELLA_PRODUCT_IMAGE_TRANSCODE_ENABLED=true`, the sync pipeline replaces every image URL
in the product gallery with the transcode proxy URL before submitting the image feed to Falabella:

```
Original URL:   https://cdn.flockstore.co/assets/abc.png
Transcode URL:  https://<TRANSCODE_PUBLIC_BASE_URL>/falabella/images/transcoded?src=https%3A%2F%2Fcdn.flockstore.co%2Fassets%2Fabc.png
```

Falabella then fetches the transcode URL and receives a valid JPEG.

---

## Security

`FALABELLA_PRODUCT_IMAGE_TRANSCODE_ALLOWED_PREFIXES` is a comma-separated list of URL prefixes
(e.g. `https://cdn.flockstore.co,https://assets.flock.co`). Only source URLs that match one of
these prefixes are accepted. This prevents the proxy from being used as an open redirect or SSRF
vector.

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `FALABELLA_PRODUCT_IMAGE_TRANSCODE_ENABLED` | `false` | Enable the transcode pipeline |
| `FALABELLA_PRODUCT_IMAGE_TRANSCODE_PUBLIC_BASE_URL` | `""` | Base URL of this Mannaiah instance (used to construct the proxy URL in image feeds) |
| `FALABELLA_PRODUCT_IMAGE_TRANSCODE_ALLOWED_PREFIXES` | `""` | Comma-separated allowed URL prefixes |
| `FALABELLA_PRODUCT_IMAGE_TRANSCODE_TIMEOUT_MS` | `15000` | Source image fetch timeout (ms) |
| `FALABELLA_PRODUCT_IMAGE_BASE_URL` | `""` | Base URL prepended to asset keys when building full image URLs for the feed |
