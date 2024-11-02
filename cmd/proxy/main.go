package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
<<<<<<< HEAD

	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

var tsHost = flag.String("hostname", "dex", "hostname to use in the tailnet")
var endpoint = flag.String("endpoint", "http://dex:5556", "the Dex host to proxy requests to")
=======
)

var endpoint = flag.String("endpoint", "http://dex:5556", "the host to proxy requests to")
>>>>>>> master

func main() {
	flag.Parse()
	if *endpoint == "" {
		log.Fatalf("endpoint must be set")
	}
	endpointUrl, err := url.Parse(*endpoint)
	if err != nil {
		log.Fatalf("endpoint must be a valid url: %v", err)
	}

<<<<<<< HEAD
	serve := new(tsnet.Server)
	serve.Hostname = *tsHost
	defer serve.Close()

	ln, err := serve.ListenFunnel("tcp", ":443")
	if err != nil {
		log.Fatalf("error listening on funnel: %v", err)
	}

	lc, err := serve.LocalClient()
	if err != nil {
		log.Fatalf("error creating tailscale local client: %v", err)
	}

=======
>>>>>>> master
	log.Printf("proxying requests to %s", endpointUrl)

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: time.Minute,
	}

<<<<<<< HEAD
	log.Fatal(http.Serve(ln,
		http.HandlerFunc(
			func(writer http.ResponseWriter, req *http.Request) {
				slog.Info("proxying", slog.Group("request",
					"method", req.Method,
					"url", req.URL,
					"remote_addr", req.RemoteAddr,
				))

				who, err := lc.WhoIs(req.Context(), req.RemoteAddr)
				if err != nil && err != tailscale.ErrPeerNotFound {
					slog.Error("tailscale whois", "err", err)
					http.Error(writer, err.Error(), 500)
					return
				}
				slog.Debug("tailscale", "whois", who)

				req.Host = endpointUrl.Host
				req.URL.Host = endpointUrl.Host
				req.URL.Scheme = endpointUrl.Scheme
				req.RequestURI = ""

				for key, value := range req.Header {
					if strings.HasPrefix(http.CanonicalHeaderKey("X-Remote-"), key) {
						slog.Info("removing spoofed header", "key", key, "value", value)
						req.Header.Del(key)
					}
				}

				if who != nil {
					req.Header.Set("X-Remote-User-Email", who.UserProfile.LoginName)
					req.Header.Set("X-Remote-User", who.UserProfile.DisplayName)
					req.Header.Set("X-Remote-User-Id", who.UserProfile.ID.String())
				}

				res, err := httpClient.Do(req)
				if err != nil {
					var urlError *url.Error
					if errors.As(err, &urlError) && urlError.Timeout() {
						writer.WriteHeader(http.StatusGatewayTimeout)
					} else {
						writer.WriteHeader(http.StatusBadGateway)
					}

					_, _ = fmt.Fprint(writer, err)
					slog.Error("proxying request", "err", err)
					return
				}

				for key, value := range res.Header {
					writer.Header()[key] = value
				}
				writer.WriteHeader(res.StatusCode)

				_, err = io.Copy(writer, res.Body)
				defer res.Body.Close()
				if err != nil {
					slog.Error("reading response body", "err", err)
				}
			})))
=======
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
			var urlError *url.Error
			if errors.As(err, &urlError) && urlError.Timeout() {
				writer.WriteHeader(http.StatusGatewayTimeout)
			} else {
				writer.WriteHeader(http.StatusBadGateway)
			}

			_, _ = fmt.Fprint(writer, err)
			slog.Error("proxying request", "err", err)
			return
		}

		for key, value := range res.Header {
			writer.Header()[key] = value
		}
		writer.WriteHeader(res.StatusCode)

		_, err = io.Copy(writer, res.Body)
		defer res.Body.Close()
		if err != nil {
			slog.Error("reading response body", "err", err)
		}
	})
	http.ListenAndServe(":8080", nil)
>>>>>>> master
}
