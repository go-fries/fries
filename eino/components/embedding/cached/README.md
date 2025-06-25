# Cache Embedder for Eino

This module provides a cache embedder for Eino, which is designed to store and retrieve embeddings efficiently. The cache embedder can be used to speed up the embedding process by caching previously computed embeddings.

> [!NOTE]  
> An earlier version of this component has been donated to https://github.com/cloudwego/eino-ext/tree/main/components/embedding/cache.
>
> Currently, this is a branch of it, and there are some differences between the two versions.

## Installation

```shell
go get github.com/go-fries/fries/eino/components/embedding/cached/v3
```

## Usage


```go
package main

import (
	"context"
	"crypto/md5"
	"log"

	"github.com/cloudwego/eino/components/embedding"
	cachedredis "github.com/go-fries/fries/eino/components/embedding/cached/cacher/redis/v3"
	"github.com/go-fries/fries/eino/components/embedding/cached/v3"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// the original embedder, you can replace it with any other embedder implementation
	// It's only a example, you need to bring a real embedder implementation here.
	var originalEmbedder embedding.Embedder
	// embedder, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
	// 	APIKey:     accessKey,
	// }
	// ...

	embedder := cached.NewEmbedder(originalEmbedder,
		cached.WithCacher(cachedredis.NewCacher(rdb)),            // using Redis as the cache
		cached.WithGenerator(cached.NewHashGenerator(md5.New)), // using md5 for generating unique keys
	)
	
	embeddings, err := embedder.EmbedStrings(context.Background(), []string{"hello", "how are you"})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("embeddings: %v", embeddings)
}
```