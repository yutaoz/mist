package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder(), true, false).BorderForeground(lipgloss.Color("#FF2A6D"))

	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FF2A6D")).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render

	cfg, cfgerr = ReadCfg()
)

// create an item delegate with custom styles and behaviour for enter keypress
func newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	//lipgloss.NewStyle.Foreground(lipgloss.Color("sdf"))
	if cfgerr != nil {
		log.Fatal("Error reading config")
	}
	primaryColor := GetPrimaryColor(cfg)
	secondaryColor := GetSecondaryColor(cfg)
	dimSecondary := GetDimSecondary(cfg)

	d.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(primaryColor))
	d.Styles.NormalDesc = lipgloss.NewStyle().Foreground(lipgloss.Color(dimSecondary))
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(lipgloss.Color(primaryColor)).BorderLeftForeground(lipgloss.Color(primaryColor))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(lipgloss.Color(secondaryColor)).BorderLeftForeground(lipgloss.Color(primaryColor))

	// custom delegate update function to handle select behaviour
	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string
		var appid string
		if i, ok := m.SelectedItem().(item); ok {
			title = i.Title()
			appid = strconv.Itoa(i.AppId())
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.choose):
				oscmd := exec.Command("cmd", "/C", "start", "steam://rungameid/"+appid)
				err := oscmd.Run()
				if err != nil {
					return m.NewStatusMessage(statusMessageStyle("Error running game: " + err.Error()))
				}
				return m.NewStatusMessage(statusMessageStyle("You chose " + title + " | AppId: " + appid))
			}
		}

		return nil
	}

	help := []key.Binding{keys.choose, keys.console}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	fullh := []key.Binding{keys.cmd, keys.alpha, keys.recent, keys.ins, keys.all}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{fullh}
	}

	return d
}

func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.choose,
		d.console,
	}
}

func (d delegateKeyMap) FullHelp() []key.Binding {
	return []key.Binding{
		d.cmd,
		d.alpha,
		d.recent,
		d.ins,
		d.all,
	}
}

type delegateKeyMap struct {
	choose  key.Binding
	console key.Binding
	alpha   key.Binding
	recent  key.Binding
	ins     key.Binding
	cmd     key.Binding
	all     key.Binding
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		choose: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "choose"),
		),
		console: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "toggle commands"),
		),
		alpha: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "sort alphabetically"),
		),
		recent: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "sort by recent"),
		),
		ins: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "filter installed"),
		),
		cmd: key.NewBinding(
			key.WithKeys("COMMANDS:"),
			key.WithHelp("COMMANDS:", ""),
		),
		all: key.NewBinding(
			key.WithKeys("all"),
			key.WithHelp("all", "show all"),
		),
	}
}

type item struct {
	title       string
	description string
	appid       int
	installed   bool
	lastPlayed  int64
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) AppId() int          { return i.appid }
func (i item) LastPlayed() int64   { return i.lastPlayed }
func (i item) FilterValue() string { return i.title }

type model struct {
	list      list.Model
	allList   []list.Item
	textInput textinput.Model
	cmdFocus  bool
	command   string
	installed map[string]bool
}

// handles commands for the command line area
func (m *model) processCommand(cmd string) tea.Cmd {

	switch cmd {
	case "r":
		items := m.list.Items()
		sort.Slice(items, func(i, j int) bool {
			return items[i].(item).lastPlayed > items[j].(item).lastPlayed
		})
		m.list.ResetSelected()
		return nil

	case "i":
		items := m.list.Items()
		var installedItems []list.Item
		for _, itema := range items {
			if itema.(item).installed {
				installedItems = append(installedItems, itema)
			}
		}
		m.list.ResetSelected()
		return m.list.SetItems(installedItems)

	case "all":
		m.list.ResetSelected()
		return m.list.SetItems(m.allList)

	case "a":
		items := m.list.Items()
		sort.Slice(items, func(i, j int) bool {
			return items[i].(item).title < items[j].(item).title
		})
		m.list.ResetSelected()
		return nil
	}
	return nil
}

// creates a new model for the terminal, with all fields such as commands, list, etc
func newModel() model {

	var (
		delegateKeys = newDelegateKeyMap()
	)

	ti := textinput.New()
	ti.Placeholder = "commands"
	ti.CharLimit = 156
	ti.Width = 20

	// gets a list of installed games
	if cfgerr != nil {
		log.Fatal("Could not read cfg")
	}
	installedAppIds := GetInstalledGames(cfg)
	installed := make(map[string]bool) // map of installed app ids, use for lookup of installed game
	for _, appid := range installedAppIds {
		installed[appid] = true
	}

	allGames := GetAllGames(cfg)
	numItems := len(allGames)

	items := make([]list.Item, numItems)

	// set up items with necessary info, including display info
	for i, game := range allGames {
		fdate := "Never"
		if game.RtimeLastPlayed != 0 {
			date := time.Unix(game.RtimeLastPlayed, 0)
			fdate = date.Format("2006-01-02")
		}

		ibool := installed[strconv.Itoa(game.AppID)]
		insString := ""

		if ibool {
			insString = "INSTALLED"
		}

		items[i] = item{
			title:       game.Name,
			description: "Last Played: " + fdate + " - " + strconv.Itoa(game.PlaytimeForever/60) + "hrs " + insString,
			appid:       game.AppID,
			installed:   ibool,
			lastPlayed:  game.RtimeLastPlayed,
		}
	}

	delegate := newItemDelegate(delegateKeys)
	gameList := list.New(items, delegate, 0, 0)
	gameList.Title = "Games"
	gameList.Styles.Title = titleStyle

	allList := make([]list.Item, len(items))
	copy(allList, gameList.Items())

	return model{
		list:      gameList,
		allList:   allList,
		textInput: ti,
		cmdFocus:  false,
		command:   "",
		installed: installed,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// update and input handling
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			// switch msg.Type {
			// case tea.KeyEsc:
			// 	m.list.FilterInput.Blur()
			// case tea.KeyEnter:
			// 	//m.list.View().
			// default:
			// 	newListModel, cmd := m.list.Update(msg)
			// 	m.list = newListModel
			// 	cmds = append(cmds, cmd)
			// }
			break
		} else {
			switch msg.Type {
			case tea.KeyCtrlF: // toggle command view on keypress
				m.cmdFocus = !m.cmdFocus
				if m.cmdFocus {
					return m, m.textInput.Focus()
				} else {
					m.textInput.Blur()
				}

			}

			if m.cmdFocus {
				var cmd tea.Cmd

				switch msg.Type {
				case tea.KeyEnter: // only process command if enter and command focused
					m.command = m.textInput.Value()
					m.textInput.SetValue("")

					cmd = m.processCommand(m.command)

					m.cmdFocus = false
					m.textInput.Blur()
					cmds = append(cmds, cmd)
					return m, cmd
				}

				m.textInput, cmd = m.textInput.Update(msg)

				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
		}
	}

	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return appStyle.Render(m.list.View(), m.textInput.View())
}

func main() {
	p := tea.NewProgram(newModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
