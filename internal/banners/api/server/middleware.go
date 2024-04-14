package server

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/Leopold1975/banners_control/pkg/logger"
)

func loggingMiddleware(logg logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rr := httptest.NewRecorder()

			defer func() {
				latency := time.Since(start).String()

				logg.Infof("METHOD %s URI %s %s	STATUS %d Latency %s Client IP %s User Agent %s",
					r.Method,
					r.Proto,
					r.URL.RequestURI(),
					rr.Code,
					latency,
					r.RemoteAddr,
					r.UserAgent(),
				)
			}()

			next.ServeHTTP(rr, r)

			for k, v := range rr.Header() {
				w.Header()[k] = v
			}

			w.WriteHeader(rr.Code)

			if rr.Code >= 400 && rr.Body.Len() != 0 {
				logg.Errorf("error: %s", rr.Body)
			}

			_, err := rr.Body.WriteTo(w)
			if err != nil {
				logg.Errorf("middleware write error: %w", err)
			}
		})
	}
}
