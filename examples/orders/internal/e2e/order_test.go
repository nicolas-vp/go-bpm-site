// Package e2e - процесс OrderApproval целиком: настоящие очередь, движок и PostgreSQL.
// База - из GO_BPM_TEST_DSN, без неё тесты пропускаются. Таблицы bpm_* тестовой базы очищаются!
package e2e

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolas-vp/go-bpm/engine"
	"github.com/nicolas-vp/go-bpm/migrations"
	"github.com/nicolas-vp/go-bpm/queue"

	"example.com/orders/internal/bpmgen"
	"example.com/orders/internal/bpmgen/services"
)

func TestSmallOrder(t *testing.T) {
	env := start(t, &fakeMailer{})

	id, err := env.engine.StartProcess(env.ctx, "OrderApproval",
		map[string]any{"amount": 500, "email": "client@example.com"}, engine.WithBusinessKey("order-1"))
	if err != nil {
		t.Fatal(err)
	}
	// маленький заказ идёт мимо согласования сразу к уведомлению
	env.waitStatus(id, engine.InstanceCompleted)
	if sent := env.mailer.Sent(); len(sent) != 1 || sent[0] != "client@example.com: Заказ order-1" {
		t.Errorf("sent = %v", sent)
	}
}

func TestBigOrderNeedsApproval(t *testing.T) {
	env := start(t, &fakeMailer{})

	id, err := env.engine.StartProcess(env.ctx, "OrderApproval",
		map[string]any{"amount": 5000, "email": "client@example.com"}, engine.WithBusinessKey("order-2"))
	if err != nil {
		t.Fatal(err)
	}
	// крупный заказ ждёт менеджера: задача в очереди группы managers, письма ещё нет
	task := env.waitTask(id, "Approve")
	if len(env.mailer.Sent()) != 0 {
		t.Fatal("письмо ушло до согласования")
	}
	if err := env.tasks.Claim(env.ctx, task.TaskUid, "bob", []string{"accounting"}); !errors.Is(err, engine.ErrNotCandidate) {
		t.Errorf("взял задачу не из своей группы: err = %v", err)
	}
	if err := env.tasks.Claim(env.ctx, task.TaskUid, "bob", []string{"managers"}); err != nil {
		t.Fatal(err)
	}
	if err := env.tasks.Complete(env.ctx, task.TaskUid, "bob", nil); err != nil {
		t.Fatal(err)
	}
	env.waitStatus(id, engine.InstanceCompleted)
	if len(env.mailer.Sent()) != 1 {
		t.Errorf("sent = %v", env.mailer.Sent())
	}
}

func TestMailerDown(t *testing.T) {
	env := start(t, &fakeMailer{err: errors.New("smtp: connection refused")})

	id, err := env.engine.StartProcess(env.ctx, "OrderApproval", map[string]any{"amount": 500})
	if err != nil {
		t.Fatal(err)
	}
	// у notifyClient retries="3": после трёх неудачных попыток экземпляр упал
	env.waitStatus(id, engine.InstanceFailed)
}

func TestInvalidOrder(t *testing.T) {
	env := start(t, &fakeMailer{})

	id, err := env.engine.StartProcess(env.ctx, "OrderApproval", map[string]any{"amount": 0})
	if err != nil {
		t.Fatal(err)
	}
	// CheckOrder отвергает пустой заказ: шаг повторяется SystemMaxAttempts раз, затем экземпляр падает
	env.waitStatus(id, engine.InstanceFailed)
}

// testEnv - движок процессов на тестовой базе.
type testEnv struct {
	t      *testing.T
	ctx    context.Context
	engine *engine.Engine
	tasks  *engine.UserTaskService
	reader *engine.HistoryReader
	mailer *fakeMailer
}

func start(t *testing.T, mailer *fakeMailer) *testEnv {
	t.Helper()
	ctx := context.Background()
	pool := testPool(t)
	log := slog.New(slog.DiscardHandler)
	on := func() bool { return true }

	svc := &services.Services{Mailer: mailer}
	processes := bpmgen.Processes(svc)
	q := queue.New(pool, queue.Options{PollInterval: 100 * time.Millisecond, Logger: log})
	e, err := engine.New(q, engine.Config{
		SystemMaxAttempts: 2,                     // упавший метод - быстро в failed
		RetryDelay:        50 * time.Millisecond, // повторы без долгих пауз
		History:           engine.HistoryConfig{ToDB: on},
		Logger:            log,
	}, processes, bpmgen.Queues(svc))
	if err != nil {
		t.Fatal(err)
	}
	q.Start(ctx)
	t.Cleanup(q.Stop)
	return &testEnv{
		t:      t,
		ctx:    ctx,
		engine: e,
		tasks:  engine.NewUserTaskService(pool, e),
		reader: engine.NewHistoryReader(pool, processes, bpmgen.Decisions),
		mailer: mailer,
	}
}

// testPool подключается к GO_BPM_TEST_DSN, применяет миграции и очищает таблицы движка.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("GO_BPM_TEST_DSN")
	if dsn == "" {
		t.Skip("GO_BPM_TEST_DSN не задан")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `truncate bpm_queue, bpm_parallel_sync, bpm_wait, bpm_history, bpm_rules_history,
		bpm_process_instance, bpm_user_task`)
	if err != nil {
		t.Fatal(err)
	}
	return pool
}

// waitStatus ждёт, пока экземпляр придёт в статус want; процесс идёт асинхронно, через очередь.
func (env *testEnv) waitStatus(id uuid.UUID, want string) {
	env.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		instance, err := env.reader.GetInstance(env.ctx, id)
		if err != nil {
			env.t.Fatal(err)
		}
		if instance.Status == want {
			return
		}
		if instance.Status != engine.InstanceRunning || time.Now().After(deadline) {
			env.t.Fatalf("экземпляр %s: статус %s, ждали %s", id, instance.Status, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitTask ждёт открытую задачу пользователя на элементе elementId.
func (env *testEnv) waitTask(id uuid.UUID, elementId string) engine.UserTask {
	env.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		tasks, err := env.tasks.ListTasks(env.ctx, engine.TaskFilter{InstanceUid: id})
		if err != nil {
			env.t.Fatal(err)
		}
		for _, task := range tasks {
			if task.ElementId == elementId {
				return task
			}
		}
		if time.Now().After(deadline) {
			env.t.Fatalf("экземпляр %s: нет задачи %s, открыты %+v", id, elementId, tasks)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// fakeMailer - заглушка services.Mailer. Обработчик вызывают воркеры очереди, поэтому под мьютексом.
type fakeMailer struct {
	err  error
	mu   sync.Mutex
	sent []string
}

func (m *fakeMailer) Send(ctx context.Context, to, subject, body string) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, to+": "+subject)
	return nil
}

func (m *fakeMailer) Sent() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.sent...)
}
