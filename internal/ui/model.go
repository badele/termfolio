package ui

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/badele/termfolio/internal/config"
)

// Model holds the application state.
type Model struct {
	viewport         viewport.Model
	layers           []config.Layer
	layerOutput      []string
	currentLayer     int
	langs            []config.Lang
	currentLangIndex int
	hasOverflowX     bool
	width            int
	height           int
	ready            bool
	backgroundColor  string
}

// NewModel builds a model from the config.
func NewModel(cfg *config.Config, preferredLang string) Model {
	model := Model{}

	if cfg != nil {
		if len(cfg.Langs) > 0 {
			model.langs = cfg.Langs
		}
		if len(cfg.Layers) > 0 {
			model.layers = cfg.Layers
			model.layerOutput = make([]string, len(cfg.Layers))
			model.layerOutput[0] = "Chargement..."
			model.currentLayer = 0
		}
		if strings.TrimSpace(cfg.BackgroundColor) != "" {
			model.backgroundColor = strings.TrimSpace(cfg.BackgroundColor)
		}
	}

	model.applyPreferredLang(preferredLang)

	return model
}

func (m *Model) applyPreferredLang(preferredLang string) {
	if len(m.langs) == 0 {
		return
	}
	langCode := strings.TrimSpace(preferredLang)
	if langCode == "" {
		langCode = "fr"
	}
	for index, lang := range m.langs {
		if strings.EqualFold(strings.TrimSpace(lang.Code), langCode) {
			m.currentLangIndex = index
			return
		}
	}
	if m.currentLangIndex < 0 || m.currentLangIndex >= len(m.langs) {
		m.currentLangIndex = 0
	}
}

// Init runs initial commands for the model.
func (m Model) Init() tea.Cmd {
	if len(m.layers) > 0 {
		return runLayerCmd(m.layerCmd(m.currentLayer), m.currentLayer)
	}

	return nil
}

type layerResultMsg struct {
	index  int
	output string
	err    error
	cmd    string
}

