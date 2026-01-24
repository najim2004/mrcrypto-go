package worker

import (
	"log"
	"sync"

	"mrcrypto-go/internal/model"
	"mrcrypto-go/internal/service"
)

type WorkerPool struct {
	workers  int
	jobs     chan string
	results  chan *model.Signal
	wg       sync.WaitGroup
	strategy *service.StrategyService
}

// NewPool creates a new worker pool
func NewPool(workers int, strategy *service.StrategyService) *WorkerPool {
	return &WorkerPool{
		workers:  workers,
		jobs:     make(chan string, 100),
		results:  make(chan *model.Signal, 100),
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
	defer p.wg.Done()

	for symbol := range p.jobs {
		log.Printf("⏳ [Worker %d] Processing %s...", id, symbol)
		signal, err := p.strategy.EvaluateSymbol(symbol)
		if err != nil {
			log.Printf("⚠️  [Worker %d] Error evaluating %s: %v", id, symbol, err)
			continue
		}

		if signal != nil {
			log.Printf("📈 [Worker %d] Signal found for %s!", id, symbol)
			p.results <- signal
		}
	}
	log.Printf("✅ [Worker %d] Completed all jobs", id)
}

// AddJob adds a symbol to the job queue
func (p *WorkerPool) AddJob(symbol string) {
	p.jobs <- symbol
}

// Wait closes the jobs channel and waits for all workers to finish
func (p *WorkerPool) Wait() []*model.Signal {
	log.Printf("⏳ [Worker Pool] Waiting for all workers to complete...")
	close(p.jobs)
	p.wg.Wait()
	close(p.results)

	// Collect all results
	signals := make([]*model.Signal, 0)
	for signal := range p.results {
		signals = append(signals, signal)
	}

	log.Printf("✅ [Worker Pool] All workers completed. Collected %d signals", len(signals))
	return signals
}
