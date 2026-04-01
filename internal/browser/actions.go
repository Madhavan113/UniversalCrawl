package browser

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/madhavanp/universalcrawl/internal/models"
)

// ExecuteActions runs a sequence of browser actions on the given page.
func ExecuteActions(page *rod.Page, actions []models.BrowserAction) error {
	for i, action := range actions {
		if err := executeAction(page, action); err != nil {
			return fmt.Errorf("action %d (%s): %w", i, action.Type, err)
		}
	}
	return nil
}

func executeAction(page *rod.Page, action models.BrowserAction) error {
	switch action.Type {
	case "click":
		if action.Selector == "" {
			return fmt.Errorf("click action requires selector")
		}
		el, err := page.Element(action.Selector)
		if err != nil {
			return fmt.Errorf("find element %q: %w", action.Selector, err)
		}
		return el.Click(proto.InputMouseButtonLeft, 1)

	case "type":
		if action.Selector == "" {
			return fmt.Errorf("type action requires selector")
		}
		el, err := page.Element(action.Selector)
		if err != nil {
			return fmt.Errorf("find element %q: %w", action.Selector, err)
		}
		return el.Input(action.Text)

	case "scroll":
		amount := action.Amount
		if amount == 0 {
			amount = 500
		}
		if action.Direction == "up" {
			amount = -amount
		}
		return page.Mouse.Scroll(0, float64(amount), 0)

	case "wait":
		ms := action.Milliseconds
		if ms == 0 {
			ms = 1000
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return nil

	case "press":
		return page.Keyboard.Press(input.Enter)

	case "screenshot":
		// Screenshot action is handled by the transform pipeline, not here
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}
