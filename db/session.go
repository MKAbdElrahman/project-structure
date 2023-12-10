package db

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSessionStore represents the session store.
type PostgresSessionStore struct {
	pool        *pgxpool.Pool
	stopCleanup chan bool
}

func NewPostgresSessionStore(pool *pgxpool.Pool, cleanupInterval time.Duration) *PostgresSessionStore {
	p := &PostgresSessionStore{pool: pool}
	if cleanupInterval > 0 {
		go p.startCleanup(cleanupInterval)
	}
	return p
}

// Find returns the data for a given session token from the PostgresSessionStore instance.
// If the session token is not found or is expired, the returned exists flag will
// be set to false.
func (p *PostgresSessionStore) Find(token string) (b []byte, exists bool, err error) {
	row := p.pool.QueryRow(context.Background(), "SELECT data FROM sessions WHERE token = $1 AND current_timestamp < expiry", token)
	err = row.Scan(&b)
	if err == pgx.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// Commit adds a session token and data to the PostgresSessionStore instance with the
// given expiry time. If the session token already exists, then the data and expiry
// time are updated.
func (p *PostgresSessionStore) Commit(token string, b []byte, expiry time.Time) error {
	_, err := p.pool.Exec(context.Background(), "INSERT INTO sessions (token, data, expiry) VALUES ($1, $2, $3) ON CONFLICT (token) DO UPDATE SET data = EXCLUDED.data, expiry = EXCLUDED.expiry", token, b, expiry)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes a session token and corresponding data from the PostgresSessionStore
// instance.
func (p *PostgresSessionStore) Delete(token string) error {
	_, err := p.pool.Exec(context.Background(), "DELETE FROM sessions WHERE token = $1", token)
	return err
}

// All returns a map containing the token and data for all active (i.e.
// not expired) sessions in the PostgresSessionStore instance.
func (p *PostgresSessionStore) All() (map[string][]byte, error) {
	rows, err := p.pool.Query(context.Background(), "SELECT token, data FROM sessions WHERE current_timestamp < expiry")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make(map[string][]byte)

	for rows.Next() {
		var (
			token string
			data  []byte
		)

		err = rows.Scan(&token, &data)
		if err != nil {
			return nil, err
		}

		sessions[token] = data
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func (p *PostgresSessionStore) startCleanup(interval time.Duration) {
	p.stopCleanup = make(chan bool)
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			err := p.deleteExpired()
			if err != nil {
				log.Println(err)
			}
		case <-p.stopCleanup:
			ticker.Stop()
			return
		}
	}
}

// StopCleanup terminates the background cleanup goroutine for the PostgresSessionStore
// instance. It's rare to terminate this; generally PostgresSessionStore instances and
// their cleanup goroutines are intended to be long-lived and run for the lifetime
// of your application.
//
// There may be occasions though when your use of the PostgresSessionStore is transient.
// An example is creating a new PostgresSessionStore instance in a test function. In this
// scenario, the cleanup goroutine (which will run forever) will prevent the
// PostgresSessionStore object from being garbage collected even after the test function
// has finished. You can prevent this by manually calling StopCleanup.
func (p *PostgresSessionStore) StopCleanup() {
	if p.stopCleanup != nil {
		p.stopCleanup <- true
	}
}

func (p *PostgresSessionStore) deleteExpired() error {
	_, err := p.pool.Exec(context.Background(), "DELETE FROM sessions WHERE expiry < current_timestamp")
	return err
}
