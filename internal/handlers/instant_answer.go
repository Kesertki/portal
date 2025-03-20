package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

type IntOrEmpty struct {
	Value *int
}

func (i *IntOrEmpty) UnmarshalJSON(data []byte) error {
	// Check if the data is an empty string
	if string(data) == `""` {
		i.Value = nil
		return nil
	}

	// Try to unmarshal the data into an int
	var intValue int
	if err := json.Unmarshal(data, &intValue); err != nil {
		return err
	}

	i.Value = &intValue
	return nil
}

type DuckDuckGoResponse struct {
	Abstract         string     `json:"Abstract"`
	AbstractSource   string     `json:"AbstractSource"`
	AbstractText     string     `json:"AbstractText"`
	AbstractURL      string     `json:"AbstractURL"`
	Answer           string     `json:"Answer"`
	Definition       string     `json:"Definition"`
	AnswerType       string     `json:"AnswerType"`
	DefinitionSource string     `json:"DefinitionSource"`
	DefinitionURL    string     `json:"DefinitionURL"`
	Entity           string     `json:"Entity"`
	Heading          string     `json:"Heading"`
	Image            string     `json:"Image"`
	ImageHeight      IntOrEmpty `json:"ImageHeight"`
	ImageIsLogo      IntOrEmpty `json:"ImageIsLogo"`
	ImageWidth       IntOrEmpty `json:"ImageWidth"`
	RelatedTopics    []struct {
		FirstURL string `json:"FirstURL"`
		Icon     struct {
			Height IntOrEmpty `json:"Height"`
			URL    string     `json:"URL"`
			Width  IntOrEmpty `json:"Width"`
		} `json:"Icon"`
		Result string `json:"Result"`
		Text   string `json:"Text"`
	} `json:"RelatedTopics"`
}

// Returns an instant answer for the given search query.
// Uses the DuckDuckGo Instant Answer API: https://api.duckduckgo.com/?q=hello&format=json
func InstantAnswer(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.String(400, "Missing search query")
	}

	url := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json", query)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error:", err)
		return c.String(500, "Failed to search")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response:", err)
		return c.String(500, "Failed to read search response")
	}

	// Pretty-print the entire JSON response
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		log.Println("Error pretty-printing JSON:", err)
		return c.String(500, "Failed to pretty-print search response")
	}
	// fmt.Println("Full JSON Response:")
	// fmt.Println(prettyJSON.String())

	var result DuckDuckGoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Println("Error parsing JSON:", err)
		return c.String(500, "Failed to parse search response")
	}

	if result.AbstractText == "" && result.Answer == "" && len(result.RelatedTopics) == 0 {
		return c.String(http.StatusNotFound, "No results found")
	}

	return c.JSON(http.StatusOK, result)
}