// Update handles user input and async results.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "L", "l":
			if len(m.langs) > 0 {
				m.currentLangIndex = (m.currentLangIndex + 1) % len(m.langs)
				if m.currentLayer >= 0 && m.currentLayer < len(m.layers) {
					m.layerOutput[m.currentLayer] = "Chargement..."
					m.updateViewportContent()
					cmds = append(cmds, runLayerCmd(m.layerCmd(m.currentLayer), m.currentLayer))
				}
			}
		default:
			if index := m.layerIndexForKey(msg.String()); index >= 0 {
				m.currentLayer = index
				m.layerOutput[index] = "Chargement..."
				m.updateViewportContent()
				cmds = append(cmds, runLayerCmd(m.layerCmd(index), index))
			}
		}

	case tea.WindowSizeMsg:
		headerHeight := 3 // Titre en haut
		footerHeight := 4 // Menu + aide en bas

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetHorizontalStep(2)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}

		if m.backgroundColor != "" {
			m.viewport.Style = lipgloss.NewStyle().
				Background(lipgloss.Color(m.backgroundColor)).
				Width(msg.Width)
		}

		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportContent()

	case layerResultMsg:
		if msg.index >= 0 && msg.index < len(m.layerOutput) {
			if msg.err != nil {
				message := fmt.Sprintf("Error: %v", msg.err)
				cmdFile := layerCmdFile(msg.cmd)
				if cmdFile != "" {
					message = fmt.Sprintf("Error (%s): %v", cmdFile, msg.err)
				}
				output := msg.output
				if strings.TrimSpace(output) != "" {
					message = message + "\n" + output
				}
				m.layerOutput[msg.index] = message
			} else {
				m.layerOutput[msg.index] = msg.output
			}

			if m.currentLayer == msg.index {
				m.updateViewportContent()
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// updateViewportContent refreshes the viewport with the current content.
func (m *Model) updateViewportContent() {
	if len(m.layerOutput) == 0 {
		m.setViewportContent("Aucun layer defini")
		return
	}
	if m.currentLayer < len(m.layerOutput) {
		output := m.layerOutput[m.currentLayer]
		if len(output) == 0 {
			output = "Aucune sortie"
		}
		m.setViewportContent(output)
	}
}

// View renders the UI.
func (m Model) View() string {
	if !m.ready {
		return "\n  Chargement..."
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(),
		m.viewport.View(),
		m.renderMenu(),
		m.renderHelp(),
	)
}

// renderHeader builds the top header.
func (m Model) renderHeader() string {
	title := m.layerTitle(m.currentLayer)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("4")).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(0, 1)

	return headerStyle.Render(title) + "\n"
}

// renderMenu builds the footer menu.
func (m Model) renderMenu() string {
	const keyWidth = 2
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	labelStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("4")).
		Foreground(lipgloss.Color("0"))
	langStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("5")).
		Foreground(lipgloss.Color("15"))
	directionStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("1")).
		Foreground(lipgloss.Color("15"))
	truncateWithEllipsis := func(text string, width int) string {
		if width <= 0 {
			return ""
		}
		if ansi.StringWidth(text) <= width {
			return text
		}
		return ansi.Truncate(text, width, "…")
	}

	type menuBlock struct {
		key         string
		label       string
		isLang      bool
		isDirection bool
	}

	var menuItems []menuBlock
	for index, layer := range m.layers {
		menuItems = append(menuItems, menuBlock{
			key:   layer.Key,
			label: m.layerLabel(layer, index),
		})
	}

	blocks := make([]menuBlock, 0, len(menuItems))
	blocks = append(blocks, menuItems...)
	blocks = append(blocks, menuBlock{key: "q", label: "Quitter"})
	langLabel := m.currentLangLabel()
	if langLabel == "" {
		langLabel = m.currentLangCode()
	}
	if langLabel != "" {
		blocks = append(blocks, menuBlock{key: "L", label: langLabel, isLang: true})
	}
	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		blocks = append(blocks, menuBlock{key: " ", label: "haut/bas", isDirection: true})
	}
	if m.hasOverflowX {
		blocks = append(blocks, menuBlock{key: " ", label: "gauche/droite", isDirection: true})
	}

	blockCount := len(blocks)
	if blockCount <= 0 {
		return "\n"
	}

	menuWidth := m.width
	if menuWidth < 0 {
		menuWidth = 0
	}

	baseWidth := menuWidth / blockCount
	remainder := menuWidth % blockCount
	blockWidths := make([]int, blockCount)
	for i := range blockWidths {
		blockWidths[i] = baseWidth
		if i < remainder {
			blockWidths[i]++
		}
	}

	var menu strings.Builder
	for i, block := range blocks {
		blockWidth := blockWidths[i]
		if blockWidth <= 0 {
			continue
		}

		currentKeyWidth := keyWidth
		if currentKeyWidth > blockWidth {
			currentKeyWidth = blockWidth
		}
		labelWidth := blockWidth - currentKeyWidth

		keyText := block.key
		if keyText == "" {
			keyText = " "
		}
		menu.WriteString(keyStyle.Width(currentKeyWidth).Align(lipgloss.Right).Render(keyText))
		if labelWidth <= 0 {
			continue
		}

		labelText := block.label
		if labelText != "" {
			labelText = " " + labelText
		}
		labelText = truncateWithEllipsis(labelText, labelWidth)
		currentLabelStyle := labelStyle
		if block.isLang {
			currentLabelStyle = langStyle
		} else if block.isDirection {
			currentLabelStyle = directionStyle
		}
		menu.WriteString(currentLabelStyle.Width(labelWidth).Render(labelText))
	}

	return "\n" + menu.String()
}

// renderHelp builds the help line under the menu.
func (m Model) renderHelp() string {
	help := m.layerHelp(m.currentLayer)
	width := m.width
	if width < 0 {
		width = 0
	}
	if strings.TrimSpace(help) == "" {
		return "\n"
	}
	if width > 0 && ansi.StringWidth(help) > width {
		help = ansi.Truncate(help, width, "...")
	}
	return "\n" + help
}

// currentLangCode returns the selected language code.
func (m Model) currentLangCode() string {
	if len(m.langs) == 0 {
		return ""
	}
	index := m.currentLangIndex
	if index < 0 || index >= len(m.langs) {
		index = 0
	}
	return strings.TrimSpace(m.langs[index].Code)
}

