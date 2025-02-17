package storage

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisCache(t *testing.T) {
	// Test için geçici bir Redis bağlantısı
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	// Redis bağlantısını kontrol et
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis bağlantısı kurulamadı, testi atlıyorum:", err)
	}

	cache := &RedisCache{client: client}
	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		key := "testKey"
		value := "testValue"
		expiration := time.Minute

		// Değeri cache'e kaydet
		err := cache.Set(ctx, key, value, expiration)
		if err != nil {
			t.Fatal("Cache'e kaydedilemedi:", err)
		}

		// Değeri cache'den getir
		got, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatal("Cache'den getirilemedi:", err)
		}

		if got != value {
			t.Errorf("Beklenen değer %s, alınan değer %s", value, got)
		}
	})

	t.Run("Get - Not Found", func(t *testing.T) {
		_, err := cache.Get(ctx, "nonexistent")
		if err != ErrNotFound {
			t.Errorf("Var olmayan key için ErrNotFound bekleniyor, alınan: %v", err)
		}
	})
} 