# URL Kısaltma Servisi

Bu servis, uzun URL'leri kısa kodlara dönüştüren ve bu kodları kullanarak orijinal URL'lere yönlendirme yapan bir REST API servisidir.

## Özellikler

- URL kısaltma
- Kısa koddan orijinal URL'ye yönlendirme
- PostgreSQL ile kalıcı depolama
- Redis ile önbellekleme
- RESTful API
- Graceful shutdown

## API Endpoints

### URL Oluşturma

```http
POST /api/v1/urls
Content-Type: application/json

{
    "url": "https://example.com/very/long/url"
}
```

Başarılı yanıt:
```json
{
    "success": true,
    "data": {
        "short_code": "abc123",
        "created_at": "2024-02-17T02:03:41Z",
        "expires_at": "2024-02-18T02:03:41Z",
        "original_url": "https://example.com/very/long/url"
    }
}
```

### URL Getirme

```http
GET /api/v1/urls/{shortCode}
Accept: application/json
```

Başarılı yanıt:
```json
{
    "success": true,
    "data": {
        "original_url": "https://example.com/very/long/url",
        "short_code": "abc123",
        "accessed_at": "2024-02-17T02:03:41Z"
    }
}
```

## Kurulum

1. Gereksinimleri yükleyin:
   - Docker
   - Docker Compose

2. Servisi başlatın:
```bash
docker compose up --build
```

3. Servisi test edin:
```bash
# URL oluştur
curl -X POST -H "Content-Type: application/json" \
     -d '{"url":"https://example.com"}' \
     http://localhost:8080/api/v1/urls

# URL getir
curl -H "Accept: application/json" \
     http://localhost:8080/api/v1/urls/{shortCode}
```

## Mimari

- **Storage Layer**: PostgreSQL (kalıcı depolama) ve Redis (önbellekleme)
- **Service Layer**: İş mantığı ve URL işlemleri
- **API Layer**: HTTP endpoints ve request/response işlemleri

## Geliştirme

1. Go modülünü başlat:
```bash
go mod init LinkApp
```

2. Bağımlılıkları yükle:
```bash
go mod tidy
```

3. Testleri çalıştır:
```bash
go test ./...
```

## Katkıda Bulunma

1. Bu repository'yi fork edin
2. Yeni bir feature branch oluşturun
3. Değişikliklerinizi commit edin
4. Pull request gönderin

## Lisans

MIT