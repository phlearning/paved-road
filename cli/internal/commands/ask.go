package commands

import (
	"bufio"
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

// Answers from a local model on CPU can take a while.
const askTimeout = 3 * time.Minute

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

// assistant calls the RAG service over HTTPS, trusting the internal CA.
type assistant struct {
	http *http.Client
	url  string
}

func newAssistant(cmd *cobra.Command, service string) (*assistant, error) {
	client, err := clusterClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()
	httpClient, err := client.HTTPClient(ctx)
	if err != nil {
		return nil, err
	}
	return &assistant{http: httpClient, url: fmt.Sprintf("https://%s.localhost/ask", service)}, nil
}

func (a *assistant) ask(ctx context.Context, question string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()
	payload, _ := json.Marshal(map[string]string{"question": question})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", a.url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d: %s", a.url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func printAnswer(cmd *cobra.Command, body []byte) error {
	var answer askResponse
	if err := json.Unmarshal(body, &answer); err != nil {
		return fmt.Errorf("unexpected response: %w", err)
	}
	out := cmd.OutOrStdout()
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
			a, err := newAssistant(cmd, service)
			if err != nil {
				return err
			}
			body, err := a.ask(cmd.Context(), strings.Join(args, " "))
			if err != nil {
				return err
			}
			if raw {
				_, err := cmd.OutOrStdout().Write(append(body, '\n'))
				return err
			}
			return printAnswer(cmd, body)
		},
	}
	cmd.Flags().StringVar(&service, "service", "rag-assistant", "service answering the questions")
	cmd.Flags().BoolVar(&raw, "json", false, "print the raw JSON response")
	return cmd
}

func newChatCommand() *cobra.Command {
	var service string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Ask the platform assistant questions one after another",
		Long:  "Read questions line by line and answer each one. Type exit, quit or Ctrl-D to leave.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAssistant(cmd, service)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "paved-road assistant. Ask about the platform, in French or English. Type exit to leave.")
			scanner := bufio.NewScanner(cmd.InOrStdin())
			for {
				fmt.Fprint(out, "\n> ")
				if !scanner.Scan() {
					fmt.Fprintln(out)
					return scanner.Err()
				}
				question := strings.TrimSpace(scanner.Text())
				switch question {
				case "":
					continue
				case "exit", "quit":
					return nil
				}
				body, err := a.ask(cmd.Context(), question)
				if err != nil {
					// Keep the session alive: the next question may work.
					fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
					continue
				}
				if err := printAnswer(cmd, body); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
				}
			}
		},
	}
	cmd.Flags().StringVar(&service, "service", "rag-assistant", "service answering the questions")
	return cmd
}
