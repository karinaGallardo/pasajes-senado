package worker

import (
	"context"
	"log"
	"sync"
)

// Job representa una tarea que se puede ejecutar en segundo plano.
type Job interface {
	Run(ctx context.Context) error
	Name() string
}

// Pool maneja la ejecución de trabajos en un número limitado de workers.
type Pool struct {
	jobQueue   chan Job
	maxWorkers int
	wg         sync.WaitGroup
	quit       chan bool
}

var (
	instance *Pool
	once     sync.Once
)

// GetPool retorna la instancia única del pool del sistema.
func GetPool() *Pool {
	once.Do(func() {
		// Por defecto 5 workers y cola de 100 tareas
		instance = NewPool(5, 100)
	})
	return instance
}

// NewPool crea un nuevo pool de workers.
func NewPool(maxWorkers int, queueSize int) *Pool {
	return &Pool{
		jobQueue:   make(chan Job, queueSize),
		maxWorkers: maxWorkers,
		quit:       make(chan bool),
	}
}

// Start inicia los workers.
func (p *Pool) Start(ctx context.Context) {
	log.Printf("[WorkerPool] Iniciando pool con %d workers", p.maxWorkers)
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.work(i, ctx)
	}
}

// Submit agrega un trabajo a la cola.
func (p *Pool) Submit(job Job) {
	select {
	case p.jobQueue <- job:
		// Trabajo encolado
	default:
		log.Printf("[WorkerPool] ERROR: Cola llena, descartando tarea: %s", job.Name())
	}
}

// Stop detiene todos los workers.
func (p *Pool) Stop() {
	close(p.quit)
	p.wg.Wait()
}

func (p *Pool) work(id int, ctx context.Context) {
	defer p.wg.Done()

	log.Printf("[WorkerPool] Worker %d listo", id)

	for {
		select {
		case job := <-p.jobQueue:
			log.Printf("[WorkerPool] Worker %d procesando: %s", id, job.Name())

			// Ejecutamos con recover para que un pánico en la tarea no mate al worker
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[WorkerPool] CRITICAL: Worker %d recuperado de pánico en %s: %v", id, job.Name(), r)
					}
				}()

				if err := job.Run(ctx); err != nil {
					log.Printf("[WorkerPool] ERROR: Worker %d falló en %s: %v", id, job.Name(), err)
				}
			}()

		case <-p.quit:
			log.Printf("[WorkerPool] Worker %d cerrando", id)
			return
		case <-ctx.Done():
			log.Printf("[WorkerPool] Worker %d cancelado por contexto", id)
			return
		}
	}
}
