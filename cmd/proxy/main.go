package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

var logLevel = flag.String("logLevel", "INFO", "log level (DEBUG, INFO, WARN, ERROR)")
var tsHost = flag.String("hostname", "dex", "hostname to use in the tailnet")
var endpoint = flag.String("endpoint", "http://dex:5556", "the Dex host to proxy requests to")

func ParseLevel(s string) (slog.Level, error) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(s))
	return level, err
}

func main() {
	logLevel, err := ParseLevel(*logLevel)
	if err != nil {
		log.Fatal("error parsing log level: ", err)
	}
	slog.SetLogLoggerLevel(logLevel)

	flag.Parse()
	if *endpoint == "" {
		log.Fatalf("endpoint must be set")
	}
	endpointUrl, err := url.Parse(*endpoint)
	if err != nil {
		log.Fatalf("endpoint must be a valid url: %v", err)
	}

	serve := &tsnet.Server{
		Hostname: *tsHost,
		Dir:      "/state",
	}
	defer serve.Close()

	ln, err := serve.ListenFunnel("tcp", ":443")
	if err != nil {
		log.Fatalf("error listening on funnel: %v", err)
	}

	lc, err := serve.LocalClient()
	if err != nil {
		log.Fatalf("error creating tailscale local client: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(endpointUrl)
	log.Printf("proxying requests to %s", endpointUrl)

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
				for key, value := range req.Header {
					if strings.HasPrefix(http.CanonicalHeaderKey("X-Remote-"), key) {
						slog.Info("removing spoofed header", "key", key, "value", value)
						req.Header.Del(key)
					}
				}

				if who != nil {
					req.Header.Set("X-Remote-User-Email", strings.ToLower(who.UserProfile.LoginName))
					req.Header.Set("X-Remote-User", who.UserProfile.DisplayName)
					req.Header.Set("X-Remote-User-Id", who.UserProfile.ID.String())
				}

				proxy.ServeHTTP(writer, req)
			})))
}
