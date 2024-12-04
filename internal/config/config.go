package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gomiddleware/realip"
	_ "github.com/lib/pq"
	"golang.org/x/time/rate"
)

// Config Update the config struct to hold the SMTP server settings.
type Config struct {
	Port            string
	Env             string
	CtxTimeout      time.Duration
	JWTDuration     time.Duration
	PhrasePause     int
	AudioPattern    int
	MaxNumPhrases   int
	TTSBasePath     string
	PrivateKeyPath  string
	FileUploadLimit int64
	Db              struct {
		Dsn          string
		MaxOpenConns int
		MaxIdleConns int
		MaxIdleTime  string
	}
	Limiter struct {
		Enabled bool
		Rps     float64
		Burst   int
	}
}

func SetConfigs(config *Config) error {
	// get port and debug from commandline flags... if not present use defaults
	flag.StringVar(&config.Port, "port", "8080", "API server port")

	flag.StringVar(&config.Env, "env", "development", "Environment (development|staging|cloud)")
	flag.DurationVar(&config.CtxTimeout, "ctx-timeout", 3*time.Second, "Context timeout for db queries in seconds")

	flag.StringVar(&config.Db.Dsn, "db-dsn", "", "PostgreSQL DSN")

	flag.IntVar(&config.Db.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&config.Db.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&config.Db.MaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.BoolVar(&config.Limiter.Enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Float64Var(&config.Limiter.Rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&config.Limiter.Burst, "limiter-burst", 4, "Rate limiter maximum burst")

	flag.StringVar(&config.TTSBasePath, "tts-base-path", "/tmp/audio/", "text-to-speech base path temporary storage of mp3 audio files")

	flag.DurationVar(&config.JWTDuration, "jwt-duration", 24, "JWT duration in hours")
	flag.Int64Var(&config.FileUploadLimit, "upload-size-limit", 8*8000, "File upload size limit in KB (default is 8)")
	flag.IntVar(&config.PhrasePause, "phrase-pause", 4, "Pause in seconds between phrases (must be between 3 and 10)'")
	flag.IntVar(&config.MaxNumPhrases, "maximum-number-phrases", 100, "Maximum number of phrases to be turned into audio files")
	flag.IntVar(&config.AudioPattern, "audio-pattern", 2, "Audio pattern to be used in constructing mp3's {1: standard, 2: advanced, 3: review}")

	if !isValidPause(config.PhrasePause) {
		return errors.New("invalid pause value (must be between 3 and 10)")
	}
	// PrivateKey is an ECDSA private key which was generated with the following
	// command:
	//	openssl ecparam -name prime256v1 -genkey -noout -out ecprivatekey.pem
	flag.StringVar(&config.PrivateKeyPath, "private-key-path", "../ecprivatekey.pem", "EcdsaPrivateKey for jws authenticator")

	return nil
}

func isValidPause(port int) bool {
	return port >= 3 && port <= 10
}

func (cfg *Config) RateLimit(next http.Handler) http.Handler {
	// Define a client struct to hold the rate limiter and last seen time for each
	// client.
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu sync.Mutex
		// Update the map so the values are pointers to a client struct.
		clients = make(map[string]*client)
	)

	// Launch a background goroutine which removes old entries from the clients map once
	// every minute.
	go func() {
		for {
			time.Sleep(time.Minute)

			// Lock the mutex to prevent any rate limiter checks from happening while
			// the cleanup is taking place.
			mu.Lock()

			// Loop through all clients. If they haven't been seen within the last three
			// minutes, delete the corresponding entry from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			// Importantly, unlock the mutex when the cleanup is complete.
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.Limiter.Enabled {
			// Use the realip.FromRequest() function to get the client's real IP address.
			ip := realip.RealIpFromRequest(r)

			mu.Lock()

			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(cfg.Limiter.Rps), cfg.Limiter.Burst),
				}
			}

			clients[ip].lastSeen = time.Now()

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				rateLimitExceededResponse(w, r)
				return
			}

			mu.Unlock()
		}

		next.ServeHTTP(w, r)
	})
}

func rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	type envelope map[string]interface{}
	env := envelope{"error": message}

	js, err := json.MarshalIndent(env, "", "\t")
	if err != nil {
		fmt.Printf("error in rateLimitExceededResponse: %s, %s, %s", err, r.Method, r.URL.String())
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_, err = w.Write(js)
	if err != nil {
		fmt.Printf("error in rateLimitExceededResponse Write: %s, %s, %s", err, r.Method, r.URL.String())
	}
}

func (cfg *Config) OpenDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Db.Dsn)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of open (in-use + idle) connections in the pool. Note that
	// passing a value less than or equal to 0 will mean there is no limit.
	db.SetMaxOpenConns(cfg.Db.MaxOpenConns)

	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.Db.MaxIdleConns)

	// Use the time.ParseDuration() function to convert the idle timeout duration string
	// to a time.Duration type.
	duration, err := time.ParseDuration(cfg.Db.MaxIdleTime)
	if err != nil {
		return nil, err
	}

	// Set the maximum idle timeout.
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.CtxTimeout)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
