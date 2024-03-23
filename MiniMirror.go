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
	SecondaryDomains = strings.Split(os.Getenv("SECONDARY_DOMAINS"), ";")
	port             = os.Getenv("PORT")
)

func mirrorUrl(url string, c *fiber.Ctx) error {
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

	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return c.Status(fiber.StatusInternalServerError).SendStatus(fiber.StatusInternalServerError)
	}
	defer resp.Body.Close()

	for name, values := range resp.Header {
		for _, value := range values {
			c.Append(name, value)
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
			body = []byte(strings.ReplaceAll(string(body), secDomain, "/_EXTERNAL_?url="+secDomain))
		}
	}

	return c.Status(resp.StatusCode).Send(body)
}

func handleInternalRequest(c *fiber.Ctx) error {

	// Form new URL
	newURL := TargetDomain + c.Path()

	return mirrorUrl(newURL, c)
}

func handleExternalRequest(c *fiber.Ctx) error {
	return mirrorUrl(c.Query("url"), c)
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
