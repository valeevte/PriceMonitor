package scheduler

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeevte/PriceMonitor/internal/products"
)

// Config конфигурация планировщика
type Config struct {
	IntervalSeconds int
}

// Run запускает scheduler и блокирует выполнение, пока ctx не отменён.
// Используйте его если хотите управлять жизненным циклом через контекст.
func Run(ctx context.Context, db *pgxpool.Pool, cfg Config) {
	interval := time.Duration(cfg.IntervalSeconds) * time.Second
	if cfg.IntervalSeconds <= 0 {
		interval = 60 * time.Second
	}
	repo := products.NewRepository(db)
	rand.Seed(time.Now().UnixNano())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("scheduler: started, interval %v", interval)

	// выполнить один проход сразу
	updatePrices(ctx, repo)

	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler: stopping due to context cancelled")
			return
		case <-ticker.C:
			updatePrices(ctx, repo)
		}
	}
}

// Start — удобная обёртка для обратной совместимости.
// Она запускает scheduler в отдельной горутине и не блокирует вызов.
// NOTE: этот метод не привязан к внешнему контексту — для корректного graceful
// shutdown лучше вызывать Run с контролируемым ctx.
func Start(db *pgxpool.Pool, cfg Config) {
	ctx := context.Background()
	go Run(ctx, db, cfg)
}

func updatePrices(ctx context.Context, repo *products.Repository) {
	ids, err := repo.GetAllProductIDs(ctx)
	if err != nil {
		log.Printf("scheduler: failed to list product ids: %v", err)
		return
	}
	for _, id := range ids {
		// Если контекст отменён — завершаем цикл
		select {
		case <-ctx.Done():
			return
		default:
		}

		lastPrice, has, err := repo.GetLatestPrice(ctx, id)
		if err != nil {
			log.Printf("scheduler: failed to get latest price for id=%d: %v", id, err)
			continue
		}

		var newPrice float64
		if !has {
			newPrice = float64(100 + rand.Intn(900))
		} else {
			delta := (rand.Float64()*0.1 - 0.05) // +-5%
			newPrice = lastPrice * (1 + delta)
			newPrice = math.Round(newPrice*100) / 100
			if newPrice <= 0 {
				newPrice = lastPrice
			}
		}

		if err := repo.InsertPrice(ctx, id, newPrice); err != nil {
			log.Printf("scheduler: failed to insert price for id=%d: %v", id, err)
		} else {
			log.Printf("scheduler: product %d new price %.2f", id, newPrice)
		}
	}
}