// currentLangLabel returns the display label for the current language.
func (m Model) currentLangLabel() string {
	if len(m.langs) == 0 {
		return ""
	}
	index := m.currentLangIndex
	if index < 0 || index >= len(m.langs) {
		index = 0
	}
	label := strings.TrimSpace(m.langs[index].Label)
	if label == "" {
		return strings.TrimSpace(m.langs[index].Code)
	}
	return label
}

// fallbackLangCode returns the default language code.
func (m Model) fallbackLangCode() string {
	if len(m.langs) == 0 {
		return ""
	}
	return strings.TrimSpace(m.langs[0].Code)
}

// localizedValue picks the best value for the active language.
func (m Model) localizedValue(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	current := m.currentLangCode()
	if current != "" {
		if value, ok := values[current]; ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	fallback := m.fallbackLangCode()
	if fallback != "" {
		if value, ok := values[fallback]; ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// layerCmd returns the command string for a layer.
func (m Model) layerCmd(index int) string {
	if index < 0 || index >= len(m.layers) {
		return ""
	}
	return strings.TrimSpace(m.localizedValue(m.layers[index].Cmd))
}

// layerLabel returns the menu label for a layer.
func (m Model) layerLabel(layer config.Layer, index int) string {
	label := m.localizedValue(layer.Label)
	if strings.TrimSpace(label) != "" {
		return label
	}
	title := m.localizedValue(layer.Title)
	if strings.TrimSpace(title) != "" {
		return title
	}
	return fmt.Sprintf("Layer %d", index+1)
}

// layerTitle returns the header title for a layer.
func (m Model) layerTitle(index int) string {
	if index < 0 || index >= len(m.layers) {
		return "CV"
	}

	layer := m.layers[index]
	title := m.localizedValue(layer.Title)
	if strings.TrimSpace(title) != "" {
		return title
	}
	return m.layerLabel(layer, index)
}

// layerHelp returns the help line for a layer.
func (m Model) layerHelp(index int) string {
	if index < 0 || index >= len(m.layers) {
		return ""
	}

	return m.localizedValue(m.layers[index].Help)
}

// layerIndexForKey maps a pressed key to a layer index.
func (m Model) layerIndexForKey(key string) int {
	for index, layer := range m.layers {
		if layer.Key == "" {
			continue
		}
		if strings.EqualFold(layer.Key, key) {
			return index
		}
	}

	return -1
}

// runLayerCmd executes a layer command in the shell.
func runLayerCmd(cmd string, index int) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(cmd) == "" {
			return layerResultMsg{index: index, output: "", err: fmt.Errorf("commande vide"), cmd: cmd}
		}
		output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
		return layerResultMsg{index: index, output: string(output), err: err, cmd: cmd}
	}
}

// setViewportContent updates the viewport and overflow state.
func (m *Model) setViewportContent(content string) {
	vpWidth := m.viewport.Width - m.viewport.Style.GetHorizontalFrameSize()

	// Detect horizontal overflow on original content before padding.
	m.hasOverflowX = hasHorizontalOverflow(content, vpWidth)

	// Pad lines with background color so the entire viewport area is filled.
	if m.backgroundColor != "" && vpWidth > 0 {
		content = m.padContentLines(content, vpWidth)
	}

	m.viewport.SetContent(content)
}

// padContentLines pads each line of content with the background color
// so that short lines are filled to the viewport width.
func (m *Model) padContentLines(content string, vpWidth int) string {
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(m.backgroundColor))
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lineWidth := ansi.StringWidth(line)
		if lineWidth < vpWidth {
			padding := strings.Repeat(" ", vpWidth-lineWidth)
			lines[i] = line + bgStyle.Render(padding)
		}
	}
	return strings.Join(lines, "\n")
}

var layerCmdFileRegex = regexp.MustCompile(`[^\s'"]+\.(?:neo|ansi|ans)`)

// layerCmdFile extracts a file name from the command string.
func layerCmdFile(cmd string) string {
	matches := layerCmdFileRegex.FindAllString(cmd, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

// hasHorizontalOverflow reports whether any line exceeds width.
func hasHorizontalOverflow(content string, width int) bool {
	if width <= 0 {
		return false
	}

	content = strings.ReplaceAll(content, "\r\n", "\n")
	for _, line := range strings.Split(content, "\n") {
		if ansi.StringWidth(line) > width {
			return true
		}
	}

	return false
}
