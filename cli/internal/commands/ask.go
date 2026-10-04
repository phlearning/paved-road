package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type askResponse struct {
	Answer  string `json:"answer"`
	Model   string `json:"model"`
	Sources []struct {
		Ref     int     `json:"ref"`
		Source  string  `json:"source"`
		Heading string  `json:"heading"`
		Score   float64 `json:"score"`
	} `json:"sources"`
	Timings map[string]float64 `json:"timings"`
}

func newAskCommand() *cobra.Command {
	var service string
	var raw bool

	cmd := &cobra.Command{
		Use:     "ask QUESTION",
		Short:   "Ask the platform assistant a question about paved-road",
		Example: `  platformctl ask "Comment ajouter une base de données à mon service ?"`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clusterClient()
			if err != nil {
				return err
			}
			// Answers from a local model on CPU can take a while.
			ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Minute)
			defer cancel()

			httpClient, err := client.HTTPClient(ctx)
			if err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]string{"question": strings.Join(args, " ")})
			url := fmt.Sprintf("https://%s.localhost/ask", service)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("call %s: %w", url, err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("%s returned %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
			}

			out := cmd.OutOrStdout()
			if raw {
				_, err := out.Write(append(body, '\n'))
				return err
			}
			var answer askResponse
			if err := json.Unmarshal(body, &answer); err != nil {
				return fmt.Errorf("unexpected response: %w", err)
			}
			fmt.Fprintf(out, "%s\n", answer.Answer)
			if len(answer.Sources) > 0 {
				fmt.Fprintln(out, "\nSources:")
				for _, s := range answer.Sources {
					fmt.Fprintf(out, "  [%d] %s > %s\n", s.Ref, s.Source, s.Heading)
				}
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "\n(%s, retrieval %.2fs, generation %.1fs)\n",
				answer.Model, answer.Timings["retrieval"], answer.Timings["generation"])
			return nil
		},
	}
	cmd.Flags().StringVar(&service, "service", "rag-assistant", "service answering the questions")
	cmd.Flags().BoolVar(&raw, "json", false, "print the raw JSON response")
	return cmd
}
