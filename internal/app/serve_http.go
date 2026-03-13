package app

import (
	"fmt"
	"strings"
)

type serveHTTPOptions struct {
	ListenAddr     string
	EndpointPath   string
	AllowedOrigins []string
}

func parseServeHTTPOptions(args []string) (serveHTTPOptions, error) {
	options := serveHTTPOptions{
		ListenAddr:   "127.0.0.1:8080",
		EndpointPath: "/mcp",
	}

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch arg {
		case "":
			continue
		case "--listen":
			value, next, err := optionValue(args, i)
			if err != nil {
				return serveHTTPOptions{}, err
			}
			options.ListenAddr = value
			i = next
		case "--path":
			value, next, err := optionValue(args, i)
			if err != nil {
				return serveHTTPOptions{}, err
			}
			options.EndpointPath = value
			i = next
		case "--allow-origin":
			value, next, err := optionValue(args, i)
			if err != nil {
				return serveHTTPOptions{}, err
			}
			options.AllowedOrigins = append(options.AllowedOrigins, value)
			i = next
		default:
			return serveHTTPOptions{}, fmt.Errorf("unknown serve-http flag %q", arg)
		}
	}

	options.ListenAddr = strings.TrimSpace(options.ListenAddr)
	options.EndpointPath = strings.TrimSpace(options.EndpointPath)
	if options.ListenAddr == "" {
		return serveHTTPOptions{}, fmt.Errorf("serve-http listen address is required")
	}
	if options.EndpointPath == "" {
		return serveHTTPOptions{}, fmt.Errorf("serve-http path is required")
	}
	if !strings.HasPrefix(options.EndpointPath, "/") {
		options.EndpointPath = "/" + options.EndpointPath
	}
	return options, nil
}

func optionValue(args []string, index int) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("missing value for %q", strings.TrimSpace(args[index]))
	}
	return strings.TrimSpace(args[index+1]), index + 1, nil
}
