package tui

import (
	"download-cleaner/internal/organizer"
	"download-cleaner/internal/rules"
	"download-cleaner/internal/scanner"
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type App struct {
	app          *tview.Application
	files        []scanner.FileInfo
	rules        *rules.Manager
	downloadsDir string
}

func NewApp(downloads string, files []scanner.FileInfo, rules *rules.Manager) *App {
	return &App{
		app:          tview.NewApplication(),
		files:        files,
		rules:        rules,
		downloadsDir: downloads,
	}
}

func (a *App) Run() error {
	// main layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow)

	// logo/header area
	logo := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(`
╔══════════════════════════════╗
║     DOWNLOAD CLEANER         ║
╚══════════════════════════════╝

`).
		SetDynamicColors(true)

	// stats
	_, _, fileCount := scanner.GetStats(a.files)
	stats := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("📂 %s  |  %d files", a.downloadsDir, fileCount))

	// menu
	menu := tview.NewList().
		AddItem("Basic Mode", "Auto-organize with default rules", '1', a.basicMode).
		AddItem("Custom Mode", "Define your own rules", '2', a.customMode).
		AddItem("View Rules", "See current classification rules", '3', a.viewRules).
		AddItem("Exit", "Quit the application", 'q', func() { a.app.Stop() })

	menu.SetBorder(true).SetTitle("Menu")

	// main container
	flex.AddItem(logo, 6, 0, false)
	flex.AddItem(stats, 2, 0, false)
	flex.AddItem(menu, 0, 1, true)

	a.app.SetRoot(flex, true)
	return a.app.Run()
}

// Basic Mode handler
func (a *App) basicMode() {
	modal := tview.NewModal().
		SetText("Organize Downloads using default rules?").
		AddButtons([]string{"Preview", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Preview" {
				a.showPreview()
			} else {
				a.app.Stop()
			}
		})

	a.app.SetRoot(modal, true)
}

// Custom Mode handler
func (a *App) customMode() {
	// show rule management UI
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	title := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("CUSTOM RULES")

	ruleList := tview.NewList()
	for i, r := range a.rules.Rules {
		desc := strings.Join(r.Extensions, " ")
		ruleList.AddItem(r.Name, desc, byte('0'+i%10), nil)
	}

	// buttons
	buttons := tview.NewFlex()
	btnAdd := tview.NewButton("Add").SetSelectedFunc(a.addRule)
	btnEdit := tview.NewButton("Edit").SetSelectedFunc(a.editRule)
	btnDelete := tview.NewButton("Delete").SetSelectedFunc(a.deleteRule)
	btnBack := tview.NewButton("Back").SetSelectedFunc(func() { a.app.Stop() })

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

// View Rules handler
func (a *App) viewRules() {
	var text strings.Builder
	text.WriteString("Current Classification Rules\n\n")

	for _, r := range a.rules.Rules {
		text.WriteString(fmt.Sprintf("📁 %s\n", r.Name))
		for _, ext := range r.Extensions {
			text.WriteString(fmt.Sprintf("   %s\n", ext))
		}
		text.WriteString("\n")
	}

	modal := tview.NewModal().
		SetText(text.String()).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.app.Stop()
		})

	a.app.SetRoot(modal, true)
}

// show preview of what would be moved
func (a *App) showPreview() {
	plan := organizer.PlanMoves(a.files, a.rules)

	var text strings.Builder
	text.WriteString("PREVIEW - NO FILES WILL BE MOVED\n\n")

	for _, op := range plan.Operations {
		text.WriteString(fmt.Sprintf("  %s → %s/\n", op.Filename, op.Category))
	}

	if len(plan.Unknown) > 0 {
		text.WriteString(fmt.Sprintf("\n⚠️  %d files left untouched (unknown type)\n", len(plan.Unknown)))
	}

	if len(plan.Collisions) > 0 {
		text.WriteString(fmt.Sprintf("⚠️  %d filename conflicts will be resolved\n", len(plan.Collisions)))
	}

	text.WriteString(fmt.Sprintf("\n📊 %d files will be organized\n", len(plan.Operations)))

	modal := tview.NewModal().
		SetText(text.String()).
		AddButtons([]string{"Execute", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Execute" {
				
				a.showComplete(plan)
			} else {
				a.app.Stop()
			}
		})

	a.app.SetRoot(modal, true)
}


func (a *App) showComplete(plan organizer.Plan) {
	var text strings.Builder
	text.WriteString("✅ CLEANUP COMPLETE\n\n")
	text.WriteString(fmt.Sprintf("%d files organized\n\n", len(plan.Operations)))

	for cat, count := range plan.CategoryCounts {
		text.WriteString(fmt.Sprintf("  %-12s %d\n", cat, count))
	}

	if len(plan.Unknown) > 0 {
		text.WriteString(fmt.Sprintf("\n⚠️  %d files left untouched\n", len(plan.Unknown)))
	}

	modal := tview.NewModal().
		SetText(text.String()).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.app.Stop()
		})

	a.app.SetRoot(modal, true)
}

// rule management helpers
func (a *App) addRule() {
	// simple form for adding a rule
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
		a.app.Stop()
	})
	form.AddButton("Cancel", func() { a.app.Stop() })

	form.SetBorder(true).SetTitle("Add Rule")

	a.app.SetRoot(form, true)
}

func (a *App) editRule() {

	modal := tview.NewModal().
		SetText("Edit rule: select a rule from the list first").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.app.Stop()
		})

	a.app.SetRoot(modal, true)
}

func (a *App) deleteRule() {
	modal := tview.NewModal().
		SetText("Delete rule: select a rule from the list first").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.app.Stop()
		})

	a.app.SetRoot(modal, true)
}