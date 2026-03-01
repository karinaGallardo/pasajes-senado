package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
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
	running    atomic.Int32 // 1 activo, 0 apagándose
}

var (
	instance *Pool
	once     sync.Once
)

// GetPool retorna la instancia única del pool del sistema.
func GetPool() *Pool {
	once.Do(func() {
		// Por defecto 5 workers y cola de 200 tareas
		instance = NewPool(5, 200)
	})
	return instance
}

// NewPool crea un nuevo pool de workers.
func NewPool(maxWorkers int, queueSize int) *Pool {
	return &Pool{
		jobQueue:   make(chan Job, queueSize),
		maxWorkers: maxWorkers,
	}
}

// Start inicia los workers.
func (p *Pool) Start(ctx context.Context) {
	log.Printf("[WorkerPool] Iniciando pool con %d workers", p.maxWorkers)
	p.running.Store(1)

	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.work(i, ctx)
	}
}

// Submit agrega un trabajo a la cola.
func (p *Pool) Submit(job Job) {
	// Verificar si el pool está aceptando tareas
	if p.running.Load() == 0 {
		log.Printf("[WorkerPool] RECHAZADO: Apagado en curso, descartando: %s", job.Name())
		return
	}

	select {
	case p.jobQueue <- job:
		// Trabajo encolado
	default:
		log.Printf("[WorkerPool] ERROR: Cola llena, descartando tarea: %s", job.Name())
	}
}

// Stop detiene todos los workers de forma ordenada después de vaciar la cola.
func (p *Pool) Stop() {
	log.Println("[WorkerPool] Solicitando parada ordenada...")
	p.running.Store(0) // Dejar de aceptar nuevas tareas

	close(p.jobQueue) // Avisar a los workers que terminen lo pendiente
	p.wg.Wait()       // Esperar a que todos terminen
	log.Println("[WorkerPool] Pool detenido satisfactoriamente.")
}

func (p *Pool) work(id int, ctx context.Context) {
	defer p.wg.Done()

	log.Printf("[WorkerPool] Worker %d listo", id)

	// Range termina automáticamente cuando el canal se cierra y se vacía
	for job := range p.jobQueue {
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

		// Verificar si el contexto fue cancelado entre tareas
		select {
		case <-ctx.Done():
			log.Printf("[WorkerPool] Worker %d finalizado por cancelación de contexto", id)
			return
		default:
		}
	}

	log.Printf("[WorkerPool] Worker %d cerrando (cola vacía)", id)
}
