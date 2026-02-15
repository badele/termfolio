package ui

import (
	"fmt"
	"os/exec"
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
}

// NewModel builds a model from the config.
func NewModel(cfg *config.Config) Model {
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
	}

	return model
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
		footerHeight := 3 // Menu en bas

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetHorizontalStep(2)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}

		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportContent()

	case layerResultMsg:
		if msg.index >= 0 && msg.index < len(m.layerOutput) {
			if msg.err != nil {
				message := fmt.Sprintf("Erreur: %v", msg.err)
				if strings.TrimSpace(msg.output) != "" {
					message = message + "\n" + msg.output
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
		output := strings.TrimSpace(m.layerOutput[m.currentLayer])
		if output == "" {
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
		key   string
		label string
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
		blocks = append(blocks, menuBlock{key: "L", label: langLabel})
	}
	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		blocks = append(blocks, menuBlock{key: " ", label: "haut/bas"})
	}
	if m.hasOverflowX {
		blocks = append(blocks, menuBlock{key: " ", label: "gauche/droite"})
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
		if block.key == "L" {
			currentLabelStyle = langStyle
		}
		menu.WriteString(currentLabelStyle.Width(labelWidth).Render(labelText))
	}

	return "\n" + menu.String()
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
			return layerResultMsg{index: index, output: "", err: fmt.Errorf("commande vide")}
		}
		output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
		return layerResultMsg{index: index, output: strings.TrimRight(string(output), "\n"), err: err}
	}
}

// setViewportContent updates the viewport and overflow state.
func (m *Model) setViewportContent(content string) {
	m.viewport.SetContent(content)
	m.hasOverflowX = hasHorizontalOverflow(content, m.viewport.Width-m.viewport.Style.GetHorizontalFrameSize())
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
