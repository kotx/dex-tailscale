package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

var endpoint = flag.String("endpoint", "http://dex:5556", "the host to proxy requests to")

func main() {
	flag.Parse()
	if *endpoint == "" {
		log.Fatalf("endpoint must be set")
	}
	endpointUrl, err := url.Parse(*endpoint)
	if err != nil {
		log.Fatalf("endpoint must be a valid url: %v", err)
	}

	log.Printf("proxying requests to %s", endpointUrl)

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		req.Host = endpointUrl.Host
		req.URL.Host = endpointUrl.Host
		req.URL.Scheme = endpointUrl.Scheme
		req.RequestURI = ""

		slog.Info("proxying", slog.Group("request",
			"method", req.Method,
			"url", req.URL,
			"remote_addr", req.RemoteAddr,
		))

		res, err := httpClient.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = fmt.Fprint(w, err)
			slog.Error("proxying request", "err", err)
			return
		}

		for key, value := range req.Header {
			if strings.HasPrefix(http.CanonicalHeaderKey("X-Remote-"), key) {
				continue
			}

			switch key {
			case http.CanonicalHeaderKey("Tailscale-User-Login"):
				slog.Debug("header", "tailscale-user-login", value)

				w.Header()["X-Remote-User-Id"] = value
				w.Header()["X-Remote-User-Email"] = value
			case http.CanonicalHeaderKey("Tailscale-User-Name"):
				slog.Debug("header", "tailscale-user-name", value)

				w.Header()["X-Remote-User"] = value
			default:
				w.Header()[key] = value
			}
		}

		w.WriteHeader(res.StatusCode)

		var buf []byte
		_, err = res.Body.Read(buf)
		if err != nil {
			slog.Error("reading response body", "err", err)
		}
		defer res.Body.Close()

		w.Write(buf)
	})
	http.ListenAndServe(":8080", nil)
}
