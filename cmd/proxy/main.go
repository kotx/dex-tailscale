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

	http.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
		req.Host = endpointUrl.Host
		req.URL.Host = endpointUrl.Host
		req.URL.Scheme = endpointUrl.Scheme
		req.RequestURI = ""

		slog.Info("proxying", slog.Group("request",
			"method", req.Method,
			"url", req.URL,
			"remote_addr", req.RemoteAddr,
		))

		for key, value := range req.Header {
			if strings.HasPrefix(http.CanonicalHeaderKey("X-Remote-"), key) {
				slog.Info("removing spoofed header", "key", key, "value", value)
				req.Header.Del(key)
			}

			switch key {
			case http.CanonicalHeaderKey("Tailscale-User-Login"):
				slog.Debug("header", "tailscale-user-login", value)

				req.Header["X-Remote-User-Id"] = value
				req.Header["X-Remote-User-Email"] = value
			case http.CanonicalHeaderKey("Tailscale-User-Name"):
				slog.Debug("header", "tailscale-user-name", value)

				req.Header["X-Remote-User"] = value
			default:
				slog.Debug("allowing header", "key", key)
			}
		}

		res, err := httpClient.Do(req)
		if err != nil {
			writer.WriteHeader(http.StatusBadGateway)
			_, _ = fmt.Fprint(writer, err)
			slog.Error("proxying request", "err", err)
			return
		}

		for key, value := range res.Header {
			writer.Header()[key] = value
		}
		writer.WriteHeader(res.StatusCode)

		var buf []byte
		_, err = res.Body.Read(buf)
		if err != nil {
			slog.Error("reading response body", "err", err)
		}
		defer res.Body.Close()

		writer.Write(buf)
	})
	http.ListenAndServe(":8080", nil)
}
