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

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		req.Host = endpointUrl.Host
		req.URL.Host = endpointUrl.Host
		req.URL.Scheme = endpointUrl.Scheme
		req.RequestURI = ""

		slog.Info("proxying", "method", req.Method, "url", req.URL, "remote_addr", req.RemoteAddr)

		res, err := http.DefaultClient.Do(req)
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
				w.Header()["X-Remote-User-Id"] = value
				w.Header()["X-Remote-User-Email"] = value
			case http.CanonicalHeaderKey("Tailscale-User-Name"):
				w.Header()["X-Remote-User"] = value
			default:
				w.Header()[key] = value
			}
		}

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