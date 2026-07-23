package handlers

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/evocrm/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func Liveness() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "alive", "time": time.Now().UTC()})
	}
}

func Readiness(svc *services.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		checks := fiber.Map{}
		ready := true
		add := func(name string, err error, required bool) {
			status := "ok"
			if err != nil {
				status = "error"
				if required {
					ready = false
				}
			}
			checks[name] = fiber.Map{"status": status, "required": required}
		}

		add("postgres", svc.DB.PingContext(ctx), true)
		add("redis", svc.Redis.Ping(ctx).Err(), true)
		add("evolution", checkHTTPReachable(ctx, svc.Config.EvolutionAPIURL), true)
		add("smtp", checkTCP(ctx, net.JoinHostPort(svc.Config.SMTPHost, svc.Config.SMTPPort)), svc.Config.SMTPHost != "")
		add("clamav", checkClamAV(ctx, svc.Config.ClamAVAddr), svc.Config.ClamAVAddr != "")

		jobHealth := svc.Jobs.Health(ctx)
		checks["workers"] = jobHealth
		if databaseError, _ := jobHealth["database_error"].(string); databaseError != "" {
			ready = false
		}
		if workerStatus, _ := jobHealth["status"].(string); workerStatus != "ok" {
			ready = false
		}

		status := fiber.StatusOK
		label := "ready"
		if !ready {
			status = fiber.StatusServiceUnavailable
			label = "not_ready"
		}
		return c.Status(status).JSON(fiber.Map{"status": label, "checks": checks})
	}
}

func checkHTTPReachable(ctx context.Context, rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("not configured")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(rawURL, "/"), nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: 4 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 500 {
		return fmt.Errorf("status %d", response.StatusCode)
	}
	return nil
}

func checkTCP(ctx context.Context, address string) error {
	if strings.TrimSpace(address) == "" || strings.HasPrefix(address, ":") {
		return fmt.Errorf("not configured")
	}
	var dialer net.Dialer
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	return connection.Close()
}

func checkClamAV(ctx context.Context, address string) error {
	if strings.TrimSpace(address) == "" {
		return fmt.Errorf("not configured")
	}
	var dialer net.Dialer
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err = connection.Write([]byte("zPING\x00")); err != nil {
		return err
	}
	response, err := bufio.NewReader(connection).ReadString(0)
	if err != nil {
		return err
	}
	if !strings.Contains(response, "PONG") {
		return fmt.Errorf("unexpected response")
	}
	return nil
}
