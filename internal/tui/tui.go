package tui

import (
	"downloadcleaner/internal/history"
	"downloadcleaner/internal/organizer"
	"downloadcleaner/internal/rules"
	"downloadcleaner/internal/scanner"
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"
)

type App struct {
	app          *tview.Application
	files        []scanner.FileInfo
	rules        *rules.Manager
	history      *history.Manager
	downloadsDir string
	currentPlan  *organizer.Plan
}

func NewApp(downloads string, files []scanner.FileInfo, rules *rules.Manager) *App {
	hist := history.NewManager()
	hist.Load()

	return &App{
		app:          tview.NewApplication(),
		files:        files,
		rules:        rules,
		history:      hist,
		downloadsDir: downloads,
	}
}

func (a *App) Run() error {
	return a.showMainMenu()
}

func (a *App) showMainMenu() error {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	// logo
	logo := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(`
╔══════════════════════════════════╗
║      DOWNLOAD CLEANER            ║
║   Terminal download organizer    ║
╚══════════════════════════════════╝

`)

	// stats
	_, _, fileCount := scanner.GetStats(a.files)
	stats := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("📂 %s  |  %d files", a.downloadsDir, fileCount))

	// menu
	menu := tview.NewList().
		AddItem("Basic Mode", "Auto-organize with default rules", '1', a.basicMode).
		AddItem("Custom Mode", "Define your own rules", '2', a.customMode).
		AddItem("History", "View past cleanup operations", '3', a.showHistory).
		AddItem("Undo Last", "Reverse the most recent cleanup", 'u', a.undoLast).
		AddItem("Exit", "Quit the application", 'q', func() { a.app.Stop() })

	menu.SetBorder(true).SetTitle("Menu")

	flex.AddItem(logo, 6, 0, false)
	flex.AddItem(stats, 2, 0, false)
	flex.AddItem(menu, 0, 1, true)

	a.app.SetRoot(flex, true)
	return a.app.Run()
}

func (a *App) basicMode() {
	// show loading
	loading := tview.NewModal().SetText("Analyzing files...")
	a.app.SetRoot(loading, true)

	// do the work
	plan := organizer.PlanMoves(a.files, a.rules)
	a.currentPlan = &plan

	// show preview
	a.showPreview(plan)
}

func (a *App) showPreview(plan organizer.Plan) {
	var text strings.Builder

	if len(plan.Operations) == 0 {
		text.WriteString("No files to organize!\n\nYour Downloads folder is already clean.")
		modal := tview.NewModal().
			SetText(text.String()).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.showMainMenu()
			})
		a.app.SetRoot(modal, true)
		return
	}

	text.WriteString("📋 PREVIEW — NO FILES WILL BE MOVED YET\n\n")

	// show first 10 operations
	show := 10
	if len(plan.Operations) < show {
		show = len(plan.Operations)
	}

	for i := 0; i < show; i++ {
		op := plan.Operations[i]
		text.WriteString(fmt.Sprintf("  %s → %s/\n", op.Filename, op.Category))
	}

	if len(plan.Operations) > 10 {
		text.WriteString(fmt.Sprintf("\n  ... and %d more files\n", len(plan.Operations)-10))
	}

	text.WriteString("\n")
	text.WriteString(fmt.Sprintf("📊 %d files will be organized\n", len(plan.Operations)))

	// breakdown
	text.WriteString("\n")
	for cat, count := range plan.CategoryCounts {
		text.WriteString(fmt.Sprintf("  %-12s %d\n", cat, count))
	}

	if len(plan.Unknown) > 0 {
		text.WriteString(fmt.Sprintf("\n⚠️  %d files will be left untouched\n", len(plan.Unknown)))
	}

	if len(plan.Collisions) > 0 {
		text.WriteString(fmt.Sprintf("⚠️  %d filename conflicts will be resolved\n", len(plan.Collisions)))
	}

	modal := tview.NewModal().
		SetText(text.String()).
		AddButtons([]string{"Execute", "Dry Run", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case "Execute":
				a.executePlan(false)
			case "Dry Run":
				a.executePlan(true)
			default:
				a.showMainMenu()
			}
		})

	a.app.SetRoot(modal, true)
}

func (a *App) executePlan(dryRun bool) {
	if a.currentPlan == nil {
		a.showMainMenu()
		return
	}

	// show loading
	loading := tview.NewModal().SetText("Moving files...")
	a.app.SetRoot(loading, true)

	start := time.Now()
	ops, err := organizer.Execute(a.currentPlan, dryRun)
	if err != nil {
		// show error
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Error: %v", err)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.showMainMenu()
			})
		a.app.SetRoot(modal, true)
		return
	}

	// save history if not dry run
	if !dryRun {
		// convert to history operations
		histOps := []history.Operation{}
		for _, op := range ops {
			if op.Success {
				histOps = append(histOps, history.Operation{
					From:     op.From,
					To:       op.To,
					Filename: op.Filename,
					Category: op.Category,
				})
			}
		}
		summary := fmt.Sprintf("%d files organized", len(histOps))
		a.history.AddEntry(histOps, summary)
	}

	// show report
	report := organizer.GenerateReport(*a.currentPlan, dryRun, start)

	if len(ops) > 0 {
		report += "\n\n" + strings.Repeat("─", 50) + "\n\nMoved:\n"
		show := 5
		if len(ops) < show {
			show = len(ops)
		}
		for i := 0; i < show; i++ {
			op := ops[i]
			if op.Success {
				report += fmt.Sprintf("  %s → %s/\n", op.Filename, op.Category)
			}
		}
		if len(ops) > 5 {
			report += fmt.Sprintf("  ... and %d more\n", len(ops)-5)
		}
	}

	modal := tview.NewModal().
		SetText(report).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.showMainMenu()
		})

	a.app.SetRoot(modal, true)
}

