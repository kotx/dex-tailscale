package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"tailscale.com/client/local"
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
	flag.Parse()

	logLevel, err := ParseLevel(*logLevel)
	if err != nil {
		log.Fatal("error parsing log level: ", err)
	}
	slog.SetLogLoggerLevel(logLevel)

	if *endpoint == "" {
		log.Fatal("endpoint must be set")
	}
	endpointUrl, err := url.Parse(*endpoint)
	if err != nil {
		log.Fatal("endpoint must be a valid url: ", err)
	}

	serve := &tsnet.Server{
		Hostname: *tsHost,
		Dir:      "/state",
	}
	defer serve.Close()

	ln, err := serve.ListenFunnel("tcp", ":443")
	if err != nil {
		log.Fatal("error listening on funnel: ", err)
	}

	lc, err := serve.LocalClient()
	if err != nil {
		log.Fatal("error creating tailscale local client: ", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(endpointUrl)
	slog.Info("proxying requests", "to", endpointUrl)

	log.Fatal(http.Serve(ln,
		http.HandlerFunc(
			func(writer http.ResponseWriter, req *http.Request) {
				slog.Info("proxying", slog.Group("request",
					"method", req.Method,
					"url", req.URL,
					"remote_addr", req.RemoteAddr,
				))

				who, err := lc.WhoIs(req.Context(), req.RemoteAddr)
				if err != nil && err != local.ErrPeerNotFound {
					slog.Error("tailscale whois", "err", err)
					http.Error(writer, err.Error(), 500)
					return
				}

				req.Host = endpointUrl.Host
				for key, value := range req.Header {
					if strings.HasPrefix(http.CanonicalHeaderKey("X-Remote-"), key) {
						slog.Info("removing spoofed header", "key", key, "value", value)
						req.Header.Del(key)
					}
				}

				if who != nil {
					loginName := strings.ToLower(who.UserProfile.LoginName)
					userName, _, _ := strings.Cut(loginName, "@")

					slog.Debug("tailscale", slog.Group("whois",
						slog.Group("node", "id", who.Node.ID, "name", who.Node.Name),
						slog.Group("user",
							"id", who.UserProfile.ID,
							"login_name", loginName,
							"username", userName,
						),
					))

					// Disallow tagged devices (the username is shared, which leads to unexpected behavior).
					// This also prevents funnel from being "authenticated" with the following credentials:
					//   node(id=nodeid:f3e4fcf98730b3, name="")
					//   user(id=userid:6a98d6d013b0f, login_name=tagged-devices, display_name="tagged devices")
					if loginName != "tagged-devices" {
						// https://github.com/dexidp/dex/blob/master/connector/authproxy/authproxy.go
						req.Header.Set("X-Remote-User-Email", loginName) // emailish
						req.Header.Set("X-Remote-User", userName)        // user name (full name) AND preferred username (shorthand)
						req.Header.Set("X-Remote-User-Id", who.UserProfile.ID.String())
					}
				} else {
					slog.Debug("tailscale", "whois", nil)
				}

				proxy.ServeHTTP(writer, req)
			})))
}
