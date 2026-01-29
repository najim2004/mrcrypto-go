package worker

import (
	"fmt"
	"log"
	"sync"

	"mrcrypto-go/internal/model"
	"mrcrypto-go/internal/service"
)

type ScanResult struct {
	Signal *model.Signal
	Symbol string
	Price  float64
}

type WorkerPool struct {
	workers  int
	jobs     chan string
	results  chan ScanResult
	wg       sync.WaitGroup
	strategy *service.StrategyService
}

// NewPool creates a new worker pool
func NewPool(workers int, strategy *service.StrategyService) *WorkerPool {
	return &WorkerPool{
		workers:  workers,
		jobs:     make(chan string, 100),
		results:  make(chan ScanResult, 100),
		strategy: strategy,
	}
}

// Start launches the worker goroutines
func (p *WorkerPool) Start() {
	log.Printf("🔄 [Worker Pool] Starting %d workers...", p.workers)
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("✅ [Worker Pool] All %d workers started", p.workers)
}

// worker processes jobs from the jobs channel
func (p *WorkerPool) worker(id int) {
	// Critical: Add panic recovery to prevent entire pool crash
	defer service.RecoverAndLog(fmt.Sprintf("Worker %d", id))
	defer p.wg.Done()

	for symbol := range p.jobs {
		log.Printf("⏳ [Worker %d] Processing %s...", id, symbol)

		// Add individual job panic recovery
		func() {
			defer service.RecoverAndLog(fmt.Sprintf("Worker %d processing %s", id, symbol))

			signal, price, err := p.strategy.EvaluateSymbol(symbol)

			if err != nil {
				log.Printf("⚠️  [Worker %d] Error evaluating %s: %v", id, symbol, err)
				return
			}

			// Always report result (for Price monitoring)
			p.results <- ScanResult{
				Signal: signal,
				Symbol: symbol,
				Price:  price,
			}

			if signal != nil {
				log.Printf("📈 [Worker %d] Signal found for %s!", id, symbol)
			}
		}()
	}
	log.Printf("✅ [Worker %d] Completed all jobs", id)
}

// AddJob adds a symbol to the job queue
func (p *WorkerPool) AddJob(symbol string) {
	p.jobs <- symbol
}

// Wait closes the jobs channel and waits for all workers to finish
// Returns potential signals and a map of current prices for all scanned symbols
func (p *WorkerPool) Wait() ([]*model.Signal, map[string]float64) {
	log.Printf("⏳ [Worker Pool] Waiting for all workers to complete...")
	close(p.jobs)
	p.wg.Wait()
	close(p.results)

	// Collect all results
	signals := make([]*model.Signal, 0)
	prices := make(map[string]float64)

	for res := range p.results {
		if res.Price > 0 {
			prices[res.Symbol] = res.Price
		}
		if res.Signal != nil {
			signals = append(signals, res.Signal)
		}
	}

	log.Printf("✅ [Worker Pool] All workers completed. Collected %d signals, %d prices", len(signals), len(prices))
	return signals, prices
}
