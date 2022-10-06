// Package cmd handles CLI commands
package cmd

/*
Copyright © 2020 Regents of the University of California

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

import (
	"fmt"
	"log"
	"os"
	"time"
	rlog "tsm/log"

	tea "github.com/charmbracelet/bubbletea"
)

// // Status runs the status command
// func (c *cmdService) Status() error {

// 	rlog.NoticeMsg(fmt.Sprintf("running %s command on host: %s:%s\n", c.args[0], c.Host, c.Port))

// 	if _, _, err := c.queryForModel(); err != nil {
// 		return err
// 	}

// 	if err := c.snmpService.InitAndConnect(c.Host, c.Port, c.Community); err != nil {
// 		return err
// 	}
// 	defer c.snmpService.Close()

// 	initOids(c)

// 	ts, results, err := c.snmpService.QueryOids(&allOids)
// 	if err != nil {
// 		rlog.ErrMsg("error querying device %s:%s", c.Host, c.Port)
// 		return err
// 	}

// 	// result := c.displayStatusInfo(ts, &results)
// 	result := c.serializer.Format(ts, c.Host, c.Port, &results, c.TSMCfg)
// 	fmt.Println(result)

// 	return nil
// }

// package cmd

// import (
// 	"fmt"
// 	"log"
// 	"os"
// 	"time"
// 	rlog "tsm/log"

// 	tea "github.com/charmbracelet/bubbletea"
// )

type winfo struct {
	width  int
	height int
}

type model struct {
	timenow time.Time
	w       winfo
	cmdSvc  *cmdService
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func newModel() (*model, error) {

	return &model{timenow: time.Now().UTC()}, nil
}

func (state model) Init() tea.Cmd {

	state.timenow = time.Now().UTC()

	return tea.Batch(tick(), tea.EnterAltScreen)

}

func (state model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		state.w.width, state.w.height = msg.Width, msg.Height
	case tea.KeyMsg:

		// check for valid Quit keystrokes
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyCtrlBackslash:
			return state, tea.Quit
		case tea.KeyRunes:
			switch msg.String() {
			case "q", "Q":
				return state, tea.Quit
			}
		}

	case tickMsg:
		state.timenow = time.Now()
		return state, tick()

	}
	return state, nil

}

func (state model) View() string {

	vstr := "\n"

	ts, results, err := state.cmdSvc.snmpService.QueryOids(&allOids)
	if err != nil {
		rlog.ErrMsg("error querying device %s:%s", state.cmdSvc.Host, state.cmdSvc.Port)
		return err.Error()
	}

	vstr += state.cmdSvc.serializer.Format(ts, state.cmdSvc.Host, state.cmdSvc.Port, &results, state.cmdSvc.TSMCfg)

	return vstr

}

func setupConnection(c *cmdService) error {
	rlog.NoticeMsg(fmt.Sprintf("running %s command on host: %s:%s\n", c.args[0], c.Host, c.Port))

	if _, _, err := c.queryForModel(); err != nil {
		return err
	}

	if err := c.snmpService.InitAndConnect(c.Host, c.Port, c.Community); err != nil {
		return err
	}
	defer c.snmpService.Close()

	initOids(c)

	return nil
}

func initSNMP(c *cmdService) error {

	if err := setupConnection(c); err != nil {
		return err
	}

	rlog.NoticeMsg(fmt.Sprintf("running %s command on host: %s:%s\n", c.args[0], c.Host, c.Port))

	if _, _, err := c.queryForModel(); err != nil {
		return err
	}

	if err := c.snmpService.InitAndConnect(c.Host, c.Port, c.Community); err != nil {
		return err
	}

	return nil
}

func (c *cmdService) Status() error {

	if err := initSNMP(c); err != nil {
		log.Fatal(err)
	}
	defer c.snmpService.Close()

	initOids(c)

	state, err := newModel()
	if err != nil {
		fmt.Println(fmt.Sprintf("Error starting init command: %s\n", err))
		os.Exit(1)
	}
	state.cmdSvc = c
	// tea.NewProgram starts the Bubbletea framework which will render our
	// application using our state.
	if err := tea.NewProgram(state).Start(); err != nil {
		log.Fatal(err)
	}

	return nil
}
