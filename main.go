package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	// "github.com/brianvoe/gofakeit/v6"
	"github.com/atotto/clipboard"
	table "github.com/calyptia/go-bubble-table"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	// "github.com/charmbracelet/lipgloss"
	kp "github.com/tobischo/gokeepasslib/v3"
)

type model struct {
	table           table.Model
	entries         []kp.Entry
	progress        progress.Model
	tooltipMessage  string
	textInput       textinput.Model
	page            string
	db              *kp.Database
	currentNewEntry newentry
}

type newentry struct {
	title    textInputWithLabel
	userName textInputWithLabel
	password textInputWithLabel
	url      textInputWithLabel
	cursor   int
}

type textInputWithLabel struct {
	label     string
	textinput textinput.Model
}

var (
	styleDoc = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			PaddingTop(2).
			PaddingLeft(4)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AFAFAF"))
)

func initialModel() model {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))

	if err != nil {
		fmt.Println("Error")
		w = 100
		h = 24
	}

	top, right, bottom, left := styleDoc.GetPadding()
	w = w - left - right
	h = h - top - bottom

	ti := textinput.New()
	ti.Placeholder = "Password"
	ti.Focus()
	ti.Width = w
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'

	return model{progress: progress.New(progress.WithDefaultGradient()), page: "password", textInput: ti}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func newTextInputWithLabel(label string, isfocused bool, w int, isPassword bool) textInputWithLabel {
	ti := textinput.New()
	ti.Placeholder = label
	ti.Width = w
	if isfocused {
		ti.Focus()
	}
	if isPassword {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '*'
	}
	return textInputWithLabel{
		label:     label,
		textinput: ti,
	}
}

func InitNewEntryView(m model) model {
	w, _, _ := term.GetSize(int(os.Stdout.Fd()))

	_, right, _, left := styleDoc.GetPadding()
	w = w - left - right

	m.currentNewEntry.title = newTextInputWithLabel("Title", true, w, false)
	m.currentNewEntry.userName = newTextInputWithLabel("User name", false, w, false)
	m.currentNewEntry.password = newTextInputWithLabel("Password", false, w, true)
	m.currentNewEntry.password.textinput.SetValue(GeneratePassword())
	m.currentNewEntry.url = newTextInputWithLabel("URL", false, w, false)

	return m
}

func GeneratePassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQR" +
		"STUVWXYZ0123456789!@#$%^&*()-_=+[{}];:'\",<.>/?"

	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}

func KeepassUpdate(m model, msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			clipboard.WriteAll(m.entries[m.table.Cursor()].GetPassword())
			time.AfterFunc(15*time.Second, func() { clipboard.WriteAll("") })
			cmd := m.progress.SetPercent(1.0)
			m.tooltipMessage = "Copied!"
			return m, tea.Batch(cmd, tickCmd())
		case "o":
			m.page = "newentry"
			m = InitNewEntryView(m)
		case "w":
			m, _ = saveDatabase(m)
		}
	case tickMsg:
		if m.progress.Percent() <= 0 {
			m.tooltipMessage = ""
			return m, nil
		}

		cmd := m.progress.DecrPercent(1.0 / 150)
		return m, tea.Batch(tickCmd(), cmd)
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)

	return m, cmd
}

func focusNewEntryInput(m model, direction int) model {
	switch m.currentNewEntry.cursor {
	case 0:
		m.currentNewEntry.title.textinput.Blur()
	case 1:
		m.currentNewEntry.userName.textinput.Blur()
	case 2:
		m.currentNewEntry.password.textinput.Blur()
	case 3:
		m.currentNewEntry.url.textinput.Blur()
	}

	m.currentNewEntry.cursor += direction
	if m.currentNewEntry.cursor < 0 {
		m.currentNewEntry.cursor += 4
	}
	m.currentNewEntry.cursor %= 4

	switch m.currentNewEntry.cursor {
	case 0:
		m.currentNewEntry.title.textinput.Focus()
	case 1:
		m.currentNewEntry.userName.textinput.Focus()
	case 2:
		m.currentNewEntry.password.textinput.Focus()
	case 3:
		m.currentNewEntry.url.textinput.Focus()
	}

	return m
}

func makeValue(key string, value string) kp.ValueData {
	return kp.ValueData{Key: key, Value: kp.V{Content: value}}
}

func createEntry(m model, title string, username string, password string, url string) model {
	entry := kp.NewEntry()
	entry.Values = append(entry.Values, makeValue("Title", title))
	entry.Values = append(entry.Values, makeValue("UserName", username))
	entry.Values = append(entry.Values, makeValue("Password", password))
	entry.Values = append(entry.Values, makeValue("URL", url))

	m.db.Content.Root.Groups[0].Entries = append(m.db.Content.Root.Groups[0].Entries, entry)
	return m
}

