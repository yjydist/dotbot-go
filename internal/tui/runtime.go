package tui

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yjydist/dotbot-go/internal/output"
)

func RunReview(stdin io.Reader, stdout io.Writer, noColor bool, data output.ReviewData) error {
	model := newReviewModel(data, noColor)
	_, err := tea.NewProgram(
		model,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithAltScreen(),
	).Run()
	if err != nil {
		return fmt.Errorf("runtime error: review ui: %w", err)
	}
	return nil
}
