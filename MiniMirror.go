package main

import (
	"bytes"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

var (
	TargetDomain     = os.Getenv("TARGET_DOMAIN")
	TargetEndpoint   = os.Getenv("TARGET_ENDPOINT")
	SecondaryDomains = strings.Split(os.Getenv("SECONDARY_DOMAINS"), ";")
	port             = os.Getenv("PORT")
)

const MaxRetry = 3

func mirrorUrl(url string, c *fiber.Ctx, retry int8) error {
	log.Printf("mirroring %s", url)
	reqBodyBuffer := bytes.NewBuffer(c.Body())

	req, err := http.NewRequest(c.Method(), url, reqBodyBuffer)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendStatus(fiber.StatusInternalServerError)
	}
	client := &http.Client{}

	// Copy headers from Fiber context to the new http.Request
	for k, v := range c.GetReqHeaders() {
		for _, vv := range v {
			if k != "Accept-Encoding" {
				req.Header.Add(k, vv)
			}
		}
	}

	// Copy query params
	q := req.URL.Query()
	for key, val := range c.Queries() {
		if key == "EXTERNAL_URL" {
			continue
		}
		q.Add(key, val)
	}
	req.URL.RawQuery = q.Encode()

	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		if retry < MaxRetry {
			log.Printf(err.Error())
			log.Printf("retrying to mirror %s", url)
			return mirrorUrl(url, c, retry+1)
		}
		log.Printf("Failed after %d retries, returning error", retry)
		return c.Status(fiber.StatusInternalServerError).SendStatus(fiber.StatusInternalServerError)
	}
	// Retry if server error
	if resp.StatusCode >= 500 && resp.StatusCode < 600 && retry < MaxRetry {
		log.Printf("Status code %d, retrying to mirror %s", resp.StatusCode, url)
		return mirrorUrl(url, c, retry+1)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf(err.Error())
		}
	}(resp.Body)

	for name, values := range resp.Header {
		for _, value := range values {
			c.Set(name, value)
		}
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendStatus(fiber.StatusInternalServerError)
	}

	// Replace domain with relative link
	body = []byte(strings.ReplaceAll(string(body), TargetDomain+"/", "/"))

	// Replace secondary domains if there are any with proxy link
	if len(SecondaryDomains) > 0 && !(SecondaryDomains[0] == "") {
		for _, secDomain := range SecondaryDomains {
			body = []byte(strings.ReplaceAll(string(body), secDomain, "/_EXTERNAL_?EXTERNAL_URL="+secDomain))
		}
	}

	return c.Status(resp.StatusCode).Send(body)
}

func handleInternalRequest(c *fiber.Ctx) error {
	// Form new URL
	newURL := c.Path()
	if TargetEndpoint != "" {
		newURL = TargetEndpoint + newURL
	} else {
		newURL = TargetDomain + newURL
	}

	return mirrorUrl(newURL, c, 0)
}

func handleExternalRequest(c *fiber.Ctx) error {
	return mirrorUrl(c.Query("EXTERNAL_URL"), c, 0)
}

func main() {
	if port == "" {
		port = "3000"
	}

	app := fiber.New()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c
		fmt.Println("Gracefully shutting down...")
		_ = app.Shutdown()
	}()

	app.All("/_EXTERNAL_", func(c *fiber.Ctx) error {
		return handleExternalRequest(c)
	})

	app.Get("/check", func(c *fiber.Ctx) error {
		return c.SendString("Ok")
	})

	app.All("/*", func(c *fiber.Ctx) error {
		return handleInternalRequest(c)
	})

	if err := app.Listen(":" + port); err != nil {
		log.Panic(err)
	}
	fmt.Println("Goodbye!")
}