func NewentryUpdate(m model, msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyEsc.String():
			m.page = "keepass"
			cmd = m.progress.SetPercent(0)
		case "tab":
			m = focusNewEntryInput(m, 1)
		case "shift+tab":
			m = focusNewEntryInput(m, -1)
		case "enter":
			m = createEntry(
				m,
				m.currentNewEntry.title.textinput.Value(),
				m.currentNewEntry.userName.textinput.Value(),
				m.currentNewEntry.password.textinput.Value(),
				m.currentNewEntry.url.textinput.Value(),
			)
			m = InitTable(m)
			m.page = "keepass"
			cmd = m.progress.SetPercent(0)
		}
	}

	var cmd1, cmd2, cmd3, cmd4 tea.Cmd

	m.currentNewEntry.password.textinput, cmd1 = m.currentNewEntry.password.textinput.Update(msg)
	m.currentNewEntry.userName.textinput, cmd2 = m.currentNewEntry.userName.textinput.Update(msg)
	m.currentNewEntry.title.textinput, cmd3 = m.currentNewEntry.title.textinput.Update(msg)
	m.currentNewEntry.url.textinput, cmd4 = m.currentNewEntry.url.textinput.Update(msg)

	return m, tea.Batch(cmd, cmd1, cmd2, cmd3, cmd4)
}

func loadDatabase(m model, pass string) (model, error) {
	filename := os.Args[1]

	file, _ := os.Open(filename)
	m.db = kp.NewDatabase()
	m.db.Credentials = kp.NewPasswordCredentials(pass)
	err := kp.NewDecoder(file).Decode(m.db)

	if err != nil {
		return m, err
	}

	m.db.UnlockProtectedEntries()

	m = InitTable(m)

	m.page = "keepass"
	file.Close()

	return m, nil
}

func saveDatabase(m model) (model, error) {
	filename := os.Args[1]
	file, _ := os.OpenFile(filename, os.O_WRONLY, os.ModePerm)
	m.db.LockProtectedEntries()
	keepassEncoder := kp.NewEncoder(file)
	if err := keepassEncoder.Encode(m.db); err != nil {
		m.tooltipMessage = err.Error()
		return m, err
	}
	file.Close()
	m.tooltipMessage = "Saved!"

	return m, nil
}

func InitTable(m model) model {
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))

	top, right, bottom, left := styleDoc.GetPadding()
	w = w - left - right
	h = h - top - bottom - 2

	entries := m.db.Content.Root.Groups[0].Entries
	m.table = table.New([]string{"ID", "TITLE", "UserName", "PASS"}, w, h)
	rows := make([]table.Row, len(entries))
	for i, entry := range entries {
		rows[i] = table.SimpleRow{
			i,
			entry.GetTitle(),
			entry.Get("UserName").Value.Content,
			strings.Repeat("*", 12),
		}
	}

	m.table.SetRows(rows)
	m.table.KeyMap.Down = key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("down", "j"),
	)
	m.table.KeyMap.Up = key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("down", "k"),
	)
	m.entries = entries

	return m
}

func PasswordUpdate(m model, msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.tooltipMessage = ""
		switch msg.Type {
		case tea.KeyEnter:
			var err error
			m, err = loadDatabase(m, m.textInput.Value())
			if err != nil {
				m.tooltipMessage = err.Error()
				m.textInput.SetValue("")
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		top, right, bottom, left := styleDoc.GetPadding()
		m.table.SetSize(
			msg.Width-left-right,
			msg.Height-top-bottom-3,
		)
		m.progress.Width = msg.Width - left - right
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd

	switch {
	case m.page == "password":
		m, cmd = PasswordUpdate(m, msg)
	case m.page == "keepass":
		m, cmd = KeepassUpdate(m, msg)
	case m.page == "newentry":
		m, cmd = NewentryUpdate(m, msg)
	}

	return m, cmd
}

func keepassView(m model) string {
	return styleDoc.Render(
		lipgloss.JoinVertical(
			lipgloss.Position(0),
			m.table.View(),
			m.progress.View(),
			helpStyle.Render(m.tooltipMessage),
		),
	)
}

func newentryView(m model) string {
	return lipgloss.JoinVertical(
		lipgloss.Position(0),
		styleDoc.Render("New entry"),
		m.currentNewEntry.title.View(),
		m.currentNewEntry.userName.View(),
		m.currentNewEntry.password.View(),
		m.currentNewEntry.url.View(),
	)
}

func (t textInputWithLabel) View() string {
	return styleDoc.Render(
		fmt.Sprintf(
			"%s\n\n%s",
			t.label,
			t.textinput.View(),
		),
	)
}

func passwordView(m model) string {
	return styleDoc.Render(
		fmt.Sprintf(
			"Type in your password \n\n%s\n\n%s",
			m.textInput.View(),
			helpStyle.Render(m.tooltipMessage),
		),
	)
}

func (m model) View() string {
	if m.page == "keepass" {
		return keepassView(m)
	}
	if m.page == "password" {
		return passwordView(m)
	}
	if m.page == "newentry" {
		return newentryView(m)
	}

	return ""
}

func main() {
	rand.Seed(time.Now().UnixNano())
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

type tickMsg time.Time

type passwordMsg string

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