func (a *App) customMode() {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	title := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("📝 CUSTOM RULES")

	ruleList := tview.NewList()
	for i, r := range a.rules.Rules {
		desc := strings.Join(r.Extensions, " ")
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		ruleList.AddItem(r.Name, desc, byte('0'+i%10), nil)
	}
	ruleList.SetBorder(true).SetTitle("Rules")

	buttons := tview.NewFlex()
	btnAdd := tview.NewButton("Add").SetSelectedFunc(a.addRule)
	btnEdit := tview.NewButton("Edit").SetSelectedFunc(a.editRule)
	btnDelete := tview.NewButton("Delete").SetSelectedFunc(a.deleteRule)
	btnBack := tview.NewButton("Back").SetSelectedFunc(func() { a.showMainMenu() })

	buttons.AddItem(btnAdd, 0, 1, false)
	buttons.AddItem(btnEdit, 0, 1, false)
	buttons.AddItem(btnDelete, 0, 1, false)
	buttons.AddItem(btnBack, 0, 1, false)

	flex.AddItem(title, 3, 0, false)
	flex.AddItem(ruleList, 0, 1, true)
	flex.AddItem(buttons, 3, 0, false)

	flex.SetBorder(true).SetTitle("Custom Mode")

	a.app.SetRoot(flex, true)
}

func (a *App) addRule() {
	form := tview.NewForm()
	form.AddInputField("Rule Name", "", 20, nil, nil)
	form.AddInputField("Extensions (space separated)", "", 40, nil, nil)
	form.AddButton("Save", func() {
		name := form.GetFormItem(0).(*tview.InputField).GetText()
		exts := strings.Fields(form.GetFormItem(1).(*tview.InputField).GetText())
		if name != "" && len(exts) > 0 {
			a.rules.SetRule(name, exts)
			a.rules.Save()
		}
		a.customMode()
	})
	form.AddButton("Cancel", func() { a.customMode() })

	form.SetBorder(true).SetTitle("Add Rule")

	a.app.SetRoot(form, true)
}

func (a *App) editRule() {
	// get selected rule from list - for now just show a selection modal
	var text strings.Builder
	text.WriteString("Select a rule to edit:\n\n")
	for i, r := range a.rules.Rules {
		text.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Name))
	}

	modal := tview.NewModal().
		SetText(text.String()).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			
			a.customMode()
		})

	a.app.SetRoot(modal, true)
}

func (a *App) deleteRule() {
	var text strings.Builder
	text.WriteString("Select a rule to delete:\n\n")
	for i, r := range a.rules.Rules {
		if !isDefaultRule(r) {
			text.WriteString(fmt.Sprintf("%d. %s (custom)\n", i+1, r.Name))
		}
	}
	text.WriteString("\n(only custom rules can be deleted)")

	modal := tview.NewModal().
		SetText(text.String()).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.customMode()
		})

	a.app.SetRoot(modal, true)
}

func (a *App) showHistory() {
	summary := a.history.GetHistorySummary()
	modal := tview.NewModal().
		SetText(summary).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.showMainMenu()
		})

	a.app.SetRoot(modal, true)
}

func (a *App) undoLast() {
	entry, err := a.history.GetLast()
	if err != nil || entry == nil {
		msg := "No operations to undo"
		if err != nil {
			msg = fmt.Sprintf("Error: %v", err)
		}
		modal := tview.NewModal().
			SetText(msg).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.showMainMenu()
			})
		a.app.SetRoot(modal, true)
		return
	}

	// confirm
	confirm := tview.NewModal().
		SetText(fmt.Sprintf("Undo cleanup #%d?\n%s", entry.ID, entry.Summary)).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				a.doUndo()
			} else {
				a.showMainMenu()
			}
		})

	a.app.SetRoot(confirm, true)
}

func (a *App) doUndo() {
	// show loading
	loading := tview.NewModal().SetText("Undoing...")
	a.app.SetRoot(loading, true)

	ops, err := a.history.UndoLast()
	if err != nil {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Undo failed: %v", err)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.showMainMenu()
			})
		a.app.SetRoot(modal, true)
		return
	}

	// show what was undone
	var text strings.Builder
	text.WriteString("✅ UNDO COMPLETE\n\n")
	text.WriteString(fmt.Sprintf("%d files restored to original locations\n", len(ops)))

	if len(ops) > 0 {
		text.WriteString("\nRestored:\n")
		show := 5
		if len(ops) < show {
			show = len(ops)
		}
		for i := 0; i < show; i++ {
			text.WriteString(fmt.Sprintf("  %s\n", ops[i].Filename))
		}
		if len(ops) > 5 {
			text.WriteString(fmt.Sprintf("  ... and %d more\n", len(ops)-5))
		}
	}

	modal := tview.NewModal().
		SetText(text.String()).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.showMainMenu()
		})

	a.app.SetRoot(modal, true)
}

// helper to check if a rule is default
func isDefaultRule(r rules.Rule) bool {
	for _, d := range rules.DefaultRules() {
		if d.Name == r.Name && equalSlices(d.Extensions, r.Extensions) {
			return true
		}
	}
	return false
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}