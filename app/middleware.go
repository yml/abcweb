package app

// initMiddleware enables useful middleware for the router.
import (
	"github.com/nullbio/abcweb/middleware"
	chimiddleware "github.com/pressly/chi/middleware"
)

// See https://github.com/pressly/chi#middlewares for additional middleware.
func (s State) InitMiddleware() {
	m := middleware.Middleware{
		Log: s.Log,
	}

	// Use zap logger for all routing
	s.Router.Use(m.Zap)

	// Graceful panic recovery that uses zap to log the stack trace
	s.Router.Use(m.Recover)

	// Puts a ceiling on the number of concurrent requests.
	// throttle := chimiddleware.Throttle(100)
	// s.router.Use(throttle)

	// Timeout is a middleware that cancels ctx after a given timeout and return
	// a 504 Gateway Timeout error to the client.
	// Generally readTimeout and writeTimeout is all that is required for timeouts.
	// timeout := chimiddleware.Timeout(time.Second * 30)
	// s.router.Use(timeout)

	// Strip and redirect slashes on routing paths
	s.Router.Use(chimiddleware.StripSlashes)

	// More available middleware. Uncomment to enable:

	// Injects a request ID into the context of each request
	// s.Router.Use(chimiddleware.RequestID)

	// Heartbeat is a monitoring endpoint to check the servers pulse.
	// route := chimiddleware.Route("/ping")
	// s.router.Use(route)

	// Sets a http.Request's RemoteAddr to either X-Forwarded-For or X-Real-IP
	// s.router.Use(chimiddleware.RealIP)

	// Signals to the request context when a client has closed their connection.
	// It can be used to cancel long operations on the server when the client
	// disconnects before the response is ready.
	// s.router.Use(chimiddleware.CloseNotify)

	// Sets response headers to prevent clients from caching
	if s.Config.AssetsNoCache {
		s.Router.Use(chimiddleware.NoCache)
	}

	// Enable CORS.
	// Configuration documentation at: https://godoc.org/github.com/goware/cors
	// Note: If you're getting CORS related errors you may need to adjust the
	// default settings by calling cors.New() with your own cors.Options struct.
	// s.Router.Use(cors.Default().Handler)
}
